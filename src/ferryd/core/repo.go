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
	"libdb"
	"libeopkg"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	// RepoPathComponent is the base for all repository directories
	RepoPathComponent = "repo"

	// AssetPathComponent is where we'll find extra files like distribution.xml
	AssetPathComponent = "assets"

	// DeltaPathComponent is a temporary tree for creating delta packages
	DeltaPathComponent = "deltaBuilds"

	// DeltaStagePathComponent is where we put temporary deltas until merged
	DeltaStagePathComponent = "deltaStaging"

	// DatabaseBucketRepo is the name for the main repo toplevel bucket
	DatabaseBucketRepo = "repo"

	// DatabaseBucketPackage is the path to the subbucket within a repo bucket
	DatabaseBucketPackage = "package"

	// RepoSchemaVersion is the current schema version for a RepoEntry
	RepoSchemaVersion = "1.0"
)

// The RepositoryManager maintains all repos within ferryd which are in
// turn linked to the main pool
type RepositoryManager struct {
	repoBase       string
	assetBase      string
	deltaBase      string
	deltaStageBase string

	repoLock *sync.Mutex

	repos map[string]*Repository // Cache all repositories.
}

// A Repository is a simplistic representation of a exported repository
// within ferryd
type Repository struct {
	ID             string                 // Name of this repository (unique)
	path           string                 // Where this is on disk
	assetPath      string                 // Where our assets are stored on disk
	deltaPath      string                 // Where we'll produce deltas
	deltaStagePath string                 // Where we'll stage final deltas
	dist           *libeopkg.Distribution // Distribution

	insertMut *sync.Mutex // Prevent parallel inserts
	indexMut  *sync.Mutex // Indexing requires a special, separate lock
}

// RepoEntry is the basic repository storage unit, and details what packages
// are exported in the index.
type RepoEntry struct {
	SchemaVersion string   // Version used when this repo entry was created
	Name          string   // Base package name
	Available     []string // The available packages for this package name (eopkg IDs)
	Published     string   // The "tip" version of this package (eopkg ID)
	Deltas        []string // Delta packages known for this package.
}

// Init will create our initial working paths and DB bucket
func (r *RepositoryManager) Init(ctx *Context, db libdb.Database) error {
	r.repoBase = filepath.Join(ctx.BaseDir, RepoPathComponent)
	r.assetBase = filepath.Join(ctx.BaseDir, AssetPathComponent)
	r.deltaBase = filepath.Join(ctx.BaseDir, DeltaPathComponent)
	r.deltaStageBase = filepath.Join(ctx.BaseDir, DeltaStagePathComponent)
	r.repoLock = &sync.Mutex{}
	r.repos = make(map[string]*Repository)

	paths := []string{
		r.repoBase,
		r.assetBase,
		r.deltaBase,
	}
	// Ensure we have all paths
	for _, p := range paths {
		if err := os.MkdirAll(p, 00755); err != nil {
			return err
		}
	}
	return nil
}

// Close doesn't currently do anything
func (r *RepositoryManager) Close() {}

// bakeRepo hands the internal duped code between GetRepo/CreateRepo to ensure
// they're always fully formed.
//
// This ensures the first time we GetRepo on an existing repo, we ensure that
// we actually have all support paths too.
func (r *RepositoryManager) bakeRepo(id string) (*Repository, error) {
	repository := &Repository{
		ID:             id,
		path:           filepath.Join(r.repoBase, id),
		assetPath:      filepath.Join(r.assetBase, id),
		deltaPath:      filepath.Join(r.deltaBase, id),
		deltaStagePath: filepath.Join(r.deltaStageBase, id),
		indexMut:       &sync.Mutex{},
		insertMut:      &sync.Mutex{},
	}

	paths := []string{
		repository.path,
		repository.assetPath,
		repository.deltaPath,
		repository.deltaStagePath,
	}

	// Create all required paths
	for _, p := range paths {
		if PathExists(p) {
			continue
		}
		if err := os.MkdirAll(p, 00755); err != nil {
			return nil, err
		}
	}

	return repository, nil
}

