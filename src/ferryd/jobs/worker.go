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
	"ferryd/core"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// JobFetcher will be provided by either the Async or Sequential claim functions
type JobFetcher func() (*JobEntry, error)

// JobReaper will be provided by either the Async or Sequential retire functions
type JobReaper func(j *JobEntry) error

// JobExecutor is the main processing function
type JobExecutor func(m *core.Manager) error

// JobDescription is used to describe simple types
type JobDescription func() string

var (
	// timeIndexes allow us to gradually increase our sleep duration
	timeIndexes = []time.Duration{
		time.Millisecond * 500,
		time.Second * 1,
		time.Second * 5,
		time.Second * 10,
		time.Second * 20,
		time.Second * 40,
		time.Second * 60,
	}
)

// A Worker is used to execute some portion of the incoming workload, and will
// keep polling for the correct job type to process
type Worker struct {
	sequential bool
	exit       chan int
	ticker     *time.Ticker
	wg         *sync.WaitGroup
	manager    *core.Manager
	store      *JobStore
	timeIndex  int // Increment time index to match timeIndexes, or wrap

	fetcher JobFetcher // Fetch a new job
	reaper  JobReaper  // Purge an old job
}

// newWorker is an internal method to initialise a worker for usage
func newWorker(manager *core.Manager, store *JobStore, wg *sync.WaitGroup, sequential bool) *Worker {
	if store == nil {
		panic("Constructed a Worker without a valid JobStore!")
	}
	if wg == nil {
		panic("Constructed a Worker without a valid WaitGroup!")
	}

	w := &Worker{
		sequential: sequential,
		wg:         wg,
		exit:       make(chan int, 1),
		ticker:     nil, // Init this when we start up
		manager:    manager,
		store:      store,
		timeIndex:  -1,
	}

	// Set up appropriate functions for dealing with jobs
	if sequential {
		w.fetcher = w.store.ClaimSequentialJob
		w.reaper = w.store.RetireSequentialJob
	} else {
		w.fetcher = w.store.ClaimAsyncJob
		w.reaper = w.store.RetireAsyncJob
	}

	return w
}

// NewWorkerAsync will return an asynchronous processing worker which will only
// pull from the store's async job queue
func NewWorkerAsync(manager *core.Manager, store *JobStore, wg *sync.WaitGroup) *Worker {
	return newWorker(manager, store, wg, false)
}

// NewWorkerSequential will return a sequential worker operating on the main
// sequential job loop
func NewWorkerSequential(manager *core.Manager, store *JobStore, wg *sync.WaitGroup) *Worker {
	return newWorker(manager, store, wg, true)
}

// Stop will demand that all new requests are no longer processed
func (w *Worker) Stop() {
	w.exit <- 1
	if w.ticker != nil {
		w.ticker.Stop()
		w.ticker = nil
	}
}

// Start will begin the main execution of this worker, and will continuously
// poll for new jobs with an increasing increment (with a ceiling limit)
func (w *Worker) Start() {
	defer w.wg.Done()

	// Let's get our ticker initialised
	w.setTimeIndex(0)

	for {
		select {
		case <-w.exit:
			// Bail now, we've been told to go home
			return

		case <-w.ticker.C:
			// Try to grab a job
			job, err := w.fetcher()

			// Report the error
			if err != nil {
				if err == ErrEmptyQueue {
					// Bump the time index
					log.WithFields(log.Fields{
						"async": !w.sequential,
					}).Debug("Empty processing queue")
				} else {
					log.WithFields(log.Fields{
						"error": err,
						"async": !w.sequential,
					}).Error("Failed to grab a work queue item")
				}
				w.setTimeIndex(w.timeIndex + 1)
				continue
			}

			// Got a job, now process it
			w.processJob(job)

			// Mark the job as dealt with
			err = w.reaper(job)

			// Report failure in retiring the job
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"id":    job.GetID(),
					"type":  job.Type,
					"async": !w.sequential,
				}).Error("Error in retiring job")
			}

			// We had a job, so we must reset the timeout period
			w.setTimeIndex(0)
		}
	}
}

// setTimeIndex will update the time index, and reset the ticker if needed
// so that we increment the wait period. It will cap the time index to the
// highest index available (60)
func (w *Worker) setTimeIndex(newTimeIndex int) {
	if newTimeIndex >= len(timeIndexes) {
		newTimeIndex = len(timeIndexes) - 1
	}
	// No sense resetting our ticker
	if w.timeIndex == newTimeIndex {
		return
	}
	w.timeIndex = newTimeIndex
	if w.ticker != nil {
		w.ticker.Stop()
	}
	w.ticker = time.NewTicker(timeIndexes[w.timeIndex])
	log.WithFields(log.Fields{
		"async":    !w.sequential,
		"duration": timeIndexes[w.timeIndex],
	}).Debug("Updated worker wait period")
}

// processJob will actually examine the given job and figure out how
// to execute it. Each Worker can only execute a single job at a time
func (w *Worker) processJob(job *JobEntry) {
	var exec JobExecutor
	var desc JobDescription

	switch job.Type {
	case CreateRepo:
		exec = job.CreateRepo
		desc = job.DescribeCreateRepo
	default:
		exec = nil
	}

	fields := log.Fields{
		"id":          job.GetID(),
		"type":        job.Type,
		"async":       !w.sequential,
		"description": desc(),
	}

	if exec == nil {
		log.WithFields(fields).Error("No known job handler, cannot continue with job")
		return
	}

	// Try to execute it, report the error
	if err := exec(w.manager); err != nil {
		fields["error"] = err
		log.WithFields(fields).Error("Job failed with error")
		return
	}

	// Succeeded
	log.WithFields(fields).Info("Job completed successfully")
}
