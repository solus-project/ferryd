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
	"github.com/boltdb/bolt"
	"libeopkg"
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
	RefCount int               // Number of times duplicated
	Metadata libeopkg.Metadata // Package information for this file
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
