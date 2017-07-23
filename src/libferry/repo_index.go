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

package libferry

import (
	"encoding/xml"
	"fmt"
	"github.com/boltdb/bolt"
	"libeopkg"
	"os"
	"path/filepath"
	"sort"
)

// initDistribution will look for the distribution.xml file which will define
// the all-important Obsoletes set
func (r *Repository) initDistribution() error {
	if r.dist != nil {
		return nil
	}

	dpath := filepath.Join(r.assetPath, "distribution.xml")
	if !PathExists(dpath) {
		fmt.Fprintf(os.Stderr, "WARNING: no distribution.xml defined\n")
		return nil
	}
	dist, err := libeopkg.NewDistribution(dpath)
	if err != nil {
		return err
	}
	r.dist = dist
	return nil
}

// emitDistribution is responsible for loading the distribution.xml file from
// the assets store and merging it into the final index
func (r *Repository) emitDistribution(encoder *xml.Encoder) error {
	elem := xml.StartElement{
		Name: xml.Name{
			Local: "Distribution",
		},
	}
	return encoder.EncodeElement(r.dist, elem)
}

// emitComponents is responsible for loading the components.xml file from
// the assets store and merging it into the final index
func (r *Repository) emitComponents(encoder *xml.Encoder) error {
	dpath := filepath.Join(r.assetPath, "components.xml")
	if !PathExists(dpath) {
		fmt.Fprintf(os.Stderr, "WARNING: no components.xml defined\n")
		return nil
	}
	comp, err := libeopkg.NewComponents(dpath)
	if err != nil {
		return err
	}

	elem := xml.StartElement{
		Name: xml.Name{
			Local: "Component",
		},
	}

	for i := range comp.Components {
		c := &comp.Components[i]
		if err := encoder.EncodeElement(c, elem); err != nil {
			return err
		}
	}
	// Now finalise the document
	return nil
}

// emitGroups is responsible for loading the groups.xml file from
// the assets store and merging it into the final index
func (r *Repository) emitGroups(encoder *xml.Encoder) error {
	dpath := filepath.Join(r.assetPath, "groups.xml")
	if !PathExists(dpath) {
		fmt.Fprintf(os.Stderr, "WARNING: no groups.xml defined\n")
		return nil
	}
	grp, err := libeopkg.NewGroups(dpath)
	if err != nil {
		return err
	}

	elem := xml.StartElement{
		Name: xml.Name{
			Local: "Group",
		},
	}

	for i := range grp.Groups {
		g := &grp.Groups[i]
		if err := encoder.EncodeElement(g, elem); err != nil {
			return err
		}
	}
	return nil
}

// emitIndex does the heavy lifting of writing to the given file descriptor,
// i.e. serialising the DB repo out to the index file
func (r *Repository) emitIndex(tx *bolt.Tx, pool *Pool, file *os.File) error {
	var pkgIds []string
	rootBucket := tx.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))

	c := rootBucket.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		code := NewGobDecoderLight()
		entry := RepoEntry{}
		if err := code.DecodeType(v, &entry); err != nil {
			return err
		}

		if r.dist != nil && r.dist.IsObsolete(entry.Name) {
			fmt.Fprintf(os.Stderr, " - Skipping obsolete package: %s\n", entry.Name)
			continue
		}

		pkgIds = append(pkgIds, entry.Published)
	}

	// Ensure we'll emit in a sane order
	sort.Strings(pkgIds)

	encoder := xml.NewEncoder(file)
	encoder.Indent("    ", "    ")
	// Wrap every output item as Package
	elem := xml.StartElement{
		Name: xml.Name{
			Local: "Package",
		},
	}

	// Ensure we have the start element
	if err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "PISI"}}); err != nil {
		return err
	}

	// Ensure distribution is at the head
	if err := r.emitDistribution(encoder); err != nil {
		return err
	}

	for _, pkg := range pkgIds {
		entry, err := pool.GetEntry(tx, pkg)
		if err != nil {
			return err
		}
		if err = encoder.EncodeElement(entry.Meta, elem); err != nil {
			return err
		}
	}

	// Stick in the components
	if err := r.emitComponents(encoder); err != nil {
		return err
	}

	// Stick in the groups ..
	if err := r.emitGroups(encoder); err != nil {
		return err
	}

	// Now finalise the document
	if err := encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "PISI"}}); err != nil {
		return err
	}

	return encoder.Flush()
}

// Index will attempt to write the eopkg index out to disk
// This only requires a read-only database view
func (r *Repository) Index(tx *bolt.Tx, pool *Pool) error {
	// If something goes wrong we need to remove our broken files
	var outPaths []string
	var finalPaths []string
	var errAbort error

	indexPath := filepath.Join(r.path, "eopkg-index.xml.new")
	indexPathFinal := filepath.Join(r.path, "eopkg-index.xml")
	outPaths = append(outPaths, indexPath)
	finalPaths = append(finalPaths, indexPathFinal)

	defer func() {
		if errAbort != nil {
			for _, p := range outPaths {
				fmt.Fprintf(os.Stdout, "Removing potentially corrupt file %s: %v\n", p, errAbort)
				os.Remove(p)
			}
		}
	}()

	if err := r.initDistribution(); err != nil {
		return err
	}

	// Create index file
	f, err := os.Create(indexPath)
	if err != nil {
		errAbort = err
		return errAbort
	}

	// Write the index file
	errAbort = r.emitIndex(tx, pool, f)
	f.Close()
	if errAbort != nil {
		return errAbort
	}

	// Sing the theme tune
	indexPathSha := filepath.Join(r.path, "eopkg-index.xml.sha1sum.new")
	indexPathShaFinal := filepath.Join(r.path, "eopkg-index.xml.sha1sum")
	outPaths = append(outPaths, indexPathSha)
	finalPaths = append(finalPaths, indexPathShaFinal)

	// Star in it
	if errAbort = WriteSha1sum(indexPath, indexPathSha); err != nil {
		return errAbort
	}

	// Write our XZ index out
	indexPathXz := filepath.Join(r.path, "eopkg-index.xml.xz.new")
	indexPathXzFinal := filepath.Join(r.path, "eopkg-index.xml.xz")
	outPaths = append(outPaths, indexPathXz)
	finalPaths = append(finalPaths, indexPathXzFinal)

	if errAbort = WriteXz(indexPath, indexPathXz); errAbort != nil {
		return errAbort
	}

	// Write sha1sum for our xz file
	indexPathXzSha := filepath.Join(r.path, "eopkg-index.xml.xz.sha1sum.new")
	indexPathXzShaFinal := filepath.Join(r.path, "eopkg-index.xml.xz.sha1sum")
	outPaths = append(outPaths, indexPathXzSha)
	finalPaths = append(finalPaths, indexPathXzShaFinal)

	if errAbort = WriteSha1sum(indexPathXz, indexPathXzSha); err != nil {
		return errAbort
	}

	for i, sourcePath := range outPaths {
		finalPath := finalPaths[i]

		if errAbort = AtomicRename(sourcePath, finalPath); errAbort != nil {
			return errAbort
		}
	}

	return nil
}
