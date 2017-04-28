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
	"os"
	"path/filepath"
)

// A Manager is the the singleton responsible for slip management
type Manager struct {
	db   *bolt.DB // Open database connection
	path string   // Path to our DB file
}

// NewManager will attempt to instaniate a manager for the given path,
// which will yield an error if the database cannot be opened for access.
func NewManager(path string) (*Manager, error) {
	// Ensure the initial directory exists first
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}
	dbPath, err := filepath.Abs(filepath.Join(path, DatabasePathComponent))
	if err != nil {
		return nil, err
	}

	// Open the database if we can
	// TODO: Add a timeout for locks
	db, err := bolt.Open(dbPath, 00600, nil)
	if err != nil {
		return nil, err
	}
	return &Manager{
		db:   db,
		path: dbPath,
	}, nil
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
