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
	"sync"
)

// JobType is a numerical representation of a kind of job
type JobType uint8

const (
	BulkAdd JobType = iota
	CreateRepo
	Delta
	DeltaRepo
	TransitProcess
)

// JobEntry is an entry in the JobQueue
type JobEntry struct {
	ID   int64
	Type JobType
}

// JobQueue is a FIFO queue for synchronous tasks
type JobQueue struct {
	sync.Mutex
	jobs []JobEntry
}

var dequeueFail error
var frontFail error

func init() {
	dequeueFail = errors.New("Could not dequeue Job, queue is empty")
	frontFail = errors.New("Could not read front Job, queue is empty")
}

// NewQueue creates a fully initialized, empty queue
func NewQueue() *JobQueue {
	return &JobQueue{
		jobs: make([]JobEntry, 0),
	}
}

// Dequeue places a new job at the end of the queue
func (q *JobQueue) Dequeue() (j JobEntry, err error) {
	if len(q.jobs) == 0 {
		err = dequeueFail
		return
	}
	q.Lock()
	j = q.jobs[0]
	q.jobs = q.jobs[1:]
	q.Unlock()
	return
}

// Enqueue places a new job at the end of the queue
func (q *JobQueue) Enqueue(j JobEntry) {
	q.Lock()
	q.jobs = append(q.jobs, j)
	q.Unlock()
}

// Front reads the first job from the queue
func (q *JobQueue) Front() (j JobEntry, err error) {
	if len(q.jobs) == 0 {
		err = frontFail
		return
	}
	j = q.jobs[0]
	return
}
