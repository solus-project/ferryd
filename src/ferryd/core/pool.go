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

package core

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

	// DatabaseBucketDeltaSkip is the identifier for the pool's "failed delta" entries
	DatabaseBucketDeltaSkip = "deltaSkip"

	// PoolPathComponent is the storage directory for all of our main files
	PoolPathComponent = "pool"

	// PoolSchemaVersion is the current schema version for a PoolEntry
	PoolSchemaVersion = "1.0"
)

// DeltaInformation is included in pool entries if they're actually a delta
// package and not a normal package
type DeltaInformation struct {
	FromRelease int    // The source release for this delta
	FromID      string // ID for the source package
	ToRelease   int    // The target release for this delta
	ToID        string // ID for the target package
}

// A DeltaSkipEntry is used to record skipped deltas from some kind of generation
// failure
type DeltaSkipEntry struct {
	SchemaVersion string // Version used when this skip entry was created
	Name          string
	Delta         DeltaInformation
}

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
	Delta         *DeltaInformation     // May actually be nil if not a delta
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
	if _, err := tx.CreateBucketIfNotExists([]byte(DatabaseBucketPool)); err != nil {
		return err
	}
	_, err := tx.CreateBucketIfNotExists([]byte(DatabaseBucketDeltaSkip))
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
	dec := NewGobDecoderLight()
	if err := dec.DecodeType(v, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// Private method to re-put the entry into the DB
func (p *Pool) putEntry(tx *bolt.Tx, entry *PoolEntry) error {
	rootBucket := tx.Bucket([]byte(DatabaseBucketPool))
	code := NewGobEncoderLight()
	enc, err := code.EncodeType(entry)
	if err != nil {
		return err
	}
	return rootBucket.Put([]byte(entry.Name), enc)
}

// GetSkipEntry will return the delta-skip entry for the given ID
func (p *Pool) GetSkipEntry(tx *bolt.Tx, id string) (*DeltaSkipEntry, error) {
	rootBucket := tx.Bucket([]byte(DatabaseBucketDeltaSkip))
	v := rootBucket.Get([]byte(id))
	if v == nil {
		return nil, fmt.Errorf("Unknown delta entry: %s", id)
	}
	entry := &DeltaSkipEntry{}
	dec := NewGobDecoderLight()
	if err := dec.DecodeType(v, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// Private method to re-put the entry into the DB
func (p *Pool) putSkipEntry(tx *bolt.Tx, entry *DeltaSkipEntry) error {
	rootBucket := tx.Bucket([]byte(DatabaseBucketDeltaSkip))
	code := NewGobEncoderLight()
	enc, err := code.EncodeType(entry)
	if err != nil {
		return err
	}
	return rootBucket.Put([]byte(entry.Name), enc)
}

// GetPackagePoolPath Convenience function to grab the target for the given package
// within the current pool
func (p *Pool) GetPackagePoolPath(pkg *libeopkg.Package) string {
	pkgDir := filepath.Join(p.poolDir, pkg.Meta.Package.GetPathComponent())
	pkgTarget := filepath.Join(pkgDir, pkg.ID)
	return pkgTarget
}

// AddDelta will add a delta package to the pool if doesn't exist, otherwise
// it will increase the refcount for the package.
//
// This is a very loose wrapper around AddPackage, but will add some delta
// information too. Note that a delta package is still a package in its own
// right, its just installed and handled differently (lacking files, etc.)
func (p *Pool) AddDelta(tx *bolt.Tx, pkg *libeopkg.Package, mapping *DeltaInformation, copyDisk bool) (*PoolEntry, error) {
	// Check if this is just a simple case of bumping the refcount
	if entry, err := p.GetEntry(tx, pkg.ID); err == nil {
		entry.RefCount++
		return entry, p.putEntry(tx, entry)
	}

	// Validate these source/target packages *actually* exist
	sourceEntry, err := p.GetEntry(tx, mapping.FromID)
	if err != nil {
		return nil, err
	}
	targetEntry, err := p.GetEntry(tx, mapping.ToID)
	if err != nil {
		return nil, err
	}

	// Now set the rest of the metadata before storing
	mapping.ToRelease = targetEntry.Meta.GetRelease()
	mapping.FromRelease = sourceEntry.Meta.GetRelease()

	return p.addPackageInternal(tx, pkg, copyDisk, mapping)
}

// addPackageInternal used by both AddDelta and AddPackage for the main bulk of
// the work
func (p *Pool) addPackageInternal(tx *bolt.Tx, pkg *libeopkg.Package, copyDisk bool, delta *DeltaInformation) (*PoolEntry, error) {
	// Check if this is just a simple case of bumping the refcount
	if entry, err := p.GetEntry(tx, pkg.ID); err == nil {
		entry.RefCount++
		return entry, p.putEntry(tx, entry)
	}

	st, err := os.Stat(pkg.Path)
	if err != nil {
		return nil, err
	}

	// We have no refcount, so now we need to actually include this package
	// into the repositories.
	pkgTarget := p.GetPackagePoolPath(pkg)
	pkgDir := filepath.Dir(pkgTarget)
	if err := os.MkdirAll(pkgDir, 00755); err != nil {
		return nil, err
	}
	// Try to hard link the file into place
	if err := LinkOrCopyFile(pkg.Path, pkgTarget, copyDisk); err != nil {
		return nil, err
	}
	sha, err := FileSha1sum(pkg.Path)
	if err != nil {
		return nil, err
	}

	// Store immediately useful index bits here
	pkg.Meta.Package.PackageHash = sha
	pkg.Meta.Package.PackageSize = st.Size()
	pkg.Meta.Package.PackageURI = fmt.Sprintf("%s/%s", pkg.Meta.Package.GetPathComponent(), pkg.ID)

	entry := &PoolEntry{
		SchemaVersion: PoolSchemaVersion,
		Name:          pkg.ID,
		RefCount:      1,
		Meta:          &pkg.Meta.Package,
		Delta:         delta, // Might be nil, thats OK
	}

	if err := p.putEntry(tx, entry); err != nil {
		// Just clean out what we did because we can't write it into the DB
		// Error isn't important, really.
		os.Remove(pkgTarget)
		RemovePackageParents(pkgTarget)
		return nil, err
	}
	return entry, nil
}

// AddPackage will determine where the new eopkg goes, and whether we need
// to actually push it on disk, or simply bump the ref count. Any file
// passed to us is believed to be under our ownership now.
func (p *Pool) AddPackage(tx *bolt.Tx, pkg *libeopkg.Package, copy bool) (*PoolEntry, error) {
	return p.addPackageInternal(tx, pkg, copy, nil)
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

// MarkDeltaFailed will insert a record indicating that it is not possible
// to actually produce a given delta ID
func (p *Pool) MarkDeltaFailed(tx *bolt.Tx, id string, delta *DeltaInformation) error {
	// Already recorded? Skip again..
	if _, err := p.GetSkipEntry(tx, id); err == nil {
		return nil
	}

	skip := &DeltaSkipEntry{
		SchemaVersion: PoolSchemaVersion,
		Name:          id,
		Delta: DeltaInformation{
			FromID:      delta.FromID,
			ToID:        delta.ToID,
			FromRelease: delta.FromRelease,
			ToRelease:   delta.ToRelease,
		},
	}
	return p.putSkipEntry(tx, skip)
}

// GetDeltaFailed will determine if generation of this delta ID has actually
// failed in the past, skipping a potentially expensive delta examination.
func (p *Pool) GetDeltaFailed(tx *bolt.Tx, id string) bool {
	skip, err := p.GetSkipEntry(tx, id)
	if err == nil && skip != nil {
		return true
	}
	return false
}
