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

// NewCreateRepoJob will return a new JobEntry specific to repo generation
func NewCreateRepoJob(repoID string) *JobEntry {
	return &JobEntry{
		Type:   CreateRepo,
		Params: []string{repoID},
	}
}

// CreateRepo will execute the CreateRepo function on the manager
func (j *JobEntry) CreateRepo(manager *core.Manager) error {
	repoID := j.Params[0]
	if err := manager.CreateRepo(repoID); err != nil {
		return err
	}
	log.WithFields(log.Fields{"repo": repoID}).Info("Created repository")
	return nil
}

// DescribeCreateRepo returns a description for the CreateRepo job
func (j *JobEntry) DescribeCreateRepo() string {
	return fmt.Sprintf("Create repository '%s'", j.Params[0])
}
