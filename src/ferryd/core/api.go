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
	"github.com/boltdb/bolt"
	"libeopkg"
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
	if err != nil {
		return err
	}
	// Index the newly created repo
	return m.Index(id)
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

	err = m.db.Update(func(tx *bolt.Tx) error {
		for _, pkg := range packages {
			if err := repo.AddPackage(tx, m.pool, pkg); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Now emit the repo index itself
	return m.db.View(func(tx *bolt.Tx) error {
		return repo.Index(tx, m.pool)
	})
}

// Index will cause the repository's index to be reconstructed
func (m *Manager) Index(repoID string) error {
	repo, err := m.GetRepo(repoID)
	if err != nil {
		return err
	}

	// Now emit the repo index itself
	return m.db.View(func(tx *bolt.Tx) error {
		return repo.Index(tx, m.pool)
	})
}

// GetPackageNames will attempt to load all package names for the given
// repository.
func (m *Manager) GetPackageNames(repoID string) ([]string, error) {
	repo, err := m.GetRepo(repoID)
	if err != nil {
		return nil, err
	}

	var ret []string
	err = m.db.View(func(tx *bolt.Tx) error {
		ret, err = repo.GetPackageNames(tx)
		return err
	})

	return ret, err
}

// GetPackages will return a set of packages for the package name within the
// specified repository
func (m *Manager) GetPackages(repoID, pkgName string) ([]*libeopkg.MetaPackage, error) {
	repo, err := m.GetRepo(repoID)
	if err != nil {
		return nil, err
	}

	var ret []*libeopkg.MetaPackage
	err = m.db.View(func(tx *bolt.Tx) error {
		ret, err = repo.GetPackages(tx, m.pool, pkgName)
		return err
	})

	return ret, err
}

// CreateDelta will attempt to create a new delta package between the old and new IDs
func (m *Manager) CreateDelta(repoID string, oldPkg, newPkg *libeopkg.MetaPackage) error {
	repo, err := m.GetRepo(repoID)
	if err != nil {
		return err
	}

	return m.db.View(func(tx *bolt.Tx) error {
		return repo.CreateDelta(tx, oldPkg, newPkg)
	})
}
