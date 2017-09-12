//
// Copyright © 2017 Solus Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package libeopkg

import (
	"archive/tar"
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DeltaProducer is responsible for taking two eopkg packages and spitting out
// a delta package for them, containing only the new files.
type DeltaProducer struct {
	old     *Package
	new     *Package
	baseDir string
	diffMap map[string]int
}

var (
	// ErrMismatchedDelta is returned when the input packages should never be delta'd,
	// i.e. they're unrelated
	ErrMismatchedDelta = errors.New("Delta is not possible between the input packages")

	// ErrDeltaPointless is returned when it is quite literally pointless to bother making
	// a delta package, due to the packages having exactly the same content.
	ErrDeltaPointless = errors.New("File set is the same, no point in creating delta")
)

// NewDeltaProducer will return a new delta producer for the given input packages
// It is very important that the old and new packages are in the correct order!
func NewDeltaProducer(baseDir string, pkgOld string, pkgNew string) (*DeltaProducer, error) {
	var err error
	ret := &DeltaProducer{
		diffMap: make(map[string]int),
	}
	defer func() {
		if err != nil {
			ret.Close()
		}
	}()
	ret.old, err = Open(pkgOld)
	if err != nil {
		return nil, err
	}

	if err = ret.old.ReadAll(); err != nil {
		return nil, err
	}

	ret.new, err = Open(pkgNew)
	if err != nil {
		return nil, err
	}

	if err = ret.new.ReadAll(); err != nil {
		return nil, err
	}

	if !IsDeltaPossible(&ret.old.Meta.Package, &ret.new.Meta.Package) {
		return nil, ErrMismatchedDelta
	}

	// Form a unique directory entry
	dirName := fmt.Sprintf("%s-%s-%s-%d-%d",
		ret.new.Meta.Package.Name,
		ret.new.Meta.Package.GetVersion(),
		ret.new.Meta.Package.Architecture,
		ret.old.Meta.Package.GetRelease(),
		ret.new.Meta.Package.GetRelease())

	ret.baseDir = filepath.Join(baseDir, dirName)

	// Make sure base directory actually exists
	if err = os.MkdirAll(ret.baseDir, 00755); err != nil {
		return nil, err
	}

	return ret, nil
}

// Close the DeltaProducer
func (d *DeltaProducer) Close() error {
	if d.old != nil {
		d.old.Close()
		d.old = nil
	}
	if d.new != nil {
		d.new.Close()
		d.new = nil
	}
	// Ensure we always nuke the work directory we used
	if d.baseDir != "" {
		return os.RemoveAll(d.baseDir)
	}
	return nil
}

// filesToMap is a helper that will let us uniquely index hash to file-set
func (d *DeltaProducer) filesToMap(p *Package) (ret map[string][]*File) {
	ret = make(map[string][]*File)
	for _, f := range p.Files.File {
		if _, ok := ret[f.Hash]; !ok {
			ret[f.Hash] = ([]*File{f})
		} else {
			ret[f.Hash] = append(ret[f.Hash], f)
		}
	}
	return ret
}

// The bulk of the work will happen here as we attempt to produce the main
// install.tar.xz tarball which will be used in the final .eopkg file
func (d *DeltaProducer) produceInstallBall() (string, error) {
	var (
		tw       *tar.Writer
		err      error
		filename string
	)

	hashOldFiles := d.filesToMap(d.old)
	hashNewFiles := d.filesToMap(d.new)

	// Note this is very simple and works just like the existing eopkg functionality
	// which is purely hash-diff based. eopkg will look for relocations on applying
	// the update so that files get "reused"
	//
	// Special Note: Key "" denotes a directory which is basically empty, so we must
	// always include these in the delta
	for h, s := range hashNewFiles {
		if _, ok := hashOldFiles[h]; ok && h != "" {
			continue
		}
		for _, p := range s {
			d.diffMap[strings.TrimSuffix(p.Path, "/")] = 1
		}
	}

	// All the same files
	if len(d.diffMap) == len(d.new.Files.File) {
		return "", ErrDeltaPointless
	}

	// No install.tar.xz to write as we have no different files
	if len(d.diffMap) == 0 {
		return "", nil
	}

	// Make sure we clean up properly!
	defer func() {
		if tw != nil {
			tw.Close()
		}
	}()

	// Open output file to write our tarfile.
	installTar := filepath.Join(d.baseDir, "delta-eopkg.install.tar")
	outF, err := os.Create(installTar)
	if err != nil {
		return "", err
	}
	tw = tar.NewWriter(outF)

	if err = d.copyInstallPartial(tw); err != nil {
		return filename, err
	}

	tw.Flush()
	tw.Close()

	if err = XzFile(installTar, false); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.xz", installTar), nil
}

// copyInstallPartial will iterate over the contents of the existing install.tar.xz
// for the new package, and only include the files that aren't hash-matched in the
// old files.xml
func (d *DeltaProducer) copyInstallPartial(tw *tar.Writer) error {

	// Ensure we have tarball ready for use
	if err := d.new.ExtractTarball(d.baseDir); err != nil {
		return err
	}

	inpFile := filepath.Join(d.baseDir, "install.tar")
	fi, err := os.Open(inpFile)
	if err != nil {
		return err
	}
	defer fi.Close()
	tarfile := tar.NewReader(fi)

	for {
		header, err := tarfile.Next()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return err
		}

		// Ensure that we compare things in the same way
		checkName := strings.TrimSuffix(header.Name, "/")

		// Skip anything not in the diff map
		if _, ok := d.diffMap[checkName]; !ok {
			continue
		}

		if err = tw.WriteHeader(header); err != nil {
			return err
		}
		if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
			if _, err = io.Copy(tw, tarfile); err != nil {
				return err
			}
		}
	}
	return tw.Flush()
}

