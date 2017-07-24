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
	log "github.com/sirupsen/logrus"
	"libferry"
)

// IndexJob is a sequential job which will cause the given repository to be
// atomically reindexed.
type IndexJob struct {
	repoID string
}

// NewIndexJob will create a new indexing job for the given repository ID
func NewIndexJob(repoID string) *IndexJob {
	return &IndexJob{repoID: repoID}
}

// IsSequential will return true as the index must be written atomically
func (i *IndexJob) IsSequential() bool {
	return true
}

// Perform will invoke the indexing operation
func (i *IndexJob) Perform(manager *libferry.Manager) error {
	if err := manager.Index(i.repoID); err != nil {
		return err
	}
	log.WithFields(log.Fields{"repo": i.repoID}).Info("Indexed repository")
	return nil
}
