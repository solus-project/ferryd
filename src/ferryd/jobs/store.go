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
	"libdb"
	"libferry"
	"sync"
	"time"
)

var (
	// BucketAsyncJobs holds all asynchronous jobs
	BucketAsyncJobs = []byte("Async")

	// BucketSequentialJobs holds all sequential jobs
	BucketSequentialJobs = []byte("Sync")

	// BucketSuccessJobs contains jobs that have completed successfully
	BucketSuccessJobs = []byte("CompletedSuccess")

	// BucketFailJobs contains jobs that completed with failure
	BucketFailJobs = []byte("CompletedFailure")

	// ErrEmptyQueue is returned to indicate a job is not available yet
	ErrEmptyQueue = errors.New("Queue is empty")

	// ErrBreakLoop is used only to break the foreach internally.
	ErrBreakLoop = errors.New("loop breaker")
)

// JobStore handles the storage and manipulation of incomplete jobs
type JobStore struct {
	db     libdb.Database
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

// setup is called during our early start to perform any relevant cleanup
// and repairs from previous runs.
func (s *JobStore) setup() error {
	if err := s.UnclaimSequential(); err != nil {
		return err
	}
	return s.UnclaimAsync()
}

// unclaimJobs will mark any previously claimed jobs as unclaimed again.
// This is only used during the initial start up ferryd as part of a
// recovery option
func (s *JobStore) unclaimJobs(bucketID []byte) error {
	s.modMut.Lock()
	defer s.modMut.Unlock()

	return s.db.Update(func(db libdb.Database) error {
		bucket := db.Bucket(bucketID)

		// Loop all claimed jobs, unclaim them
		return bucket.ForEach(func(id, value []byte) error {
			j := &JobEntry{}
			if err := bucket.Decode(value, j); err != nil {
				return err
			}
			if !j.Claimed {
				return nil
			}
			j.Timing.Begin = time.Time{}
			j.Timing.End = time.Time{}
			j.Claimed = false

			return bucket.PutObject(id, j)
		})
	})
}

// UnclaimSequential will find all claimed sequential jobs and unclaim them again
func (s *JobStore) UnclaimSequential() error {
	return s.unclaimJobs([]byte(BucketSequentialJobs))
}

// UnclaimAsync will find all claimed async jobs and unclaim them again
func (s *JobStore) UnclaimAsync() error {
	return s.unclaimJobs([]byte(BucketAsyncJobs))
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

	err := s.db.Update(func(db libdb.Database) error {
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

// Used to mark the completion of a job and store in the appropriate bucket
func (s *JobStore) markCompletion(j *JobEntry) error {
	var bucketID []byte
	if j.failure != nil {
		bucketID = BucketFailJobs
	} else {
		bucketID = BucketSuccessJobs
	}

	// We'll need to figure out how to truncate our buckets..
	return s.db.Update(func(db libdb.Database) error {
		bucket := db.Bucket(bucketID)
		nextID := db.NextSequence()

		storeJob := libferry.Job{
			Timing:      j.Timing,
			Description: j.description,
		}

		// Mark relevant failure fields
		if j.failure != nil {
			storeJob.Error = j.failure.Error()
			storeJob.Failed = true
		}

		return bucket.PutObject(nextID, &storeJob)
	})
}

// RetireAsyncJob removes a completed asynchronous job
func (s *JobStore) RetireAsyncJob(j *JobEntry) error {
	s.modMut.Lock()
	defer s.modMut.Unlock()

	err := s.db.Update(func(db libdb.Database) error {
		return db.Bucket(BucketAsyncJobs).DeleteObject(j.id)
	})

	if err != nil {
		return err
	}
	return s.markCompletion(j)
}

// RetireSequentialJob removes a completed synchronous job
func (s *JobStore) RetireSequentialJob(j *JobEntry) error {
	s.modMut.Lock()
	defer s.modMut.Unlock()

	err := s.db.Update(func(db libdb.Database) error {
		return db.Bucket(BucketSequentialJobs).DeleteObject(j.id)
	})

	if err != nil {
		return err
	}
	return s.markCompletion(j)
}

// pushJobInternal is identical between sync and async jobs, it
// just needs to know which bucket to store the job in.
func (s *JobStore) pushJobInternal(j *JobEntry, bk []byte) error {
	// Prep the job prior to insertion
	j.Timing.Queued = time.Now().UTC()
	j.Claimed = false

	j.id = s.db.Bucket(bk).NextSequence()

	s.modMut.Lock()
	defer s.modMut.Unlock()

	return s.db.Update(func(db libdb.Database) error {
		bucket := db.Bucket(bk)
		// Use next natural sequence in the bucket

		j.id = bucket.NextSequence()
		return bucket.PutObject(j.id, j)
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

// ActiveJobs will attempt to return a list of active jobs within
// the scheduler suitable for consumption by the CLI client
func (s *JobStore) ActiveJobs() ([]*libferry.Job, error) {
	var ret []*libferry.Job

	if err := s.cloneCurrentJobs(&ret, []byte(BucketSequentialJobs)); err != nil {
		return nil, err
	}

	if err := s.cloneCurrentJobs(&ret, []byte(BucketAsyncJobs)); err != nil {
		return nil, err
	}

	return ret, nil
}

// cloneCurrentJobs will push clones of our jobs out to the libferry API
func (s *JobStore) cloneCurrentJobs(ret *[]*libferry.Job, bucketID []byte) error {
	s.modMut.Lock()
	defer s.modMut.Unlock()

	return s.db.Bucket(bucketID).View(func(db libdb.ReadOnlyView) error {
		return db.ForEach(func(k, v []byte) error {
			j := &JobEntry{}
			if err := db.Decode(v, j); err != nil {
				return err
			}

			// Now stuff the job into the ret
			hnd, err := NewJobHandler(j)
			if err != nil {
				return err
			}

			r := &libferry.Job{
				Description: hnd.Describe(),
				Timing:      j.Timing,
			}
			*ret = append(*ret, r)

			return nil
		})
	})
}

// CompletedJobs will return all successfully completed jobs still stored
func (s *JobStore) CompletedJobs() ([]*libferry.Job, error) {
	var ret []*libferry.Job
	if err := s.clonePastJobs(&ret, BucketSuccessJobs); err != nil {
		return nil, err
	}
	return ret, nil
}

// FailedJobs will return all failed jobs that are still stored
func (s *JobStore) FailedJobs() ([]*libferry.Job, error) {
	var ret []*libferry.Job
	if err := s.clonePastJobs(&ret, BucketFailJobs); err != nil {
		return nil, err
	}
	return ret, nil
}

// clonePastJobs will pull the libferry.Job references from the DB and return
// clones
func (s *JobStore) clonePastJobs(ret *[]*libferry.Job, bucketID []byte) error {
	s.modMut.Lock()
	defer s.modMut.Unlock()

	return s.db.Bucket(bucketID).View(func(db libdb.ReadOnlyView) error {
		return db.ForEach(func(k, v []byte) error {
			j := &libferry.Job{}
			if err := db.Decode(v, j); err != nil {
				return err
			}
			*ret = append(*ret, j)

			return nil
		})
	})
}
