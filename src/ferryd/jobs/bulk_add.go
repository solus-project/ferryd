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

// BulkAddJob is a sequential job which will attempt to add all of the packages
// listed in bulk to the repository.
type BulkAddJob struct {
	repoID       string
	packagePaths []string
}

// NewBulkAddJob will create a new job for the given repo and packages
func NewBulkAddJob(repoID string, packagePaths []string) *BulkAddJob {
	return &BulkAddJob{repoID: repoID, packagePaths: packagePaths}
}

// IsSequential will return true as we're going to need to index after
func (i *BulkAddJob) IsSequential() bool {
	return true
}

// Perform will invoke the operation
func (i *BulkAddJob) Perform(manager *core.Manager) error {
	if err := manager.AddPackages(i.repoID, i.packagePaths); err != nil {
		return err
	}

	log.WithFields(log.Fields{"repo": i.repoID}).Info("Added bulk packages")
	return nil
}

// Describe will explain the purpose of this job
func (i *BulkAddJob) Describe() string {
	return fmt.Sprintf("Bulk add %d packages to '%s'", len(i.packagePaths), i.repoID)
}