// copyZipPartial will iterate the central zip directory and skip only the
// install.tar.xz files, whilst copying everything else into the new zip
func (d *DeltaProducer) copyZipPartial(zw *zip.Writer) error {
	for _, zipFile := range d.new.zipFile.File {
		// Skip any kind of install.tar internally
		if strings.HasPrefix(zipFile.Name, "install.tar") {
			continue
		}

		iop, err := zipFile.Open()
		if err != nil {
			return err
		}
		defer iop.Close()

		zwh := &zip.FileHeader{}
		*zwh = zipFile.FileHeader

		// Copy the File member across (it implements FileHeader)
		w, err := zw.CreateHeader(zwh)
		if err != nil {
			return err
		}
		if _, err = io.Copy(w, iop); err != nil {
			return err
		}
		iop.Close() // be really sure we close it..
	}
	return nil
}

func (d *DeltaProducer) pushInstallBall(zipFile *zip.Writer, xzFileName string) error {
	// No install.tar.xz to write as we have no different files
	if len(d.diffMap) == 0 {
		return nil
	}
	f, err := os.Open(xzFileName)
	if err != nil {
		return err
	}
	// Stat for header
	fst, err := f.Stat()
	if err != nil {
		return err
	}
	defer f.Close()

	fh, err := zip.FileInfoHeader(fst)
	if err != nil {
		return err
	}

	// Ensure it's always the right name.
	fh.Name = "install.tar.xz"

	w, err := zipFile.CreateHeader(fh)
	if err != nil {
		return err
	}

	if _, err = io.Copy(w, f); err != nil {
		return err
	}

	return zipFile.Flush()
}

// Commit will attempt to produce a delta between the 2 eopkg files
// This will be performed in temporary storage so must then be copied into
// the final resting location, and unlinked, before it can be used.
func (d *DeltaProducer) Commit() (string, error) {
	xzFileName, err := d.produceInstallBall()
	var zipFileName string
	defer func() {
		if xzFileName != "" {
			os.Remove(xzFileName)
		}
		// If we're successful, we don't delete this
		if zipFileName != "" {
			os.Remove(zipFileName)
		}
	}()
	if err != nil {
		return "", err
	}
	fpath := filepath.Join(d.baseDir, ComputeDeltaName(&d.old.Meta.Package, &d.new.Meta.Package))
	out, err := os.Create(fpath)
	if err != nil {
		return "", err
	}
	zipFileName = out.Name()
	zip := zip.NewWriter(out)
	if err = d.copyZipPartial(zip); err != nil {
		return "", err
	}

	// Now copy our install.tar.xz into the mix
	if err = d.pushInstallBall(zip, xzFileName); err != nil {
		return "", err
	}

	ret := "" + zipFileName

	if err := zip.Close(); err != nil {
		return "", err
	}

	zipFileName = ""
	return ret, nil
}
