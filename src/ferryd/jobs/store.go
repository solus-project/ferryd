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
	"errors"
	"ferryd/core"
	"fmt"
	"github.com/google/uuid"
	"libdb"
	"os"
	"sync"
)

var (
	// BucketAsyncJobs holds all asynchronous jobs
	BucketAsyncJobs = []byte("Async")

	// BucketSequentialJobs holds all sequential jobs
	BucketSequentialJobs = []byte("Sync")

	// ErrEmptyQueue is returned to indicate a job is not available yet
	ErrEmptyQueue = errors.New("Queue is empty")

	// ErrBreakLoop is used only to break the foreach internally.
	ErrBreakLoop = errors.New("loop breaker")
)

// JobStore handles the storage and manipulation of incomplete jobs
type JobStore struct {
	db     *libdb.Database
	modMut *sync.Mutex
}

// NewStore creates a fully initialized JobStore and sets up Bolt Buckets as needed
func NewStore(path string) (*JobStore, error) {
	ctx, err := core.NewContext(path)

	// Open the database if we can
	db, err := libdb.Open(ctx.JobDbPath)
	if err != nil {
		return nil, err
	}

	s := &JobStore{
		db:     db,
		modMut: &sync.Mutex{},
	}

	if err := s.setup(); err != nil {
		defer s.Close()
		return nil, err
	}
	return s, nil
}

// Close will clean up our private job database
func (s *JobStore) Close() {
	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
}

// Setup may be used at a later stage to purge old jobs on startup
func (s *JobStore) setup() error {
	return nil
}

// claimJobInternal handles the similarity of the async/sync operations, grabbing
// the first available job and stuffing it back in as a claimed job. Note that
// in order to preserve order + sanity, we actually employ a mutex internally
// to mutate the state of each job, and return them sequentially.
//
// While more than one async job may be running at a time, we funnel job
// claim/retire calls.
func (s *JobStore) claimJobInternal(bucketID []byte) (*JobEntry, error) {
	s.modMut.Lock()
	defer s.modMut.Unlock()

	var job *JobEntry

	con, err := s.db.Connection()
	if err != nil {
		return nil, err
	}
	defer con.Close()

	err = con.Update(func(db libdb.DatabaseConnection) error {
		bucket := db.Bucket(bucketID)

		// Attempt to find relevant job, break when we have it + id
		err := bucket.ForEach(func(id, value []byte) error {
			j := &JobEntry{}
			if err := bucket.Decode(value, j); err != nil {
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
		return bucket.PutObject(job.id, job)
	})

	if err != nil {
		return nil, err
	}

	if job == nil {
		return nil, ErrEmptyQueue
	}

	return job, nil
}

// ClaimAsyncJob gets the first available asynchronous job, if one exists
func (s *JobStore) ClaimAsyncJob() (*JobEntry, error) {
	return s.claimJobInternal([]byte(BucketAsyncJobs))
}

// ClaimSequentialJob gets the first available synchronous job, if one exists
func (s *JobStore) ClaimSequentialJob() (*JobEntry, error) {
	return s.claimJobInternal([]byte(BucketSequentialJobs))
}

// RetireAsyncJob removes a completed asynchronous job
func (s *JobStore) RetireAsyncJob(j *JobEntry) error {
	s.modMut.Lock()
	defer s.modMut.Unlock()

	con, err := s.db.Connection()
	if err != nil {
		return err
	}
	defer con.Close()

	return con.Update(func(db libdb.DatabaseConnection) error {
		return db.Bucket(BucketAsyncJobs).DeleteObject(j.id)
	})
}

// RetireSequentialJob removes a completed synchronous job
func (s *JobStore) RetireSequentialJob(j *JobEntry) error {
	s.modMut.Lock()
	defer s.modMut.Unlock()

	con, err := s.db.Connection()
	if err != nil {
		return err
	}
	defer con.Close()

	return con.Update(func(db libdb.DatabaseConnection) error {
		return db.Bucket(BucketSequentialJobs).DeleteObject(j.id)
	})
}

func (s *JobStore) generateUUID(con libdb.DatabaseConnection) []byte {
	nTries := 0
	for nTries < 10 {
		u, err := uuid.NewRandom()
		if err != nil {
			nTries++
			fmt.Fprintf(os.Stderr, "UUID generation failure: %v\n", err)
			continue
		}
		b := []byte(u.String())
		// Skip used UUIDs..
		if has, _ := con.HasObject(b); has {
			nTries++
			fmt.Fprintf(os.Stderr, "The end is nigh! Duplicate UUID: %v\n", b)
			continue
		}
		return b
	}
	// Die here. We're fucked.
	panic("uuid generation completely failed")
}

// pushJobInternal is identical between sync and async jobs, it
// just needs to know which bucket to store the job in.
func (s *JobStore) pushJobInternal(j *JobEntry, bk []byte) error {
	j.Claimed = false

	con, err := s.db.Connection()
	if err != nil {
		return err
	}
	defer con.Close()

	j.id = s.generateUUID(con)

	s.modMut.Lock()
	defer s.modMut.Unlock()

	return con.Update(func(db libdb.DatabaseConnection) error {
		return db.Bucket(bk).PutObject(j.id, j)
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
