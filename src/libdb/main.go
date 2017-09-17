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

package libdb

import (
	"errors"
	"sync"
)

// Database is the opaque interface to the underlying database implementation
type Database struct {
	storagePath string // Where we are on disk
	closeMut    *sync.Mutex
	closed      bool
	handle      databaseHandle
}

type databaseHandle interface {
	Close() // Close handle to database
}

// Open will return an opaque representation of the underlying database
// implementation suitable for usage within ferryd
func Open(path string) (*Database, error) {
	return nil, errors.New("Not yet implemented")
}

// Close the underlying storage
func (d *Database) Close() {
	d.closeMut.Lock()
	defer d.closeMut.Unlock()

	if d.closed {
		return
	}
	d.handle.Close()
	d.closed = true
}
