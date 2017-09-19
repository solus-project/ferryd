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
	"strconv"
)

// RemoveSourceJobHandler is responsible for removing packages by identifiers
type RemoveSourceJobHandler struct {
	repoID  string
	source  string
	release int
}

// NewRemoveSourceJob will return a job suitable for adding to the job processor
func NewRemoveSourceJob(repoID, source string, release int) *JobEntry {
	return &JobEntry{
		sequential: true,
		Type:       RemoveSource,
		Params:     []string{repoID, source, fmt.Sprintf("%d", release)},
	}
}

// NewRemoveSourceJobHandler will create a job handler for the input job and ensure it validates
func NewRemoveSourceJobHandler(j *JobEntry) (*RemoveSourceJobHandler, error) {
	if len(j.Params) != 3 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	rel, err := strconv.ParseInt(j.Params[2], 10, 32)
	if err != nil {
		return nil, err
	}
	return &RemoveSourceJobHandler{
		repoID:  j.Params[0],
		source:  j.Params[1],
		release: int(rel),
	}, nil
}

// Execute will remove the source&rel match from the repo
func (j *RemoveSourceJobHandler) Execute(_ *Processor, manager *core.Manager) error {
	if err := manager.RemoveSource(j.repoID, j.source, j.release); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"repo":          j.repoID,
		"source":        j.source,
		"releaseNumber": j.release,
	}).Info("Removed source")
	return nil
}

// Describe returns a human readable description for this job
func (j *RemoveSourceJobHandler) Describe() string {
	return fmt.Sprintf("Remove sources by id '%s' (rel: %d) in '%s'", j.source, j.release, j.repoID)
}
