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
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/boltdb/bolt"
	"path/filepath"
)

var (
	// RepoDirectory is the base directory for all repositories.
	RepoDirectory = "repo"
)

// A Repository is the base unit of storage in binman
type Repository struct {
	Name string
}

// GetDirectory will return the directory component for where this
// repository lives on disk.
func (r *Repository) GetDirectory() string {
	return filepath.Join(RepoDirectory, r.Name)
}

// BucketPathPackages will return the unique repository packages bucket name
//
// A Repository has it's own named bucket within the Repo bucket,
// which corresponds to the name of this repository.
func (r *Repository) BucketPathPackages() []byte {
	return []byte(fmt.Sprintf("%s.%s", BucketPrefixPackages, r.Name))
}

// CreateRepo will attempt to create a new repository
func (m *Manager) CreateRepo(name string) error {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	repo := &Repository{
		Name: name,
	}
	if err := enc.Encode(repo); err != nil {
		return err
	}

	// Encoded name
	nom := []byte(name)

	return m.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNameRepos)
		// Check it doesn't already exist in the bucket
		if len(b.Get(nom)) != 0 {
			return ErrResourceExists
		}
		path := repo.BucketPathPackages()
		if _, err := b.CreateBucketIfNotExists(path); err != nil {
			return err
		}
		return b.Put(nom, buf.Bytes())
	})
}

// ListRepos will return a list of repository names known to binman.
func (m *Manager) ListRepos() ([]string, error) {
	var repos []string
	err := m.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNameRepos)
		return b.ForEach(func(k, v []byte) error {
			// Skip a namespaced bucket
			if b.Bucket(k) != nil {
				return nil
			}
			repos = append(repos, string(k))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return repos, nil
}

// RemoveRepo will remove a repository from binman. In future this will also
// have to request the pool check for all unreferenced files and delete them
// too.
func (m *Manager) RemoveRepo(name string) error {
	err := m.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNameRepos)
		nom := []byte(name)
		if len(b.Get(nom)) == 0 {
			return ErrUnknownResource
		}
		r := Repository{Name: name}
		// Delete package bucket
		if err := b.DeleteBucket(r.BucketPathPackages()); err != nil {
			return err
		}
		return b.Delete(nom)
	})
	return err
}

// GetRepo will attempt to grab the named repo, if it exists.
func (m *Manager) GetRepo(name string) (*Repository, error) {
	repo := &Repository{}
	nom := []byte(name)

	err := m.db.View(func(tx *bolt.Tx) error {
		blob := tx.Bucket(BucketNameRepos).Get(nom)
		if len(blob) == 0 {
			return ErrUnknownResource
		}
		buf := bytes.NewBuffer(blob)
		dec := gob.NewDecoder(buf)
		return dec.Decode(repo)
	})
	if err != nil {
		return nil, err
	}
	return repo, nil
}
