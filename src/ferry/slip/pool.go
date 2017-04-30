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
)

const (
	// DatabaseBucketPool is the identifier for the pool main bucket
	DatabaseBucketPool = "pool"
)

// A Pool is used to manage and deduplicate resources between multiple resources,
// and represents the real backing store for referenced eopkg files.
type Pool struct {
	db *bolt.DB
}

// NewPool will return a new Pool instance for use by the Manager
func NewPool(db *bolt.DB) *Pool {
	return &Pool{
		db: db,
	}
}
