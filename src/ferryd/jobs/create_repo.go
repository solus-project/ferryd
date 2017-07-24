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

// CreateRepoJob is a sequential job which will attempt to create a new repo
type CreateRepoJob struct {
	repoID string
}

// NewCreateRepoJob will create a new job with the given ID
func NewCreateRepoJob(repoID string) *CreateRepoJob {
	return &CreateRepoJob{repoID: repoID}
}

// IsSequential will return true as the repo state must be sane in the server
func (c *CreateRepoJob) IsSequential() bool {
	return true
}

// Perform will invoke the indexing operation
func (c *CreateRepoJob) Perform(manager *core.Manager) error {
	if err := manager.CreateRepo(c.repoID); err != nil {
		return err
	}
	log.WithFields(log.Fields{"repo": c.repoID}).Info("Created repository")
	return nil
}

// Describe will explain the purpose of this job
func (c *CreateRepoJob) Describe() string {
	return fmt.Sprintf("Create new repository '%s'", c.repoID)
}
