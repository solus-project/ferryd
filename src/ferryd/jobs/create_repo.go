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

// CreateRepoJobHandler is responsible for creating new repositories and should only
// ever be used in sequential queues.
type CreateRepoJobHandler struct {
	repoID string
}

// NewCreateRepoJob will return a job suitable for adding to the job processor
func NewCreateRepoJob(id string) *JobEntry {
	return &JobEntry{
		sequential: true,
		Type:       CreateRepo,
		Params:     []string{id},
	}
}

// NewCreateRepoJobHandler will create a job handler for the input job and ensure it validates
func NewCreateRepoJobHandler(j *JobEntry) (*CreateRepoJobHandler, error) {
	if len(j.Params) != 1 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	return &CreateRepoJobHandler{
		repoID: j.Params[0],
	}, nil
}

// Execute will construct a new repository if possible
func (j *CreateRepoJobHandler) Execute(_ *Processor, manager *core.Manager) error {
	if err := manager.CreateRepo(j.repoID); err != nil {
		return err
	}
	log.WithFields(log.Fields{"repo": j.repoID}).Info("Created repository")
	return nil
}

// Describe returns a human readable description for this job
func (j *CreateRepoJobHandler) Describe() string {
	return fmt.Sprintf("Create repository '%s'", j.repoID)
}
