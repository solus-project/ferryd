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

// CopySourceJobHandler is responsible for removing packages by identifiers
type CopySourceJobHandler struct {
	repoID  string
	target  string
	source  string
	release int
}

// NewCopySourceJob will return a job suitable for adding to the job processor
func NewCopySourceJob(repoID, target, source string, release int) *JobEntry {
	return &JobEntry{
		sequential: true,
		Type:       CopySource,
		Params:     []string{repoID, target, source, fmt.Sprintf("%d", release)},
	}
}

// NewCopySourceJobHandler will create a job handler for the input job and ensure it validates
func NewCopySourceJobHandler(j *JobEntry) (*CopySourceJobHandler, error) {
	if len(j.Params) != 4 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	rel, err := strconv.ParseInt(j.Params[3], 10, 32)
	if err != nil {
		return nil, err
	}
	return &CopySourceJobHandler{
		repoID:  j.Params[0],
		target:  j.Params[1],
		source:  j.Params[2],
		release: int(rel),
	}, nil
}

// Execute will copy the source&rel match from the repo to the target
func (j *CopySourceJobHandler) Execute(_ *Processor, manager *core.Manager) error {
	if err := manager.CopySource(j.repoID, j.target, j.source, j.release); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"from":          j.repoID,
		"to":            j.target,
		"source":        j.source,
		"releaseNumber": j.release,
	}).Info("Removed source")
	return nil
}

// Describe returns a human readable description for this job
func (j *CopySourceJobHandler) Describe() string {
	return fmt.Sprintf("Copy sources by id '%s' (rel: %d) in '%s' to '%s'", j.source, j.release, j.repoID, j.target)
}