// GetRepos will return a copy of the repositores in our database
func (r *RepositoryManager) GetRepos(db libdb.Database) ([]*Repository, error) {
	var ret []*Repository
	err := db.Bucket([]byte(DatabaseBucketRepo)).View(func(db libdb.ReadOnlyView) error {
		return db.ForEach(func(key, value []byte) error {
			var repo Repository
			if err := db.Decode(value, &repo); err != nil {
				return err
			}
			ret = append(ret, &repo)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// GetRepo will attempt to get the named repo if it exists, otherwise
// return an error. This is a transactional helper to make the API simpler
func (r *RepositoryManager) GetRepo(db libdb.Database, id string) (*Repository, error) {
	// Cache each repository.
	if repo, ok := r.repos[id]; ok {
		return repo, nil
	}

	var rTmp Repository
	rootBucket := db.Bucket([]byte(DatabaseBucketRepo))
	if err := rootBucket.GetObject([]byte(id), &rTmp); err != nil {
		return nil, fmt.Errorf("The specified repository '%s' does not exist", id)
	}

	repository, err := r.bakeRepo(id)
	if err != nil {
		return nil, err
	}

	// Cache this guy for later
	r.repos[id] = repository

	return repository, nil
}

// CreateRepo will create a new repository (bucket) within the top level
// repo bucket.
func (r *RepositoryManager) CreateRepo(db libdb.Database, id string) (*Repository, error) {
	r.repoLock.Lock()
	defer r.repoLock.Unlock()

	if _, err := r.GetRepo(db, id); err == nil {
		return nil, fmt.Errorf("The specified repository '%s' already exists", id)
	}

	// Create the main sub-bucket for this repo
	rootBucket := db.Bucket([]byte(DatabaseBucketRepo))
	repo := Repository{
		ID: id,
	}

	if err := rootBucket.PutObject([]byte(id), &repo); err != nil {
		return nil, err
	}

	repository, err := r.bakeRepo(id)
	if err != nil {
		return nil, err
	}

	// Cache this guy for later
	r.repos[id] = repository

	return repository, nil
}

// DeleteRepo is not yet implemented
func (r *RepositoryManager) DeleteRepo(db libdb.Database, pool *Pool, id string) error {
	r.repoLock.Lock()
	defer r.repoLock.Unlock()

	repo, err := r.GetRepo(db, id)
	if err != nil {
		return fmt.Errorf("The specified repository '%s' does not exist", id)
	}

	// TODO: lock this repository from accepting inclusions ..
	delete(r.repos, id)

	// Let's iterate over every one of our packages here and start up an unref
	// cycle
	err = db.Update(func(db libdb.Database) error {
		repoBucket := db.Bucket([]byte(DatabaseBucketRepo))
		rootBucket := repoBucket.Bucket([]byte(repo.ID)).Bucket([]byte(DatabaseBucketPackage))

		err := rootBucket.ForEach(func(k, v []byte) error {
			// Grab each repo entry
			entry := RepoEntry{}
			if err := rootBucket.Decode(v, &entry); err != nil {
				return err
			}

			// First up, find all the packages to unref
			for _, id := range entry.Available {
				if err := repo.removePackageInternal(db, pool, id); err != nil {
					return err
				}
			}

			// Next up, find all the deltas to unref
			for _, id := range entry.Deltas {
				if err := repo.removeDeltaInternal(db, pool, id); err != nil {
					return err
				}
			}

			// Remove all IDs for this package, now we must remove this entry
			// Note this isn't applied within the foreach.
			return rootBucket.DeleteObject([]byte(entry.Name))
		})
		if err != nil {
			return err
		}
		// Now remove the repository object itself
		return repoBucket.DeleteObject([]byte(repo.ID))
	})

	if err != nil {
		return err
	}

	deletionPaths := []string{
		repo.path,
		repo.assetPath,
		repo.deltaPath,
		repo.deltaStagePath,
	}

	// Clean up the repo paths.
	// TODO: Make non fatal, just attempt what we can.

	for _, p := range deletionPaths {
		if !PathExists(p) {
			continue
		}
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}

	return nil
}

// GetEntry will return the package entry for the given ID
func (r *Repository) GetEntry(db libdb.Database, id string) (*RepoEntry, error) {
	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))
	entry := &RepoEntry{}
	if err := rootBucket.GetObject([]byte(id), entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// Private method to re-put the entry into the DB
//
// TODO: Consider write protection
func (r *Repository) putEntry(db libdb.Database, entry *RepoEntry) error {
	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))
	return rootBucket.PutObject([]byte(entry.Name), entry)
}

// RefDelta will take the existing delta from the pool and insert it into our own repository
func (r *Repository) RefDelta(db libdb.Database, pool *Pool, deltaID string) error {
	r.insertMut.Lock()
	defer r.insertMut.Unlock()

	// Ensure we REALLY have the delta.
	poolEntry, err := pool.GetEntry(db, deltaID)
	if err != nil {
		return err
	}

	// Now make sure we actually have the local entry
	entry, err := r.GetEntry(db, poolEntry.Meta.Name)
	if err != nil {
		return err
	}

	// Extract our relevant paths
	localPath := pool.GetMetaPoolPath(deltaID, poolEntry.Meta)
	targetDir := filepath.Join(r.path, poolEntry.Meta.GetPathComponent())
	targetPath := filepath.Join(targetDir, deltaID)

	// Check we don't know about this delta already
	for _, id := range entry.Deltas {
		if id == deltaID {
			fmt.Printf("Skipping already included delta %s\n", id)
			return nil
		}
	}

	// Insert this deltas ID to this package map
	entry.Deltas = append(entry.Deltas, deltaID)
	sort.Strings(entry.Deltas)

	// Construct root dirs
	if err := os.MkdirAll(targetDir, 00755); err != nil {
		return err
	}

	// Grab the pool reference for this package
	if err := pool.RefEntry(db, deltaID); err != nil {
		return err
	}

	// Ensure the eopkg file is linked inside our own tree
	if err := LinkOrCopyFile(localPath, targetPath, false); err != nil {
		return err
	}

	return r.putEntry(db, entry)
}

// AddDelta will first open and read the .delta.eopkg, before passing it back off to AddLocalDelta
func (r *Repository) AddDelta(db libdb.Database, pool *Pool, filename string, mapping *DeltaInformation) error {
	pkg, err := libeopkg.Open(filename)
	if err != nil {
		return err
	}

	defer pkg.Close()
	if err = pkg.ReadMetadata(); err != nil {
		return err
	}

	return r.AddLocalDelta(db, pool, pkg, mapping)
}

// AddLocalDelta will attempt to add the delta to this repository, if possible
// All ref'd deltas are retained, but not necessarily emitted unless they're
// valid for the from-to relationship.
func (r *Repository) AddLocalDelta(db libdb.Database, pool *Pool, pkg *libeopkg.Package, mapping *DeltaInformation) error {
	r.insertMut.Lock()
	defer r.insertMut.Unlock()

	// Find our local package entry for the delta package first
	entry, err := r.GetEntry(db, pkg.Meta.Package.Name)
	if err != nil {
		return err
	}

	pkgDir := filepath.Join(r.path, pkg.Meta.Package.GetPathComponent())
	pkgTarget := filepath.Join(pkgDir, pkg.ID)

	// Check we don't know about this delta already
	for _, id := range entry.Deltas {
		if id == pkg.ID {
			fmt.Printf("Skipping already included delta %s\n", id)
			return nil
		}
	}

	// Insert this deltas ID to this package map
	entry.Deltas = append(entry.Deltas, pkg.ID)
	sort.Strings(entry.Deltas)

	// Construct root dirs
	if err := os.MkdirAll(pkgDir, 00755); err != nil {
		return err
	}

	// Grab the pool reference for this package
	if _, err = pool.AddDelta(db, pkg, mapping, false); err != nil {
		return err
	}

	// Ensure the eopkg file is linked inside our own tree
	source := pool.GetPackagePoolPath(pkg)
	if err = LinkOrCopyFile(source, pkgTarget, false); err != nil {
		return err
	}

	return r.putEntry(db, entry)
}

// Internal helper to remove packages
func (r *Repository) removePackageInternal(db libdb.Database, pool *Pool, id string) error {
	r.insertMut.Lock()
	defer r.insertMut.Unlock()

	poolEntry, err := pool.GetEntry(db, id)
	if err != nil {
		return nil
	}

	pkgDir := filepath.Join(r.path, poolEntry.Meta.GetPathComponent())
	pkgTarget := filepath.Join(pkgDir, poolEntry.Name)

	// TODO: Consider making non-fatal..
	if err = os.Remove(pkgTarget); err != nil {
		return err
	}

	// TODO: This likely shouldn't be fatal either
	if err = RemovePackageParents(pkgTarget); err != nil {
		return err
	}

	// Tell the pool we no longer need this guy
	return pool.UnrefEntry(db, id)
}

// removeDeltaInternal has the same job as removePackageInternal, but in future should
// be extended to remove the skip records
func (r *Repository) removeDeltaInternal(db libdb.Database, pool *Pool, id string) error {
	return r.removePackageInternal(db, pool, id)
}

// UnrefPackage will remove a package from our storage, and potentially remove the
// entire RepoEntry for the package if none are left.
//
// Additionally, we'll locate stray deltas which lead either TO or FROM the given
// package as they'll now be useless to anyone.
func (r *Repository) UnrefPackage(db libdb.Database, pool *Pool, pkgID string) error {
	newHighest := 0
	var newHighestID string

	// Require a pool entry to remove it
	poolEntry, err := pool.GetEntry(db, pkgID)
	if err != nil {
		return err
	}

	// Now examine our own local entry
	entry, err := r.GetEntry(db, poolEntry.Meta.Name)
	if err != nil {
		return err
	}

	// Try to remove it from disk first
	if err := r.removePackageInternal(db, pool, pkgID); err != nil {
		return err
	}

	// Deltas remaining after removals
	var remainDeltas []string

	// Check out all the deltas
	for _, deltaID := range entry.Deltas {
		pkgDelta, err := pool.GetEntry(db, deltaID)
		if err != nil {
			return err
		}

		// We found a delta that is referencing us, we must garbage collect it now
		if pkgDelta.Delta.FromID == pkgID || pkgDelta.Delta.ToID == pkgID {
			if err := r.removeDeltaInternal(db, pool, pkgDelta.Name); err != nil {
				return err
			}
		} else {
			remainDeltas = append(remainDeltas, pkgDelta.Name)
		}
	}

	// These are the deltas left
	entry.Deltas = remainDeltas
	sort.Strings(entry.Deltas)

	// Filter our ID from the available set
	var remainAvailable []string
	for _, id := range entry.Available {
		if id == pkgID {
			continue
		}
		availEntry, err := pool.GetEntry(db, id)
		if err != nil {
			return err
		}
		// Learn highest relno here, it's going to be the published ID
		curRel := availEntry.Meta.GetRelease()
		if curRel > newHighest {
			newHighest = curRel
			newHighestID = id
		}
		remainAvailable = append(remainAvailable, id)
	}

	entry.Available = remainAvailable
	sort.Strings(entry.Available)
	// Assign the new Published link
	entry.Published = newHighestID

	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))

	// Is this package set now "empty"? Then remove it from our indexes
	if len(entry.Available) < 1 {
		return rootBucket.DeleteObject([]byte(entry.Name))
	}

	// Stuff it back into the DB with the modified bits in place.
	return r.putEntry(db, entry)
}

// RefPackage will dupe a package from the pool into our own storage
func (r *Repository) RefPackage(db libdb.Database, pool *Pool, pkgID string) error {
	r.insertMut.Lock()
	defer r.insertMut.Unlock()

	// Require a pool entry to clone from
	poolEntry, err := pool.GetEntry(db, pkgID)
	if err != nil {
		return err
	}

	localPath := pool.GetMetaPoolPath(pkgID, poolEntry.Meta)
	targetDir := filepath.Join(r.path, poolEntry.Meta.GetPathComponent())
	targetPath := filepath.Join(targetDir, pkgID)

	repoEntry := r.buildSaneEntry(db, pool, poolEntry.Meta, pkgID)
	// Already included
	if repoEntry == nil {
		return nil
	}

	// Construct root dirs
	if err := os.MkdirAll(targetDir, 00755); err != nil {
		return err
	}

	// Grab the pool reference for this package (Always copy)
	if err = pool.RefEntry(db, pkgID); err != nil {
		return err
	}

	// Ensure the eopkg file is linked inside our own tree
	if err = LinkOrCopyFile(localPath, targetPath, false); err != nil {
		return err
	}

	return r.putEntry(db, repoEntry)
}

// buildSaneEntry will either return a plain entry if none exists already, otherwise it will
// take an existing entry and correctly set up the available/published fields
func (r *Repository) buildSaneEntry(db libdb.Database, pool *Pool, newPkg *libeopkg.MetaPackage, newID string) *RepoEntry {
	// Fallback in case one actually doesn't exist yet
	repoEntry := &RepoEntry{
		SchemaVersion: RepoSchemaVersion,
		Name:          newPkg.Name,
		Published:     newID,
	}

	// Not so worried about the error, just having the entry
	entry, _ := r.GetEntry(db, newPkg.Name)

	// Clone across the relevant field now
	if entry != nil {
		repoEntry.Available = entry.Available
		repoEntry.Published = entry.Published

		pkgAvail, err := pool.GetEntry(db, repoEntry.Published)
		if err == nil {
			if newPkg.GetRelease() > pkgAvail.Meta.GetRelease() {
				repoEntry.Published = newID
			} else if newPkg.GetRelease() == pkgAvail.Meta.GetRelease() && pkgAvail.Name != newID {
				fmt.Printf(" **** DUPLICATE RELEASE NUMBER DETECTED. FAK: %s %s **** \n", newID, pkgAvail.Name)
			}
		} else {
			repoEntry.Published = newID
		}
	}

	// Check if we've already indexed it, non-fatal
	for _, id := range repoEntry.Available {
		if id == newID {
			fmt.Printf("Skipping already included %s\n", id)
			return nil
		}
	}

	// Keep the available list clean + sorted
	repoEntry.Available = append(repoEntry.Available, newID)
	sort.Strings(repoEntry.Available)

	return repoEntry
}

// AddLocalPackage will do the real work of adding an open & loaded eopkg to the repository
func (r *Repository) AddLocalPackage(db libdb.Database, pool *Pool, pkg *libeopkg.Package) error {
	r.insertMut.Lock()
	defer r.insertMut.Unlock()

	pkgDir := filepath.Join(r.path, pkg.Meta.Package.GetPathComponent())
	pkgTarget := filepath.Join(pkgDir, pkg.ID)

	// Already have a package, so let's copy the existing bits over
	repoEntry := r.buildSaneEntry(db, pool, &pkg.Meta.Package, pkg.ID)

	// nil == already included
	if repoEntry == nil {
		return nil
	}

	// Construct root dirs
	if err := os.MkdirAll(pkgDir, 00755); err != nil {
		return err
	}

	// Grab the pool reference for this package (Always copy)
	if _, err := pool.AddPackage(db, pkg, false); err != nil {
		return err
	}

	// Ensure the eopkg file is linked inside our own tree
	source := pool.GetPackagePoolPath(pkg)
	if err := LinkOrCopyFile(source, pkgTarget, false); err != nil {
		return err
	}

	return r.putEntry(db, repoEntry)
}

// AddPackage will attempt to load the local package and then add it to the
// repository via AddLocalPackage
func (r *Repository) AddPackage(db libdb.Database, pool *Pool, filename string) error {
	pkg, err := libeopkg.Open(filename)
	if err != nil {
		return err
	}

	defer pkg.Close()
	if err = pkg.ReadMetadata(); err != nil {
		return err
	}

	return r.AddLocalPackage(db, pool, pkg)
}

// GetPackageNames will traverse the buckets and find all package names as stored
// within the DB. This doesn't account for obsolete names, which should in fact
// be removed from the repo entirely.
func (r *Repository) GetPackageNames(db libdb.Database) ([]string, error) {
	var pkgIds []string
	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))

	err := rootBucket.ForEach(func(k, v []byte) error {
		entry := RepoEntry{}
		if err := rootBucket.Decode(v, &entry); err != nil {
			return err
		}
		pkgIds = append(pkgIds, entry.Name)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return pkgIds, nil
}

// GetPackages will return all package objects for a given name
func (r *Repository) GetPackages(db libdb.Database, pool *Pool, pkgName string) ([]*libeopkg.MetaPackage, error) {
	var pkgs []*libeopkg.MetaPackage

	entry, err := r.GetEntry(db, pkgName)
	if err != nil || entry == nil {
		return nil, err
	}

	for _, id := range entry.Available {
		p, err := pool.GetEntry(db, id)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p.Meta)
	}

	return pkgs, nil
}

