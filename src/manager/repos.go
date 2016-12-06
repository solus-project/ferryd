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
	"github.com/boltdb/bolt"
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
	return m.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("repos")).Put([]byte(name), buf.Bytes())
	})
}
