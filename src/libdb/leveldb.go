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
	tr := NewGobDecoderLight()
	val, err := l.db.Get(id, nil)
	if err != nil {
		return err
	}
	if err = tr.DecodeType(val, outObject); err != nil {
		outObject = nil
		return err
	}
	return nil
}

func (l *levelDbHandle) PutObject(id []byte, inObject interface{}) error {
	tr := NewGobEncoderLight()
	by, err := tr.EncodeType(inObject)
	if err != nil {
		return err
	}
	return l.db.Put(id, by, nil)
}

func (l *levelDbHandle) ForEach(f DbForeachFunc) error {
	// No matching on prefixes..
	iter := l.db.NewIterator(nil, nil)
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

// ForEachType will expect the input type to match that of the input function,
// attempting to convert it on each loop into a usable, already decoded object.
func (l *levelDbHandle) ForEachType(inType interface{}, f DbForeachTypeFunc) error {
	// No matching on prefixes..
	iter := l.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		dec := NewGobDecoderLight()
		if err := dec.DecodeType(value, inType); err != nil {
			return err
		}

		if err := f(key, dec); err != nil {
			return err
		}
	}
	return iter.Error()
}
