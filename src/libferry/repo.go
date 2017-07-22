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
	"fmt"
	"github.com/boltdb/bolt"
	"os"
	"path/filepath"
)

const (
	// RepoPathComponent is the base for all repository directories
	RepoPathComponent = "repo"

	// DatabaseBucketRepo is the name for the main repo toplevel bucket
	DatabaseBucketRepo = "repo"
)

// The RepositoryManager maintains all repos within ferryd which are in
// turn linked to the main pool
type RepositoryManager struct {
	repoBase   string
	transcoder *GobTranscoder
}

// A Repository is a simplistic representation of a exported repository
// within ferryd
type Repository struct {
	ID   string // Name of this repository (unique)
	path string // Where this is on disk
}

// Init will create our initial working paths and DB bucket
func (r *RepositoryManager) Init(ctx *Context, tx *bolt.Tx) error {
	r.repoBase = filepath.Join(ctx.BaseDir, RepoPathComponent)
	r.transcoder = NewGobTranscoder()
	if err := os.MkdirAll(r.repoBase, 00755); err != nil {
		return err
	}
	_, err := tx.CreateBucketIfNotExists([]byte(DatabaseBucketRepo))
	return err
}

// Close doesn't currently do anything
func (r *RepositoryManager) Close() {}

// GetRepo will attempt to get the named repo if it exists, otherwise
// return an error. This is a transactional helper to make the API simpler
func (r *RepositoryManager) GetRepo(tx *bolt.Tx, id string) (*Repository, error) {
	rootBucket := tx.Bucket([]byte(DatabaseBucketRepo))
	repo := rootBucket.Bucket([]byte(id))
	if repo == nil {
		return nil, fmt.Errorf("The specified repository '%s' does not exist", id)
	}
	return &Repository{
		ID:   id,
		path: filepath.Join(r.repoBase, id),
	}, nil
}

// CreateRepo will create a new repository (bucket) within the top level
// repo bucket.
func (r *RepositoryManager) CreateRepo(tx *bolt.Tx, id string) (*Repository, error) {
	if _, err := r.GetRepo(tx, id); err == nil {
		return nil, fmt.Errorf("The specified repository '%s' already exists", id)
	}
	rootBucket := tx.Bucket([]byte(DatabaseBucketRepo))
	if _, err := rootBucket.CreateBucket([]byte(id)); err != nil {
		return nil, err
	}
	repoDir := filepath.Join(r.repoBase, id)
	if err := os.MkdirAll(repoDir, 00755); err != nil {
		return nil, err
	}
	return &Repository{
		ID:   id,
		path: repoDir,
	}, nil
}
