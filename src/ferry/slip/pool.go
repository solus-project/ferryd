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

package slip

import (
	"fmt"
	"github.com/boltdb/bolt"
	"libeopkg"
	"os"
	"path/filepath"
)

const (
	// DatabaseBucketPool is the identifier for the pool main bucket
	DatabaseBucketPool = "pool"

	// PoolPathComponent is the storage directory for all of our main files
	PoolPathComponent = "pool"

	// PoolSchemaVersion is the current schema version for a PoolEntry
	PoolSchemaVersion = "1.0"
)

// A PoolEntry is the main storage unit within ferryd.
// Each entry contains the full data for a given eopkg file, as well as the
// reference count.
//
// When the refcount hits 0, files are then purge from the pool and freed from
// disk. When adding a pool item to a repository, the ref count is increased,
// and the file is then hard-linked into place, saving on disk storage.
type PoolEntry struct {
	SchemaVersion string                // Version used when this pool entry was created
	Name          string                // Name&ID of the pool entry
	RefCount      uint64                // How many instances of this file exist right now
	Meta          *libeopkg.MetaPackage // The eopkg metadata
}

// A Pool is used to manage and deduplicate resources between multiple resources,
// and represents the real backing store for referenced eopkg files.
type Pool struct {
	poolDir    string // Storage area
	transcoder *GobTranscoder
}

// Init will create our initial working paths and DB bucket
func (p *Pool) Init(ctx *Context, tx *bolt.Tx) error {
	p.poolDir = filepath.Join(ctx.BaseDir, PoolPathComponent)
	p.transcoder = NewGobTranscoder()
	if err := os.MkdirAll(p.poolDir, 00755); err != nil {
		return err
	}
	_, err := tx.CreateBucketIfNotExists([]byte(DatabaseBucketPool))
	return err
}

// Close doesn't currently do anything
func (p *Pool) Close() {}

// GetEntry will return the package entry for the given ID
func (p *Pool) GetEntry(tx *bolt.Tx, id string) (*PoolEntry, error) {
	rootBucket := tx.Bucket([]byte(DatabaseBucketPool))
	v := rootBucket.Get([]byte(id))
	if v == nil {
		return nil, fmt.Errorf("Unknown pool entry: %s", id)
	}
	entry := &PoolEntry{}
	if err := p.transcoder.DecodeType(v, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// Private method to re-put the entry into the DB
func (p *Pool) putEntry(tx *bolt.Tx, entry *PoolEntry) error {
	rootBucket := tx.Bucket([]byte(DatabaseBucketPool))
	enc, err := p.transcoder.EncodeType(entry)
	if err != nil {
		return err
	}
	return rootBucket.Put([]byte(entry.Name), enc)
}

// AddPackage will determine where the new eopkg goes, and whether we need
// to actually push it on disk, or simply bump the ref count. Any file
// passed to us is believed to be under our ownership now.
func (p *Pool) AddPackage(tx *bolt.Tx, pkg *libeopkg.Package) error {
	// Check if this is just a simple case of bumping the refcount
	if entry, err := p.GetEntry(tx, pkg.ID); err != nil {
		entry.RefCount++
		return p.putEntry(tx, entry)
	}
	// We have no refcount, so now we need to actually include this package
	// into the repositories.
	pkgDir := filepath.Join(p.poolDir, pkg.Meta.Package.GetPathComponent())
	if err := os.MkdirAll(pkgDir, 00755); err != nil {
		return err
	}
	pkgTarget := filepath.Join(pkgDir, pkg.ID)
	// Try to hard link the file into place
	if err := LinkOrCopyFile(pkg.Path, pkgTarget); err != nil {
		return err
	}
	entry := &PoolEntry{
		SchemaVersion: PoolSchemaVersion,
		Name:          pkg.ID,
		RefCount:      1,
		Meta:          &pkg.Meta.Package,
	}
	if err := p.putEntry(tx, entry); err != nil {
		// Just clean out what we did because we can't write it into the DB
		// Error isn't important, really.
		os.Remove(pkgTarget)
		RemovePackageParents(pkgTarget)
		return err
	}
	return nil
}

// RefEntry will include the given eopkg if it doesn't yet exist, otherwise
// it will simply increase the ref count by 1.
func (p *Pool) RefEntry(tx *bolt.Tx, id string) error {
	entry, err := p.GetEntry(tx, id)
	if err != nil {
		return err
	}
	entry.RefCount++
	return p.putEntry(tx, entry)
}

// UnrefEntry will unref a given ID from the repository.
// Should the refcount hit 0, the package will then be removed from the pool
// storage.
func (p *Pool) UnrefEntry(tx *bolt.Tx, id string) error {
	entry, err := p.GetEntry(tx, id)
	if err != nil {
		return err
	}
	entry.RefCount--
	if entry.RefCount > 0 {
		return p.putEntry(tx, entry)
	}

	// RefCount is 0 so we now need to delete this entry
	pkgPath := filepath.Join(p.poolDir, entry.Meta.GetPathComponent(), id)
	if err := os.Remove(pkgPath); err != nil {
		return err
	}

	// TODO: Warn if unable to delete parents
	RemovePackageParents(pkgPath)

	// Now remove from DB
	b := tx.Bucket([]byte(DatabaseBucketPool))
	return b.Delete([]byte(id))
}
