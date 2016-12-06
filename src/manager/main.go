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

// Package manager provides the main guts of binman itself.
package manager

import (
	// Force boltdb into the build
	"github.com/boltdb/bolt"
)

// A Manager is used for all binman operations and stores the global
// state, database, etc.
type Manager struct {
	db *bolt.DB
}

// New will return a new Manager instance
func New() (*Manager, error) {
	// TODO: Support read-only operation, and don't hardcode the feckin'
	// path.
	options := &bolt.Options{
		Timeout: 0,
	}
	db, err := bolt.Open("binman.db", 0600, options)
	if err != nil {
		return nil, err
	}
	return &Manager{
		db: db,
	}, nil
}

// Cleanup will close any resources that this Manager instance owns, such
// as the main database.
func (m *Manager) Cleanup() {
	if m.db == nil {
		return
	}
	m.db.Close()
	m.db = nil
}