// CreateDelta is responsible for trying to create a new delta package between
// oldPkg and newPkg, with newPkg being the delta *to*.
//
// This function may fail to produce a delta because they're incompatible packages,
// or because a delta between the two packages would be pointless (i.e. they're
// either identical or 100% the same.)
//
// Lastly, this function will move the delta out of the build area into the
// staging area if it successfully produces a delta. This does not mark a delta
// attempt as "pointless", nor does it actually *include* the delta package
// within the repository.
func (r *Repository) CreateDelta(db libdb.Database, oldPkg, newPkg *libeopkg.MetaPackage) (string, error) {
	if !libeopkg.IsDeltaPossible(oldPkg, newPkg) {
		return "", libeopkg.ErrMismatchedDelta
	}
	fileName := libeopkg.ComputeDeltaName(oldPkg, newPkg)
	fullPath := filepath.Join(r.deltaStagePath, fileName)

	// This guy exists, no point in trying to rebuild it
	if PathExists(fullPath) {
		return fullPath, nil
	}

	oldPath := filepath.Join(r.path, oldPkg.PackageURI)
	newPath := filepath.Join(r.path, newPkg.PackageURI)

	if err := ProduceDelta(r.deltaPath, oldPath, newPath, fullPath); err != nil {
		return "", err
	}

	return fullPath, nil
}

