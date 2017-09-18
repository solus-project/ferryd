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

// DeleteRepoJobHandler is responsible for creating new repositories and should only
// ever be used in sequential queues.
type DeleteRepoJobHandler struct {
	repoID string
}

// NewDeleteRepoJob will return a job suitable for adding to the job processor
func NewDeleteRepoJob(id string) *JobEntry {
	return &JobEntry{
		sequential: true,
		Type:       DeleteRepo,
		Params:     []string{id},
	}
}

// NewDeleteRepoJobHandler will create a job handler for the input job and ensure it validates
func NewDeleteRepoJobHandler(j *JobEntry) (*DeleteRepoJobHandler, error) {
	if len(j.Params) != 1 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	return &DeleteRepoJobHandler{
		repoID: j.Params[0],
	}, nil
}

// Execute will construct a new repository if possible
func (j *DeleteRepoJobHandler) Execute(_ *Processor, manager *core.Manager) error {
	if err := manager.DeleteRepo(j.repoID); err != nil {
		return err
	}
	log.WithFields(log.Fields{"repo": j.repoID}).Info("Deleted repository")
	return nil
}

// Describe returns a human readable description for this job
func (j *DeleteRepoJobHandler) Describe() string {
	return fmt.Sprintf("Delete repository '%s'", j.repoID)
}
