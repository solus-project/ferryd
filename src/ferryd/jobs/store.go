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

package jobs

import (
	"errors"
	"github.com/boltdb/bolt"
	"log"
)

var asyncJobs []byte
var syncJobs []byte
var jobStore []byte

// EmptyQueue occurs when trying to claim a job from an empty queue
var EmptyQueue error

func init() {
	asyncJobs = []byte("AsynchronousJobs")
	syncJobs = []byte("SynchronousJobs")
	jobStore = []byte("JobStore")
	EmptyQueue = errors.New("Queue is empty")
}

// JobStore handles the storage and manipulation of incomplete jobs
type JobStore struct {
	db *bolt.DB
}

// NewStore creates a fully initialized JobStore and sets up Bolt Buckets as needed
func NewStore(db *bolt.DB) (s *JobStore, err error) {
	s = &JobStore{db}
	err = s.setup()
	return
}

// Setup makes sure that all the necessary buckets exist and have valid contents
func (s *JobStore) setup() error {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	store, err := tx.CreateBucketIfNotExists(jobStore)
	if err != nil {
		goto FailSafe
	}
	_, err = store.CreateBucketIfNotExists(syncJobs)
	if err != nil {
		goto FailSafe
	}
	_, err = store.CreateBucketIfNotExists(asyncJobs)
	if err != nil {
		goto FailSafe
	}
	tx.Commit()
	return nil
FailSafe:
	tx.Rollback()
	return err
}

// ClaimAsyncJob gets the first available asynchronous job, if one exists
func (s *JobStore) ClaimAsyncJob() (id []byte, j JobEntry, err error) {
	tx, err := s.db.Begin(false)
	if err != nil {
		return
	}

	async := tx.Bucket(asyncJobs)
	cursor := async.Cursor()
	id, value := cursor.First()
	var newJ []byte
	for id != nil {
		j, err = Deserialize(value)
		if err != nil {
			log.Fatal(err.Error())
		}
		if !j.Claimed {
			j.Claimed = true
			newJ, err = j.Serialize()
			if err != nil {
				tx.Rollback()
				return
			}
			err = async.Put(id, newJ)
			if err != nil {
				tx.Commit()
				return
			}
		}
		id, value = cursor.Next()
	}
	tx.Rollback()
	return
}

// ClaimSyncJob gets the first available synchronous job, if one exists
func (s *JobStore) ClaimSyncJob() (id []byte, j JobEntry, err error) {
	tx, err := s.db.Begin(false)
	if err != nil {
		return
	}
	cursor := tx.Bucket(syncJobs).Cursor()
	id, value := cursor.First()
	if id == nil {
		err = EmptyQueue
		tx.Commit()
		return
	}
	j, err = Deserialize(value)
	tx.Commit()
	return
}

// RetireAsyncJob removes a completed asynchronous job
func (s *JobStore) RetireAsyncJob(id []byte) error {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	async := tx.Bucket(asyncJobs)
	return async.Delete(id)
}

// RetireSyncJob removes a completed synchronous job
func (s *JobStore) RetireSyncJob(id []byte) error {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	sync := tx.Bucket(syncJobs)
	return sync.Delete(id)
}
