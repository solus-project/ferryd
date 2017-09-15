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

package jobs

import (
	"encoding/binary"
	"errors"
	"ferryd/core"
	"github.com/boltdb/bolt"
	"sync"
)

var (
	// BucketAsyncJobs holds all asynchronous jobs
	BucketAsyncJobs = []byte("Async")

	// BucketSequentialJobs holds all sequential jobs
	BucketSequentialJobs = []byte("Sync")

	// BucketRootJobs is the parent job bucket
	BucketRootJobs = []byte("JobRoot")

	// ErrEmptyQueue is returned to indicate a job is not available yet
	ErrEmptyQueue = errors.New("Queue is empty")

	// ErrBreakLoop is used only to break the foreach internally.
	ErrBreakLoop = errors.New("loop breaker")
)

// JobStore handles the storage and manipulation of incomplete jobs
type JobStore struct {
	db     *bolt.DB
	jobMut *sync.Mutex
}

// NewStore creates a fully initialized JobStore and sets up Bolt Buckets as needed
func NewStore(path string) (*JobStore, error) {
	ctx, err := core.NewContext(path)

	// Open the database if we can
	// TODO: Add a timeout for locks
	db, err := bolt.Open(ctx.JobDbPath, 00600, nil)
	if err != nil {
		return nil, err
	}

	s := &JobStore{
		db:     db,
		jobMut: &sync.Mutex{},
	}

	if err := s.setup(); err != nil {
		defer s.Close()
		return nil, err
	}
	return s, nil
}

// Close will clean up our private job database
func (s *JobStore) Close() {
	s.jobMut.Lock()
	defer s.jobMut.Unlock()

	if s.db == nil {
		return
	}

	s.db.Close()
	s.db = nil
}

// Setup makes sure that all the necessary buckets exist and have valid contents
func (s *JobStore) setup() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		rootBucket, err := tx.CreateBucketIfNotExists(BucketRootJobs)
		if err != nil {
			return err
		}
		if _, err = rootBucket.CreateBucketIfNotExists(BucketAsyncJobs); err != nil {
			return err
		}
		if _, err = rootBucket.CreateBucketIfNotExists(BucketSequentialJobs); err != nil {
			return err
		}
		return nil
	})
}

// ClaimAsyncJob gets the first available asynchronous job, if one exists
func (s *JobStore) ClaimAsyncJob() (*JobEntry, error) {
	s.jobMut.Lock()
	defer s.jobMut.Unlock()

	var job *JobEntry

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketRootJobs).Bucket(BucketAsyncJobs)

		// Attempt to find relevant job, break when we have it + id
		err := bucket.ForEach(func(id, value []byte) error {
			j, err := Deserialize(value)
			if err != nil {
				return err
			}
			if !j.Claimed {
				j.Claimed = true
				// Got a usable job now.
				job = j
				job.id = make([]byte, len(id))
				copy(job.id, id)
				return ErrBreakLoop
			}
			return nil
		})

		if err != ErrBreakLoop {
			return err
		}

		// Serialise the new guy
		newJ, err := job.Serialize()
		if err != nil {
			return err
		}

		// Put the new guy back in
		return bucket.Put(job.id, newJ)
	})

	if err != nil {
		return nil, err
	}

	if job == nil {
		return nil, ErrEmptyQueue
	}

	return job, nil
}

// ClaimSequentialJob gets the first available synchronous job, if one exists
func (s *JobStore) ClaimSequentialJob() (*JobEntry, error) {
	s.jobMut.Lock()
	defer s.jobMut.Unlock()

	var job *JobEntry

	err := s.db.Update(func(tx *bolt.Tx) error {
		cursor := tx.Bucket(BucketRootJobs).Bucket(BucketSequentialJobs).Cursor()
		id, value := cursor.First()
		if id == nil {
			return ErrEmptyQueue
		}
		j, e := Deserialize(value)
		if e != nil {
			return e
		}
		// Store private ID field
		job = j
		job.id = make([]byte, len(id))
		copy(job.id, id)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return job, nil
}

// RetireAsyncJob removes a completed asynchronous job
func (s *JobStore) RetireAsyncJob(j *JobEntry) error {
	s.jobMut.Lock()
	defer s.jobMut.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(BucketRootJobs).Bucket(BucketAsyncJobs).Delete(j.id)
	})
}

// RetireSequentialJob removes a completed synchronous job
func (s *JobStore) RetireSequentialJob(j *JobEntry) error {
	s.jobMut.Lock()
	defer s.jobMut.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(BucketRootJobs).Bucket(BucketSequentialJobs).Delete(j.id)
	})
}

// pushJobInternal is identical between sync and async jobs, it
// just needs to know which bucket to store the job in.
func (s *JobStore) pushJobInternal(j *JobEntry, bk []byte) error {
	s.jobMut.Lock()
	defer s.jobMut.Unlock()

	j.Claimed = false

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketRootJobs).Bucket(bk)
		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}
		// Adapted from boltdb itob example code
		j.id = make([]byte, 8)
		binary.BigEndian.PutUint64(j.id, uint64(id))
		blob, err := j.Serialize()
		if err != nil {
			return err
		}
		return bucket.Put(j.id, blob)
	})
}

// PushSequentialJob will enqueue a new sequential job
func (s *JobStore) PushSequentialJob(j *JobEntry) error {
	return s.pushJobInternal(j, BucketSequentialJobs)
}

// PushAsyncJob will enqueue a new asynchronous job
func (s *JobStore) PushAsyncJob(j *JobEntry) error {
	return s.pushJobInternal(j, BucketAsyncJobs)
}
