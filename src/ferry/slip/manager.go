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
	"github.com/boltdb/bolt"
)

// A Manager is the the singleton responsible for slip management
type Manager struct {
	db   *bolt.DB // Open database connection
	ctx  *Context // Context shares all our path assignments
	pool *Pool    // Our main pool for eopkgs
}

// NewManager will attempt to instaniate a manager for the given path,
// which will yield an error if the database cannot be opened for access.
func NewManager(path string) (*Manager, error) {
	ctx, err := NewContext(path)
	if err != nil {
		return nil, err
	}

	// Open the database if we can
	// TODO: Add a timeout for locks
	db, err := bolt.Open(ctx.DbPath, 00600, nil)
	if err != nil {
		return nil, err
	}
	m := &Manager{
		db:   db,
		ctx:  ctx,
		pool: NewPool(db),
	}

	// Initialise the buckets in a one-time
	if err = m.initBuckets(); err != nil {
		m.Close()
		return nil, err
	}

	return m, nil
}

// initBuckets will ensure all initial buckets are create in the toplevel
// namespace, to require less complexity further down the line
func (m *Manager) initBuckets() error {
	// TODO: Use constants here!
	buckets := []string{
		"endpoint",
		"repo",
		DatabaseBucketPool,
	}

	// Create all root-level buckets in a single transaction
	return m.db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range buckets {
			if _, e := tx.CreateBucketIfNotExists([]byte(bucket)); e != nil {
				return e
			}
		}
		return nil
	})
}

// Close will close and clean up any associated resources, such as the
// underlying database.
func (m *Manager) Close() {
	if m.db == nil {
		return
	}
	m.db.Close()
	m.db = nil
}
