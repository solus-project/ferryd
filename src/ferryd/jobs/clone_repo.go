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

// CloneRepoJobHandler is responsible for cloning an existing repository
type CloneRepoJobHandler struct {
	repoID    string
	newClone  string
	cloneMode string
}

// NewCloneRepoJob will return a job suitable for adding to the job processor
func NewCloneRepoJob(repoID, newClone string, cloneAll bool) *JobEntry {
	mode := "tip"
	if cloneAll {
		mode = "full"
	}
	return &JobEntry{
		sequential: true,
		Type:       CloneRepo,
		Params:     []string{repoID, newClone, mode},
	}
}

// NewCloneRepoJobHandler will create a job handler for the input job and ensure it validates
func NewCloneRepoJobHandler(j *JobEntry) (*CloneRepoJobHandler, error) {
	if len(j.Params) != 3 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	return &CloneRepoJobHandler{
		repoID:    j.Params[0],
		newClone:  j.Params[1],
		cloneMode: j.Params[2],
	}, nil
}

// Execute attempt to clone the repoID to newClone, optionally at full depth
func (j *CloneRepoJobHandler) Execute(_ *Processor, manager *core.Manager) error {
	fullClone := false
	if j.cloneMode == "full" {
		fullClone = true
	}

	if err := manager.CloneRepo(j.repoID, j.newClone, fullClone); err != nil {
		return err
	}
	log.WithFields(log.Fields{"repo": j.repoID}).Info("Cloned repository")
	return nil
}

// Describe returns a human readable description for this job
func (j *CloneRepoJobHandler) Describe() string {
	return fmt.Sprintf("Clone repository '%s' into '%s'", j.repoID, j.newClone)
}
