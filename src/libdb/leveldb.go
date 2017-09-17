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
	"github.com/syndtr/goleveldb/leveldb"
)

// levelDbHandle wraps leveldb up in private API
type levelDbHandle struct {
	closable
	db *leveldb.DB
}

func newLevelDBHandle(storagePath string) (*levelDbHandle, error) {
	// TODO: Set up options, support read-only, etc.
	ldb, err := leveldb.OpenFile(storagePath, nil)
	if err != nil {
		return nil, err
	}
	handle := &levelDbHandle{
		db: ldb,
	}
	handle.initClosable()
	return handle, nil
}

// Close the existing levelDbHandle
func (l *levelDbHandle) Close() {
	if l.close() {
		l.db.Close()
	}
}

func (l *levelDbHandle) GetObject(id []byte, outObject interface{}) error {
	return errors.New("Not yet implemented")
}

func (l *levelDbHandle) PutObject(id []byte, inObject interface{}) error {
	tr := NewGobEncoderLight()
	by, err := tr.EncodeType(inObject)
	if err != nil {
		return err
	}
	return l.db.Put(id, by, nil)
}
