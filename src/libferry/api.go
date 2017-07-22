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
	"github.com/boltdb/bolt"
)

// This file provides the public API functions which are used by ferryd
// and exposed through it's handlers.

// CreateRepo will request the creation of a new repository
func (m *Manager) CreateRepo(id string) error {
	err := m.db.Update(func(tx *bolt.Tx) error {
		_, err := m.repo.CreateRepo(tx, id)
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

// GetRepo will grab the repository if it exists
// Note that this is a read only operation
func (m *Manager) GetRepo(id string) (*Repository, error) {
	var repo *Repository
	err := m.db.View(func(tx *bolt.Tx) error {
		r, err := m.repo.GetRepo(tx, id)
		if err != nil {
			return err
		}
		repo = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// AddPackages will attempt to add the named packages to the repository
func (m *Manager) AddPackages(repoID string, packages []string) error {
	repo, err := m.GetRepo(repoID)
	if err != nil {
		return err
	}
	return m.db.Update(func(tx *bolt.Tx) error {
		for _, pkg := range packages {
			if err := repo.AddPackage(tx, pkg); err != nil {
				return err
			}
		}
		return nil
	})
}
