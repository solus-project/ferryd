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
	"runtime"
	"sync"
)

// A Processor is responsible for the main dispatch and bulking of jobs
// to ensure they're handled in the most optimal fashion.
type Processor struct {
	manager *core.Manager
	store   *JobStore
	wg      *sync.WaitGroup
	closed  bool
	njobs   int
	workers []*Worker
}

// NewProcessor will return a new Processor with the specified number
// of jobs. Note that "njobs" only refers to the number of *background jobs*,
// the majority of operations will run sequentially
func NewProcessor(m *core.Manager, store *JobStore, njobs int) *Processor {
	// If we set to -1, we'll automatically set to half of the system core count
	// because we use xz -T 2 (so twice the number of threads ..)
	if njobs < 0 {
		njobs = runtime.NumCPU() / 2
	}

	if njobs < 2 {
		njobs = runtime.NumCPU()
	}

	oldJobs := runtime.GOMAXPROCS(njobs + 5)
	// Don't intentionally break things.
	if oldJobs < njobs+5 {
		oldJobs = runtime.GOMAXPROCS(oldJobs)
	}

	log.WithFields(log.Fields{
		"jobs":        njobs,
		"oldMaxProcs": oldJobs,
		"maxProcs":    njobs + 5,
	}).Info("Set runtime job limits")

	ret := &Processor{
		manager: m,
		store:   store,
		wg:      &sync.WaitGroup{},
		closed:  false,
		njobs:   njobs,
	}

	// Construct worker pool
	ret.workers = append(ret.workers, NewWorkerSequential(ret))
	for i := 0; i < njobs; i++ {
		ret.workers = append(ret.workers, NewWorkerAsync(ret))
	}

	return ret
}

// Close an existing Processor, waiting for all jobs to complete
func (j *Processor) Close() {
	if j.closed {
		return
	}
	j.closed = true

	// Close all of our workers
	for _, j := range j.workers {
		j.Stop()
	}

	j.wg.Wait()
}

// Begin will start the main job processor in parallel
func (j *Processor) Begin() {
	if j.closed {
		return
	}
	j.wg.Add(j.njobs + 1)
	for _, j := range j.workers {
		go j.Start()
	}
}

// PushJob will automatically determine which queue to push a job to and place
// it there for immediate execution
func (j *Processor) PushJob(job *JobEntry) {
	if job.sequential {
		j.store.PushSequentialJob(job)
	} else {
		j.store.PushAsyncJob(job)
	}
}
