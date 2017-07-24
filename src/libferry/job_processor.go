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

package libferry

import (
	"fmt"
	"os"
	"sync"
)

// A Job is exactly what it looks like, the base operation type that we'll deal
// with.
type Job interface {

	// Perform will be called to let the Job perform its action using this manager
	// instance.
	Perform(m *Manager) error

	// IsSequential should return true for operations that can be performed on the
	// main job process. If the job is a heavyweight operation that should be run in
	// the background, it should return false (i.e. deltas)
	IsSequential() bool
}

// A JobProcessor is responsible for the main dispatch and bulking of jobs
// to ensure they're handled in the most optimal fashion.
type JobProcessor struct {
	manager        *Manager
	jobs           chan Job
	backgroundJobs chan Job
	quit           chan bool
	mut            *sync.Mutex
	wg             *sync.WaitGroup
	closed         bool
	njobs          int
}

// NewJobProcessor will return a new JobProcessor with the specified number
// of jobs. Note that "njobs" only refers to the number of *background jobs*,
// the majority of operations will run sequentially
func NewJobProcessor(m *Manager, njobs int) *JobProcessor {
	ret := &JobProcessor{
		manager:        m,
		jobs:           make(chan Job),
		backgroundJobs: make(chan Job),
		quit:           make(chan bool, 1+njobs),
		mut:            &sync.Mutex{},
		wg:             &sync.WaitGroup{},
		closed:         false,
		njobs:          njobs,
	}
	return ret
}

// Close an existing JobProcessor, waiting for all jobs to complete
func (j *JobProcessor) Close() {
	j.mut.Lock()
	defer j.mut.Unlock()
	if j.closed {
		return
	}

	// Disallow further messaging
	close(j.jobs)
	close(j.backgroundJobs)

	// Ensure all goroutines get the quit broadcast
	for i := 0; i < j.njobs+1; i++ {
		j.quit <- true
	}
	j.wg.Wait()
}

// Begin will start the main job processor in parallel
func (j *JobProcessor) Begin() {
	j.mut.Lock()
	defer j.mut.Unlock()
	if j.closed {
		return
	}
	j.wg.Add(2)
	go j.processSequentialQueue()
	go j.processBackgroundQueue()
}

// processSequentialQueue is responsible for dealing with the sequential queue
func (j *JobProcessor) processSequentialQueue() {
	defer j.wg.Done()

	for {
		select {
		case job := <-j.jobs:
			// TODO: Add proper logging for jobs
			if err := job.Perform(j.manager); err != nil {
				fmt.Fprintf(os.Stderr, "Job failed to run: %v\n", j)
			}
		case <-j.quit:
			return
		}
	}
}

// processBackgroundQueue will set up the background workers which will block
// waiting for non-sequential work that cannot run on the main queue, however
// it may put work back on the sequential queue.
func (j *JobProcessor) processBackgroundQueue() {
	defer j.wg.Done()
	j.wg.Add(j.njobs)

	for i := 0; i < j.njobs; i++ {
		go j.backgroundWorker()
	}
}

// backgroundWorker will handle the non sequential tasks as and when they come
// in. The majority of tasks will be sequential on the main queue, so we're free
// to spend more CPU time here dealing with large tasks like the construction
// of delta packages.
func (j *JobProcessor) backgroundWorker() {
	defer j.wg.Done()

	for {
		select {
		case job := <-j.backgroundJobs:
			// TODO: Add proper logging for jobs
			if err := job.Perform(j.manager); err != nil {
				fmt.Fprintf(os.Stderr, "Job failed to run: %v\n", j)
			}
		case <-j.quit:
			return
		}
	}
}

// PushJob will take the new job and push it to the appropriate queing system
// For sanity reasons this will lock on the new job add, even if the processing
// is then parallel.
func (j *JobProcessor) PushJob(job Job) {
	j.mut.Lock()
	defer j.mut.Unlock()

	if j == nil {
		panic("passed nil job!")
	}

	// TODO: Add descriptions to the Job type and emit to the log
	if job.IsSequential() {
		j.jobs <- job
	} else {
		j.backgroundJobs <- job
	}
}
