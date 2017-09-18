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
	"runtime"
	"sync"
)

// DbForeachFunc is used in the root (untyped buckets)
type DbForeachFunc func(key, val []byte) error

// ReadOnlyView offers a read-only API for the database. Note it cannot
// gain access to buckets again, so you should obtain the view from the
// bucket.
type ReadOnlyView interface {
	// Get an object from storage
	GetObject(id []byte, o interface{}) error

	// Determine if an object with that ID exists already
	HasObject(id []byte) (bool, error)

	// Attempt to decode the input into the given output pointer
	Decode(input []byte, o interface{}) error

	// For every key value pair, run the given function
	ForEach(f DbForeachFunc) error
}

// WriterView allows destructive write actions within the database
type WriterView interface {

	// Delete an object from storage
	DeleteObject(id []byte) error

	// Put an object into storage (unique key)
	PutObject(id []byte, o interface{}) error
}

// A ReadOnlyFunc is expected by the Database.View method
type ReadOnlyFunc func(view ReadOnlyView) error

// A WriterFunc is used for batch write (transactional) views
type WriterFunc func(db DatabaseConnection) error

// DatabaseConnection is the compound interface to the underlying database implementation
type DatabaseConnection interface {
	ReadOnlyView
	WriterView

	// Return a subset of the database for usage
	Bucket(id []byte) DatabaseConnection

	// Convert view of current database or bucket into a read-only one.
	// This should not be considered a transaction, just special sauce.
	View(f ReadOnlyFunc) error

	// Obtain a read-write view of the database in a transaction
	Update(f WriterFunc) error

	// Close the connection (might no-op)
	Close()
}

// A Database is an opaque object that manages connections to the underlying
// database implementation. It simply provides two methods, Connection() and Close().
//
// Consumers should always close the main database itself when they are finished
// with it, and Close() every returned Connection() to ensure that all resources
// are actually freed. This is vital to ensure memory reclamation happens.
type Database struct {
	resourcePath string // Identifier of the database. We only support leveldb right now
	conMut       *sync.Mutex
	closeMut     *sync.Mutex
	handle       *levelDb
	refCount     int
	closed       bool
}

// Close the existing database
func (d *Database) Close() {
	d.closeMut.Lock()
	defer d.closeMut.Unlock()
	if d.closed {
		return
	}
	d.closed = true
	d.conMut.Lock()
	defer d.conMut.Unlock()
	if d.handle != nil {
		d.handle.consume()
		d.handle = nil
	}
}

// Open will return an opaque representation of the underlying database
// implementation suitable for usage within ferryd.
// A single connection will be opened and collected by way of a ping
func Open(path string) (*Database, error) {
	db := &Database{
		resourcePath: path,
		conMut:       &sync.Mutex{},
		closeMut:     &sync.Mutex{},
		handle:       nil,
		refCount:     0,
	}
	c, err := newLevelDBHandle(db, db.resourcePath)
	if err != nil {
		return nil, err
	}
	c.consume()
	return db, nil
}

// Connection will return a connection to the underlying database, and should
// be Close()'d when you are done with it.
func (d *Database) Connection() (DatabaseConnection, error) {
	d.conMut.Lock()
	defer d.conMut.Unlock()
	d.refCount++

	// Opening for the "first time"
	if d.refCount != 1 {
		return d.handle, nil
	}
	handle, err := newLevelDBHandle(d, d.resourcePath)
	if err != nil {
		d.refCount--
		return nil, err
	}
	d.handle = handle
	return handle, nil
}

// unref is called by a child connection when they've been closed
func (d *Database) unref() {
	d.conMut.Lock()
	defer d.conMut.Unlock()
	d.refCount--

	if d.refCount < 0 {
		panic("resource unref'd too many times!")
	}

	// Ideally this needs to be on an idle timer
	if d.refCount != 0 {
		return
	}

	// unref'd to 0, time to free the underlying handle
	if d.handle != nil {
		d.handle.consume()
		d.handle = nil
		// ferryd is a big bastid, GC here.
		runtime.GC()
	}
}