// HasDelta will work out if we actually have a delta already
func (r *Repository) HasDelta(db libdb.Database, pkgName, deltaPath string) (bool, error) {
	entry, err := r.GetEntry(db, pkgName)
	if err != nil {
		return false, err
	}
	for _, pkgDelta := range entry.Deltas {
		if deltaPath == pkgDelta {
			return true, nil
		}
	}
	return false, nil
}

// pullAssets will pull the various asset files in prior to indexing
func (r *Repository) pullAssets(sourceRepo *Repository) error {
	copyPaths := []string{
		filepath.Join(sourceRepo.assetPath, "distribution.xml"),
		filepath.Join(sourceRepo.assetPath, "components.xml"),
		filepath.Join(sourceRepo.assetPath, "groups.xml"),
	}

	// In case anyone is being cranky ..
	if !PathExists(r.assetPath) {
		if err := os.MkdirAll(r.assetPath, 00755); err != nil {
			return err
		}
	}

	for _, p := range copyPaths {
		if !PathExists(p) {
			continue
		}
		dstPath := filepath.Join(r.assetPath, filepath.Base(p))
		if err := CopyFile(p, dstPath); err != nil {
			return err
		}
	}

	return nil
}

// CloneFrom will attempt to clone everything from the target repository into
// ourselves
func (r *Repository) CloneFrom(db libdb.Database, pool *Pool, sourceRepo *Repository, fullClone bool) error {
	// First things first, instigate a write lock on the target
	sourceRepo.insertMut.Lock()
	defer sourceRepo.insertMut.Unlock()

	var copyIDs []string
	var deltaIDs []string

	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(sourceRepo.ID)).Bucket([]byte(DatabaseBucketPackage))

	// Before doing anything, sync the assets
	if err := r.pullAssets(sourceRepo); err != nil {
		return err
	}

	// Grab every package
	err := rootBucket.ForEach(func(k, v []byte) error {
		entry := RepoEntry{}
		if err := rootBucket.Decode(v, &entry); err != nil {
			return err
		}

		if fullClone {
			copyIDs = append(copyIDs, entry.Available...)
			deltaIDs = append(deltaIDs, entry.Deltas...)
		} else {
			copyIDs = append(copyIDs, entry.Published)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Now we'll insert all the new IDs. We can't really transaction this as
	// we're going to rely on on the refcount cycle and updating published/available
	// depending on tip or ALL
	for _, id := range copyIDs {
		if err := r.RefPackage(db, pool, id); err != nil {
			return err
		}
	}

	// We can only copy deltas across on full clones.
	for _, id := range deltaIDs {
		if err := r.RefDelta(db, pool, id); err != nil {
			return err
		}
	}

	return nil
}

// PullFrom will iterate the source repositories contents, looking for any packages
// we can pull into ourselves.
//
// If a package is missing from our own indexes (i.e. no key) we'll pull that package.
// If a package is present in our own indexes, but the package in sourceRepo's published
// field is actually _newer_ than ours, we'll pull that guy in too.
//
// Note this isn't "pull" in the git sense, as we'll not perform any removals or attempt
// to sync the states completely. Pull is typically used on a clone from a volatile target,
// i.e. pulling from unstable into stable. As many rebuilds might happen in unstable, we
// actually only want to bring in the *tip* from the new sources.
//
// Over time, a clone will drift in content from the source. In terms of upgrade paths
// for users, they'll only follow *one* repository, i.e. the stable rolling snapshot
// channel. To be able to provide usable and effective delta paths for those users, we
// really need to form deltas per repository between their effective tip and "old" items,
// otherwise with a git-like sync the users might never see any useful delta packages
// for the fast moving items.
//
// Drift can be corrected by nuking a repository and performing a full clone from the
// source to have identical mirrors again. This should be performed rarely and only
// during periods of maintenance due to this method violating atomic indexes.
func (r *Repository) PullFrom(db libdb.Database, pool *Pool, sourceRepo *Repository) error {
	// First things first, instigate a write lock on the source
	sourceRepo.insertMut.Lock()
	defer sourceRepo.insertMut.Unlock()

	var copyIDs []string

	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(sourceRepo.ID)).Bucket([]byte(DatabaseBucketPackage))

	// Before doing anything, sync the assets
	if err := r.pullAssets(sourceRepo); err != nil {
		return err
	}

	// Grab every package
	err := rootBucket.ForEach(func(k, v []byte) error {
		entry := RepoEntry{}
		if err := rootBucket.Decode(v, &entry); err != nil {
			return err
		}

		localEntry, _ := r.GetEntry(db, entry.Name)

		// We haven't got this, copy the published version
		if localEntry == nil {
			copyIDs = append(copyIDs, entry.Published)
			return nil
		}

		// We have got this, so is it newer than ours?
		tipVer, err := pool.GetEntry(db, entry.Published)
		if err != nil {
			return err
		}
		ourTip, err := pool.GetEntry(db, localEntry.Published)
		if err != nil {
			return err
		}

		// Their tip is newer than ours, copy it
		if tipVer.Meta.GetRelease() > ourTip.Meta.GetRelease() {
			copyIDs = append(copyIDs, entry.Published)
		}

		// Is something completely bork?
		if tipVer.Meta.GetRelease() < ourTip.Meta.GetRelease() {
			return fmt.Errorf("inconsistent target repository, %v is NEWER in target not SOURCE", localEntry.Name)
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Now we'll insert all the new IDs. We can't really transaction this as
	// we're going to rely on on the refcount cycle and updating published/available
	// depending on tip or ALL
	for _, id := range copyIDs {
		if err := r.RefPackage(db, pool, id); err != nil {
			return err
		}
	}

	return nil
}

// RemoveSource will remove all packages that have a matching source name and
// release number. This allows us to remove multiple packages from a single
// upload/set in one go.
//
// Distributions tend to split packages across a common identifier/release
// and this method will allow us to remove "bad actors" from the index.
func (r *Repository) RemoveSource(db libdb.Database, pool *Pool, sourceID string, release int) error {
	var deleteIDs []string

	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))

	// Grab every package
	err := rootBucket.ForEach(func(k, v []byte) error {
		entry := RepoEntry{}
		if err := rootBucket.Decode(v, &entry); err != nil {
			return err
		}

		// Now we must grab every "available" package for this bucket entry
		for _, id := range entry.Available {
			poolEntry, err := pool.GetEntry(db, id)
			if err != nil {
				return err
			}

			if poolEntry.Meta.Source.Name != sourceID {
				continue
			}

			if poolEntry.Meta.GetRelease() != release {
				continue
			}

			// We've got a match.
			deleteIDs = append(deleteIDs, id)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Now we'll remove all the defunct IDs. We can't really transaction this as
	// we're going to rely on on the refcount cycle.
	for _, id := range deleteIDs {
		if err = r.UnrefPackage(db, pool, id); err != nil {
			return err
		}
	}

	return nil
}

// TrimObsolete isn't very straight forward as it has to account for some
// janky behaviour in eopkg.
//
// Effectively - any name explicitly marked as obsolete is something we need
// to remove from the repository. However, we also need to apply certain
// modifications to ensure child packages (-dbginfo) are also nuked along
// with them.
func (r *Repository) TrimObsolete(db libdb.Database, pool *Pool) error {
	if err := r.initDistribution(); err != nil {
		return err
	}

	// All the guys who we're sending to the big bitsink in the sky
	var removalIDs []string

	// TODO: Scream loudly that someones being an eejit and trying to obsolete
	// packages without a distribution.xml defined.
	if r.dist == nil {
		return nil
	}

	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))

	// Grab every package
	err := rootBucket.ForEach(func(k, v []byte) error {
		entry := RepoEntry{}
		if err := rootBucket.Decode(v, &entry); err != nil {
			return err
		}

		for _, id := range entry.Available {
			poolEntry, err := pool.GetEntry(db, id)
			if err != nil {
				return err
			}

			// Retain compatibility with eopkg, auto-drop -dbginfo
			nom := poolEntry.Meta.Name
			if strings.HasSuffix(nom, "-dbginfo") {
				nom = nom[0 : len(nom)-8]
			}

			// Check if its obsolete, if its automatically obsolete through our
			// dbginfo trick, warn in the console
			if r.dist != nil && r.dist.IsObsolete(nom) {
				if nom != entry.Name {
					// Scream really loudly, but remove it because its "just" dbginfo.
					fmt.Fprintf(os.Stderr, " **** ABANDONED OBSOLETE PACKAGE: %s ****\n", poolEntry.Meta.Name)
					removalIDs = append(removalIDs, id)
				}
				return nil
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Now attempt to unref every one of the packages marked as obsolete
	for _, id := range removalIDs {
		fmt.Fprintf(os.Stderr, "Removing obsolete package: %v\n", id)
		if err := r.UnrefPackage(db, pool, id); err != nil {
			return err
		}
	}

	return nil
}

// TrimPackages will trim back the packages in each package entry to a maximum
// amount of packages, which helps to combat the issue of rapidly inserting
// many builds into a repo, i.e. removing old backversions
func (r *Repository) TrimPackages(db libdb.Database, pool *Pool, maxKeep int) error {
	// All the guys who we're sending to the big bitsink in the sky
	var removalIDs []string

	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))

	// Grab every package
	err := rootBucket.ForEach(func(k, v []byte) error {
		entry := RepoEntry{}
		if err := rootBucket.Decode(v, &entry); err != nil {
			return err
		}

		// micro optimisation ..
		if len(entry.Available) < maxKeep {
			return nil
		}

		var candidates []*libeopkg.MetaPackage

		for _, id := range entry.Available {
			poolEntry, err := pool.GetEntry(db, id)
			if err != nil {
				return err
			}
			candidates = append(candidates, poolEntry.Meta)
		}

		sort.Sort(sort.Reverse(libeopkg.PackageSet(candidates)))

		for i := maxKeep; i < len(candidates); i++ {
			id := candidates[i].GetID()
			// Technically impossible but best to be safe.
			if id == entry.Published {
				continue
			}
			removalIDs = append(removalIDs, candidates[i].GetID())
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Now attempt to unref every one of the packages marked as obsolete
	for _, id := range removalIDs {
		fmt.Fprintf(os.Stderr, "Trimming old package: %v\n", id)
		if err := r.UnrefPackage(db, pool, id); err != nil {
			return err
		}
	}

	return nil
}
