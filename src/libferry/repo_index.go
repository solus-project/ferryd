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
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"sort"
)

// Index will attempt to write the eopkg index out to disk
// This only requires a read-only database view
func (r *Repository) Index(tx *bolt.Tx, pool *Pool) error {
	var pkgIds []string
	rootBucket := tx.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))

	c := rootBucket.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		code := NewGobDecoderLight()
		entry := RepoEntry{}
		if err := code.DecodeType(v, &entry); err != nil {
			return err
		}
		pkgIds = append(pkgIds, entry.Published)
	}

	// Ensure we'll emit in a sane order
	sort.Strings(pkgIds)

	for _, pkg := range pkgIds {
		entry, err := pool.GetEntry(tx, pkg)
		if err != nil {
			return err
		}
		fmt.Printf("-> Package '%s' (%s-%d)\n", entry.Name, entry.Meta.GetVersion(), entry.Meta.GetRelease())
		fmt.Printf("    -> %v\n", entry.Meta.Summary)
	}

	return errors.New("not yet implemented")
}
