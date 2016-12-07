//
// Copyright © 2016 Ikey Doherty <ikey@solus-project.com>
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

package manager

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/boltdb/bolt"
	"libeopkg"
	"os"
	"path/filepath"
)

const (
	// PoolDirectory is joined with our root path to form the full path
	// to our pool asset tree.
	PoolDirectory = "pool"
)

//
// A PoolEntry is the main storage area for the actual package information
// within binman.
// It is the place where package information is actually stored, the repos
// only have a linked relationship to the packages.
type PoolEntry struct {
	Name     string            // Basename of the package, including suffix
	Path     string            // Absolute path to the package file
	Metadata libeopkg.Metadata // Package information for this file

	RefCount int // Number of times duplicated
}

//
// A Pool is responsible for caching and inserting packages into the filesystem.
//
// The main goal is to facilitate deduplication, by storing .eopkg's in a single
// pool tree.
// When a pool asset is stored, the asset is then hard-linked into the repository's
// own tree.
//
type Pool struct {
	// private
	db      *bolt.DB
	poolDir string
}

// NewPool will return a new pool system. This is used primarily by Manager
// to assist in controlling the repositories.
func NewPool(root string, db *bolt.DB) *Pool {
	return &Pool{
		db:      db,
		poolDir: filepath.Join(root, PoolDirectory),
	}
}

// GetEntry will attempt to find the given entry in the pool bucket.
func (p *Pool) GetEntry(key string) (*PoolEntry, error) {
	entry := &PoolEntry{}
	err := p.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNamePool).Get([]byte(key))
		if len(b) == 0 {
			return ErrUnknownResource
		}
		buf := bytes.NewBuffer(b)
		dec := gob.NewDecoder(buf)
		// Decode the entry
		return dec.Decode(entry)
	})
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// storePackage will attempt to put the eopkg archive itself into the local
// cache.
func (p *Pool) storePackage(storagePath string, pkg *libeopkg.Package) error {
	if err := os.MkdirAll(storagePath, 00755); err != nil {
		return err
	}
	return os.Rename(pkg.Path, filepath.Join(storagePath, filepath.Base(pkg.Path)))
}

// removePackage will remove the file from the pool
func (p *Pool) removePackage(storagePath string) error {
	// TODO: Remove containing directories!
	return os.Remove(storagePath)
}

// RefPackage will potentially include a new .eopkg into the pool directory.
// If it already exists, then the refcount is increased
func (p *Pool) RefPackage(pkg *libeopkg.Package) (string, error) {
	baseName := filepath.Base(pkg.Path)
	key := []byte(baseName)
	var poolPath string

	// Potentially used twice
	buffer := &bytes.Buffer{}

	err := p.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNamePool)
		var entry PoolEntry
		var err error

		// Already have an entry? decode it
		if entBytes := b.Get(key); entBytes != nil {
			buffer.Write(entBytes)
			dec := gob.NewDecoder(buffer)
			if err := dec.Decode(&entry); err != nil {
				return err
			}
		}

		entry.Name = baseName
		entry.Metadata = *pkg.Meta
		// Bump refcount immediately
		entry.RefCount++
		storagePath := filepath.Join(p.poolDir, FormPackageBasePath(pkg.Meta))

		// We may now have to collect the package into the pool
		if entry.RefCount == 1 {
			fmt.Printf("Debug: Pooling fresh asset: %s\n", pkg.Path)
			if err = p.storePackage(storagePath, pkg); err != nil {
				return err
			}
		}
		fmt.Printf("Debug: Asset with ref count %d: %s\n", entry.RefCount, pkg.Path)

		// Relative path
		entry.Path = filepath.Join(storagePath, baseName)
		poolPath = entry.Path

		// Put the record back in place
		buffer.Reset()
		enc := gob.NewEncoder(buffer)

		if err = enc.Encode(&entry); err == nil {
			return b.Put(key, buffer.Bytes())
		}
		return err
	})
	if err != nil {
		return "", err
	}
	return poolPath, nil
}

// UnrefPackage will drop the refcount on a package if it exists, and if the
// refcount hits 0 it will be unpooled.
func (p *Pool) UnrefPackage(name string) error {
	nom := []byte(name)
	buffer := &bytes.Buffer{}

	err := p.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNamePool)
		// Can't unref that which does not exist
		blob := b.Get(nom)
		var entry PoolEntry
		if len(blob) == 0 {
			return ErrUnknownResource
		}
		buffer.Write(blob)

		// Decode the existing gob entry
		dec := gob.NewDecoder(buffer)
		if err := dec.Decode(&entry); err != nil {
			return err
		}

		// Drop refcount
		entry.RefCount--

		// Drop from filesystem now before unref'ing, in case it fails
		if entry.RefCount == 0 {
			fmt.Printf("Debug: Dropping asset from pool: %s\n", entry.Path)
			if err := p.removePackage(entry.Path); err != nil {
				return err
			}
			return b.Delete(nom)
		}
		fmt.Printf("Debug: Asset with ref count %d: %s\n", entry.RefCount, entry.Path)

		// Store the entry back in
		buffer.Reset()
		enc := gob.NewEncoder(buffer)
		if err := enc.Encode(&entry); err != nil {
			return err
		}

		return b.Put(nom, buffer.Bytes())
	})
	return err
}
