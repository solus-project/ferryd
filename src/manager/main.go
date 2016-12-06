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
	_ "github.com/boltdb/bolt"
)

// A Manager is used for all binman operations and stores the global
// state, database, etc.
type Manager struct {
}

// New will return a new Manager instance
func New() *Manager {
	return &Manager{}
}

// Cleanup would clean up the manager instance but is largely a no-op
// right now.
func (m *Manager) Cleanup() {
}
