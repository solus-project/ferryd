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
	"ferryd/core"
	"time"
)

const (
	// StatusFailed indicates run and since failed
	StatusFailed = 1 << iota

	// StatusSuccess indicates the task ran without issues
	StatusSuccess = 1 << iota

	// StatusPending indicates the task is still waiting to be run
	StatusPending = 1 << iota

	// StatusRunning indicates the task is currently active
	StatusRunning = 1 << iota
)

// A Runnable is the base interface that all Job tasks must implement to be
// run by the JobProcessor
type Runnable interface {

	// Perform will be called to let the Runnable perform its action using this manager
	// instance.
	Perform(m *core.Manager) error

	// IsSequential should return true for operations that can be performed on the
	// main job process. If the job is a heavyweight operation that should be run in
	// the background, it should return false (i.e. deltas)
	IsSequential() bool

	// Describe will request that the job identify itself in a meaningful way for
	// logging purposes
	Describe() string
}

// JobTiming provides simple timing information for a given task
type JobTiming struct {

	// The time at which the task was originally handed to the processor
	Created time.Time

	// The time at which the job processor began executing the task
	Started time.Time

	// The time at which the task completed
	Completed time.Time
}

// A Job is a unique tagged task that provides metadata about the Runnable
// and should never be directly instaniated by the user.
type Job struct {
	Task Runnable
	ID   string

	// Timing metrics
	Timing JobTiming

	// Current status of this task
	Status int
}
