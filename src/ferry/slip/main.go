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

// Package slip provides the Ferry Slip implementation.
//
// This portion of ferryd is responsible for the management of management
// of the repositories, and receives packages from the builders.
// In the ferryd design, packages are ferried to the slip, where it is then
// organised into the repositories.
package slip

import (
	"errors"
	"github.com/boltdb/bolt"
)

// A Manager is the the singleton responsible for slip management
type Manager struct {
	db *bolt.DB
}

// NewManager will attempt to instaniate a manager for the given path,
// which will yield an error if the database cannot be opened for access.
func NewManager() (*Manager, error) {
	return nil, errors.New("Not yet implemented")
}

// Close will close and clean up any associated resources, such as the
// underlying database.
func (m *Manager) Close() {
}
