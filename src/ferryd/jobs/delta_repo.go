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
	"ferryd/core"
	"fmt"
)

// DeltaRepoJob is a sequential job which will attempt to create a new repo
type DeltaRepoJob struct {
	repoID string
	jproc  *Processor
}

// NewDeltaRepoJob will create a new job with the given ID
func NewDeltaRepoJob(repoID string) *DeltaRepoJob {
	return &DeltaRepoJob{repoID: repoID}
}

// Init allows the job to store a reference to the job processor internally
// to dispatch further non sequential jobs
func (d *DeltaRepoJob) Init(jproc *Processor) {
	d.jproc = jproc
}

// IsSequential will return true as the repo state must be sane in the server
func (d *DeltaRepoJob) IsSequential() bool {
	return true
}

// Perform will invoke the indexing operation
func (d *DeltaRepoJob) Perform(manager *core.Manager) error {
	return errors.New("Not yet implemented")
}

// Describe will explain the purpose of this job
func (d *DeltaRepoJob) Describe() string {
	return fmt.Sprintf("Produce deltas for '%s'", d.repoID)
}
