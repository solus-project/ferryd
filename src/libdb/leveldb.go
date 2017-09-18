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
	"bytes"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var (
	rootBucketPrefix = []byte("|rootBucket|-")
	bucketPrefix     = []byte("|bucket|")
)

// levelDbHandle wraps leveldb up in private API
type levelDbHandle struct {
	prefixBytes *util.Range
	prefix      []byte
	keyPrefix   []byte
	db          *leveldb.DB
	batch       *leveldb.Batch // Usually nil but set for write transactions
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
	handle.prefix = []byte("|rootBucket|")
	handle.keyPrefix = []byte("|rootBucket|-")
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

func (l *levelDbHandle) HasObject(id []byte) (bool, error) {
	return l.db.Has(l.getRealKey(id), nil)
}

func (l *levelDbHandle) PutObject(id []byte, inObject interface{}) error {
	if bytes.HasPrefix(id, bucketPrefix) || bytes.HasPrefix(id, rootBucketPrefix) {
		return fmt.Errorf("key uses reserved bucket notation: %v", string(id))
	}

	tr := NewGobEncoderLight()
	by, err := tr.EncodeType(inObject)
	if err != nil {
		return err
	}
	if l.batch != nil {
		l.batch.Put(l.getRealKey(id), by)
		return nil
	}
	return l.db.Put(l.getRealKey(id), by, nil)
}

func (l *levelDbHandle) DeleteObject(id []byte) error {
	if bytes.HasPrefix(id, bucketPrefix) || bytes.HasPrefix(id, rootBucketPrefix) {
		return fmt.Errorf("key uses reserved bucket notation: %v", string(id))
	}

	if l.batch != nil {
		l.batch.Delete(l.getRealKey(id))
		return nil
	}
	return l.db.Delete(l.getRealKey(id), nil)
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

		// Pass a modified key that preserves bucket structure but is usable
		// in debugging, etc.
		newKey := bytes.TrimPrefix(key, l.keyPrefix)

		if err := f(newKey, value); err != nil {
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
		newID = []byte(fmt.Sprintf("%s-%s-%s", string(bucketPrefix), string(l.prefix), id))
	} else {
		newID = []byte(fmt.Sprintf("%s-%s", string(bucketPrefix), id))
	}
	ret := &levelDbHandle{
		db:          l.db,
		prefix:      newID,
		keyPrefix:   []byte(fmt.Sprintf("%s-", string(newID))),
		prefixBytes: util.BytesPrefix(newID),
		batch:       l.batch,
	}
	return ret
}

func (l *levelDbHandle) View(f ReadOnlyFunc) error {
	return f(l)
}

// Update is a bit cheeky in that we create a clone of ourselves
// to utilise a batch object, and then execute the passed function
// within the context of that batch.
//
// If the function doesn't return an error, we'll allow the database
// to try and write. Otherwise, we'll discard the entire batch and
// return the functions error.
func (l *levelDbHandle) Update(f WriterFunc) error {
	clone := levelDbHandle{
		db:          l.db,
		prefix:      l.prefix,
		keyPrefix:   l.keyPrefix,
		prefixBytes: l.prefixBytes,
		batch:       &leveldb.Batch{},
	}
	err := f(&clone)
	if err != nil {
		clone.batch.Reset()
		return err
	}
	return clone.db.Write(clone.batch, nil)
}
