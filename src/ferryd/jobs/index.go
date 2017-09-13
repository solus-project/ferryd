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

// IndexRepoJobHandler is responsible for indexing repositories and should only
// ever be used in sequential queues.
type IndexRepoJobHandler struct {
	repoID string
}

// NewIndexRepoJob will return a job suitable for adding to the job processor
func NewIndexRepoJob(id string) *JobEntry {
	return &JobEntry{
		sequential: true,
		Type:       IndexRepo,
		Params:     []string{id},
	}
}

// NewIndexRepoJobHandler will create a job handler for the input job and ensure it validates
func NewIndexRepoJobHandler(j *JobEntry) (*IndexRepoJobHandler, error) {
	if len(j.Params) != 1 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	return &IndexRepoJobHandler{
		repoID: j.Params[0],
	}, nil
}

// Execute will index the given repository if possible
func (j *IndexRepoJobHandler) Execute(_ *Processor, manager *core.Manager) error {
	if err := manager.Index(j.repoID); err != nil {
		return err
	}
	log.WithFields(log.Fields{"repo": j.repoID}).Info("Indexed repository")
	return nil
}

// Describe returns a human readable description for this job
func (j *IndexRepoJobHandler) Describe() string {
	return fmt.Sprintf("Index repository '%s'", j.repoID)
}
