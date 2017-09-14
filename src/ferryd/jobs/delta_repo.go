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
	log "github.com/sirupsen/logrus"
)

// DeltaRepoJobHandler is responsible for delta'ing repositories and should only
// ever be used in sequential queues.
type DeltaRepoJobHandler struct {
	repoID string
}

// NewDeltaRepoJob will return a job suitable for adding to the job processor
func NewDeltaRepoJob(id string) *JobEntry {
	return &JobEntry{
		sequential: true,
		Type:       DeltaRepo,
		Params:     []string{id},
	}
}

// NewDeltaRepoJobHandler will create a job handler for the input job and ensure it validates
func NewDeltaRepoJobHandler(j *JobEntry) (*DeltaRepoJobHandler, error) {
	if len(j.Params) != 1 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	return &DeltaRepoJobHandler{
		repoID: j.Params[0],
	}, nil
}

// Execute will delta the given repository if possible
// Note that it will NOT index the repository, this is a separate step
// as it takes a significant amount of time to fully produce all initial
// deltas.
//
// This operation is ideally only used after the first import of a repository,
// after then deltas will be produced on the fly.
func (j *DeltaRepoJobHandler) Execute(jproc *Processor, manager *core.Manager) error {
	packageNames, err := manager.GetPackageNames(j.repoID)
	if err != nil {
		return err
	}

	// Skip an empty repository
	if len(packageNames) < 1 {
		log.WithFields(log.Fields{
			"repo": j.repoID,
		}).Warning("Requested delta for empty repository")
		return nil
	}

	// Fire off parallel delta jobs for every package in this repository
	for _, name := range packageNames {
		jproc.PushJob(NewDeltaJob(j.repoID, name))
	}

	return nil
}

// Describe returns a human readable description for this job
func (j *DeltaRepoJobHandler) Describe() string {
	return fmt.Sprintf("Produce deltas for '%s'", j.repoID)
}
