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

// BulkAddJobHandler is responsible for indexing repositories and should only
// ever be used in sequential queues.
type BulkAddJobHandler struct {
	repoID       string
	packagePaths []string
}

// NewBulkAddJob will return a job suitable for adding to the job processor
func NewBulkAddJob(id string, pkgs []string) *JobEntry {
	return &JobEntry{
		sequential: true,
		Type:       BulkAdd,
		Params:     append([]string{id}, pkgs...),
	}
}

// NewBulkAddJobHandler will create a job handler for the input job and ensure it validates
func NewBulkAddJobHandler(j *JobEntry) (*BulkAddJobHandler, error) {
	if len(j.Params) < 2 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	return &BulkAddJobHandler{
		repoID:       j.Params[0],
		packagePaths: j.Params[1:],
	}, nil
}

// Execute will attempt the mass-import of packages passed to the job
func (j *BulkAddJobHandler) Execute(_ *Processor, manager *core.Manager) error {
	if err := manager.AddPackages(j.repoID, j.packagePaths); err != nil {
		return err
	}
	log.WithFields(log.Fields{"repo": j.repoID}).Info("Added packages to repository")
	return nil
}

// Describe returns a human readable description for this job
func (j *BulkAddJobHandler) Describe() string {
	return fmt.Sprintf("Add %v packages to repository '%s'", len(j.packagePaths), j.repoID)
}
