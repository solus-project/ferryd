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
	"sync"
	"time"
)

var (
	// timeIndexes allow us to gradually increase our sleep duration
	timeIndexes = []time.Duration{
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
	store      *JobStore
	timeIndex  int // Increment time index to match timeIndexes, or wrap
}

// newWorker is an internal method to initialise a worker for usage
func newWorker(store *JobStore, wg *sync.WaitGroup, sequential bool) *Worker {
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
		store:      store,
		timeIndex:  0,
	}
	return w
}

// NewWorkerAsync will return an asynchronous processing worker which will only
// pull from the store's async job queue
func NewWorkerAsync(store *JobStore, wg *sync.WaitGroup) *Worker {
	return newWorker(store, wg, false)
}

// NewWorkerSequential will return a sequential worker operating on the main
// sequential job loop
func NewWorkerSequential(store *JobStore, wg *sync.WaitGroup) *Worker {
	return newWorker(store, wg, true)
}

// Stop will demand that all new requests are no longer processed
func (w *Worker) Stop() {
	w.exit <- 1
	if w.ticker != nil {
		w.ticker.Stop()
		w.ticker = nil
	}
}
