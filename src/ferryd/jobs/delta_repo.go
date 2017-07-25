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
	"fmt"
	log "github.com/sirupsen/logrus"
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

// Perform will iterate all package names within the given repo, and then create
// one delta task per package *name*, not per package *set*.
func (d *DeltaRepoJob) Perform(manager *core.Manager) error {
	packageNames, err := manager.GetPackageNames(d.repoID)
	if err != nil {
		return err
	}

	var js []*Job
	var indexJob *Job

	for _, name := range packageNames {
		j := d.jproc.PushJobLater(NewDeltaPackageJob(d.repoID, name))
		js = append(js, j)
	}

	// Start all of our delta jobs
	for _, j := range js {
		if indexJob == nil {
			indexJob = d.jproc.PushJobLater(NewIndexJob(d.repoID))
		}
		indexJob.AddDependency(j)
		go d.jproc.StartJob(j)
	}

	if len(js) < 1 {
		log.WithFields(log.Fields{
			"repo": d.repoID,
		}).Warning("Requested delta for empty repository")
	}

	return nil
}

// Describe will explain the purpose of this job
func (d *DeltaRepoJob) Describe() string {
	return fmt.Sprintf("Produce deltas for '%s'", d.repoID)
}
