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
	log "github.com/sirupsen/logrus"
	"sync"
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

	// Allow the job to initialise itself and store any references to the job
	// processor for further dispatches
	Init(jproc *Processor)

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
	// Public ID
	id string

	// Timing metrics
	timing JobTiming

	// Current status of this task
	status int

	// Private task (not serialised)
	task Runnable

	dependents map[*Job]int
	parents    map[*Job]int
	depMut     *sync.RWMutex

	// We can only be freed by a single child completing
	claimed bool
}

// Internal helper to set a job useful™
func (j *Job) init() {
	j.dependents = make(map[*Job]int)
	j.parents = make(map[*Job]int)
	j.depMut = &sync.RWMutex{}
	j.claimed = false
}

// AddDependency will attempt to add the child @job as a dependency of this
// job.
func (j *Job) AddDependency(child *Job) {
	j.depMut.Lock()
	defer j.depMut.Unlock()

	if _, ok := j.dependents[child]; ok {
		log.WithFields(log.Fields{
			"child_id": child.id,
			"id":       j.id,
		}).Error("Attempted to re-add dependent child")
		return
	}

	j.dependents[child] = 1
	child.addParent(j)
}

// PopDependency will remove the dependency from the job
func (j *Job) PopDependency(child *Job) {
	j.depMut.Lock()
	defer j.depMut.Unlock()

	if _, ok := j.dependents[child]; !ok {
		log.WithFields(log.Fields{
			"child_id": child.id,
			"id":       j.id,
		}).Error("Attempted to remove invalid dependency")
		return
	}

	delete(j.dependents, child)
	child.popParent(j)
}

// addParent will add a parent dependent task
func (j *Job) addParent(parent *Job) {
	j.depMut.Lock()
	defer j.depMut.Unlock()

	if _, ok := j.parents[parent]; ok {
		log.WithFields(log.Fields{
			"parent_id": parent.id,
			"id":        j.id,
		}).Error("Attempted to re-add parent")
		return
	}

	j.parents[parent] = 1
}

// popParent will remove the parent from the child
func (j *Job) popParent(parent *Job) {
	j.depMut.Lock()
	defer j.depMut.Unlock()

	if _, ok := j.parents[parent]; !ok {
		log.WithFields(log.Fields{
			"parent_id": parent.id,
			"id":        j.id,
		}).Error("Attempted to remove invalid parent")
		return
	}

	delete(j.parents, parent)
}

// HasDependencies determines whether this task has any dependencies set up
// Note that a tasks dependencies must be set up BEFORE pushing to the job
// scheduler, otherwise it will never be pushed for execution!
func (j *Job) HasDependencies() bool {
	j.depMut.RLock()
	defer j.depMut.RUnlock()
	return len(j.dependents) > 0
}

// childNotify is used by a child job to notify the parent job that it is now
// complete. If this is the last dependent child we'll return TRUE so that
// we can now be processed too
func (j *Job) childNotify(child *Job) bool {
	j.PopDependency(child)
	j.depMut.Lock()
	defer j.depMut.Unlock()
	if !j.claimed {
		if len(j.dependents) == 0 {
			j.claimed = true
			return true
		}
	}
	return false
}

// NotifyDone will be used for the task to indicate that it is done, and will
// return a list of parent tasks that are now freed
func (j *Job) NotifyDone() []*Job {
	var parents []*Job
	j.depMut.RLock()
	for parent := range j.parents {
		parents = append(parents, parent)
	}
	j.depMut.RUnlock()

	var parentDone []*Job

	for _, parent := range parents {
		if parent.childNotify(j) {
			parentDone = append(parentDone, parent)
		}
	}
	return parentDone
}
