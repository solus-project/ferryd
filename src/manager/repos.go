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
	"encoding/json"
	"errors"
	"github.com/boltdb/bolt"
)

var (
	// BucketNameRepos is the fixed name of the repositories bucket
	BucketNameRepos = []byte("repos")

	// ErrRepoExists is returned when a repository alread exists, and the
	// user tries to create a new repo.
	ErrRepoExists = errors.New("The specified repository already exists")
)

// A Repository is the base unit of storage in binman
type Repository struct {
	Name string
}

// CreateRepo will attempt to create a new repository
func (m *Manager) CreateRepo(name string) error {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
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
		if b.Get(nom) != nil {
			return ErrRepoExists
		}
		return tx.Bucket(BucketNameRepos).Put(nom, buf.Bytes())
	})
}

// ListRepos will return a list of repository names known to binman.
func (m *Manager) ListRepos() ([]string, error) {
	var repos []string
	err := m.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNameRepos)
		return b.ForEach(func(k, v []byte) error {
			repos = append(repos, string(k))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return repos, nil
}
