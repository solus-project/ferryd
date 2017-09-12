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
	"fmt"
	"os"
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
	if njobs < 0 {
		njobs = runtime.NumCPU()
	}

	fmt.Fprintf(os.Stderr, "Capped backgroundJobs to %d\n", njobs)

	ret := &Processor{
		manager: m,
		store:   store,
		wg:      &sync.WaitGroup{},
		closed:  false,
		njobs:   njobs,
	}

	// Construct worker pool (TODO: Get the store from *somewhere* ..
	ret.workers = append(ret.workers, NewWorkerSequential(ret.manager, ret.store, ret.wg))
	for i := 0; i < njobs; i++ {
		ret.workers = append(ret.workers, NewWorkerAsync(ret.manager, ret.store, ret.wg))
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
