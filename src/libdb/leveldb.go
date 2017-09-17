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
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// levelDbHandle wraps leveldb up in private API
type levelDbHandle struct {
	prefixBytes *util.Range
	prefix      []byte
	db          *leveldb.DB
}

// levelDb is our concrete type
type levelDb struct {
	levelDbHandle
	closable
}

func newLevelDBHandle(storagePath string) (*levelDb, error) {
	// TODO: Set up options, support read-only, etc.
	ldb, err := leveldb.OpenFile(storagePath, nil)
	if err != nil {
		return nil, err
	}
	handle := &levelDb{}
	handle.db = ldb
	handle.prefix = []byte("rootBucket-")
	handle.prefixBytes = util.BytesPrefix(handle.prefix)
	handle.initClosable()
	return handle, nil
}

// Close the existing levelDbHandle
func (l *levelDb) Close() {
	if l.close() {
		l.db.Close()
	}
}

func (l *levelDbHandle) getRealKey(id []byte) []byte {
	return []byte(fmt.Sprintf("%s-%s", string(l.prefix), string(id)))
}

func (l *levelDbHandle) GetObject(id []byte, outObject interface{}) error {
	val, err := l.db.Get(l.getRealKey(id), nil)
	if err != nil {
		return err
	}

	return l.Decode(val, outObject)
}

func (l *levelDbHandle) PutObject(id []byte, inObject interface{}) error {
	tr := NewGobEncoderLight()
	by, err := tr.EncodeType(inObject)
	if err != nil {
		return err
	}
	return l.db.Put(l.getRealKey(id), by, nil)
}

func (l *levelDbHandle) Decode(input []byte, o interface{}) error {
	tr := NewGobDecoderLight()
	if err := tr.DecodeType(input, o); err != nil {
		o = nil
		return err
	}
	return nil
}

func (l *levelDbHandle) ForEach(f DbForeachFunc) error {
	// No matching on prefixes..
	iter := l.db.NewIterator(l.prefixBytes, nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		if err := f(key, value); err != nil {
			return err
		}
	}
	return iter.Error()
}

// Close is a no-op for our handle
func (l *levelDbHandle) Close() {}

func (l *levelDbHandle) Bucket(id []byte) Database {
	var newID []byte
	if l.prefix != nil {
		newID = []byte(fmt.Sprintf("bucket-%s-%s", string(l.prefix), id))
	} else {
		newID = []byte(fmt.Sprintf("bucket-%s", id))
	}
	ret := &levelDbHandle{
		db:          l.db,
		prefix:      newID,
		prefixBytes: util.BytesPrefix(newID),
	}
	return ret
}
