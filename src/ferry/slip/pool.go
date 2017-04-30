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
	"errors"
	"github.com/boltdb/bolt"
	"os"
	"path/filepath"
)

const (
	// DatabaseBucketPool is the identifier for the pool main bucket
	DatabaseBucketPool = "pool"

	// PoolPathComponent is the storage directory for all of our main files
	PoolPathComponent = "pool"
)

// A Pool is used to manage and deduplicate resources between multiple resources,
// and represents the real backing store for referenced eopkg files.
type Pool struct {
	poolDir string // Storage area
}

// Init will create our initial working paths and DB bucket
func (p *Pool) Init(ctx *Context, tx *bolt.Tx) error {
	p.poolDir = filepath.Join(ctx.BaseDir, PoolPathComponent)
	if err := os.MkdirAll(p.poolDir, 00755); err != nil {
		return err
	}
	_, err := tx.CreateBucketIfNotExists([]byte(DatabaseBucketPool))
	return err
}

// Close doesn't currently do anything
func (p *Pool) Close() {}

// RefEntry will include the given eopkg if it doesn't yet exist, otherwise
// it will simply increase the ref count by 1.
func (p *Pool) RefEntry(id string) error {
	return errors.New("Not yet implemented")
}

// UnrefEntry will unref a given ID from the repository.
// Should the refcount hit 0, the package will then be removed from the pool
// storage.
func (p *Pool) UnrefEntry(id string) error {
	return errors.New("Not yet implemented")
}
