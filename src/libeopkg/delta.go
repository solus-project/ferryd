//
// Copyright © 2017 Ikey Doherty <ikey@solus-project.com>
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
	"github.com/solus-project/xzed"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// DeltaProducer is responsible for taking two eopkg packages and spitting out
// a delta package for them, containing only the new files.
type DeltaProducer struct {
	old     *ArchiveReader
	new     *ArchiveReader
	diffMap map[string]int
}

var (
	// ErrMismatchedDelta is returned when the input packages should never be delta'd,
	// i.e. they're unrelated
	ErrMismatchedDelta = errors.New("Delta is not possible between the input packages")
)

// NewDeltaProducer will return a new delta producer for the given input packages
// It is very important that the old and new packages are in the correct order!
func NewDeltaProducer(pkgOld string, pkgNew string) (*DeltaProducer, error) {
	var err error
	ret := &DeltaProducer{
		diffMap: make(map[string]int),
	}
	defer func() {
		if err != nil {
			ret.Close()
		}
	}()
	ret.old, err = NewArchiveReaderFromFilename(pkgOld)
	if err != nil {
		return nil, err
	}

	ret.new, err = NewArchiveReaderFromFilename(pkgNew)
	if err != nil {
		return nil, err
	}

	if !IsDeltaPossible(&ret.old.pkg.Meta.Package, &ret.new.pkg.Meta.Package) {
		return nil, ErrMismatchedDelta
	}

	return ret, nil
}

// Close the DeltaProducer
func (d *DeltaProducer) Close() {
	if d.old != nil {
		d.old.Close()
		d.old = nil
	}
	if d.new != nil {
		d.new.Close()
		d.new = nil
	}
}

// filesToMap is a helper that will let us uniquely index hash to file-set
func (d *DeltaProducer) filesToMap(r *ArchiveReader) (ret map[string][]*File) {
	ret = make(map[string][]*File)
	for _, f := range r.pkg.Files.File {
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
		installXZ *os.File
		xzw       *xzed.Writer
		tw        *tar.Writer
		err       error
		filename  string
	)

	hashOldFiles := d.filesToMap(d.old)
	hashNewFiles := d.filesToMap(d.new)

	// Note this is very simple and works just like the existing eopkg functionality
	// which is purely hash-diff based. eopkg will look for relocations on applying
	// the update so that files get "reused"
	for h, s := range hashNewFiles {
		if _, ok := hashOldFiles[h]; ok {
			continue
		}
		for _, p := range s {
			d.diffMap[p.Path] = 1
		}
	}

	// Make sure we clean up properly!
	defer func() {
		if tw != nil {
			tw.Close()
		}
		if xzw != nil {
			xzw.Close()
		}
	}()

	// Open up an XZ-wrapped tarfile
	installXZ, err = ioutil.TempFile("", "ferryd-installtarxz")
	if err != nil {
		return "", err
	}
	filename = installXZ.Name()
	xzw, err = xzed.NewWriter(installXZ)
	if err != nil {
		return filename, err
	}
	tw = tar.NewWriter(xzw)

	if err = d.copyInstallPartial(tw); err != nil {
		return filename, err
	}

	tw.Flush()
	tw.Close()
	xzw.Close()

	return filename, nil
}

// copyInstallPartial will iterate over the contents of the existing install.tar.xz
// for the new package, and only include the files that aren't hash-matched in the
// old files.xml
func (d *DeltaProducer) copyInstallPartial(tw *tar.Writer) error {
	for {
		header, err := d.old.tarfile.Next()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return err
		}
		// Skip anything not in the diff map
		if _, ok := d.diffMap[header.Name]; !ok {
			continue
		}
		if err = tw.WriteHeader(header); err != nil {
			return err
		}
		if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
			if _, err = io.Copy(tw, d.old.tarfile); err != nil {
				return err
			}
		}
	}
	return tw.Flush()
}

// copyZipPartial will iterate the central zip directory and skip only the
// install.tar.xz files, whilst copying everything else into the new zip
func (d *DeltaProducer) copyZipPartial(zw *zip.Writer) error {
	for _, zipFile := range d.new.pkg.zipFile.File {
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
	out, err := ioutil.TempFile("", "ferryd-delta-eopkg")
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
