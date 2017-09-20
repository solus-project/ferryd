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

// TrimPackagesJobHandler is responsible for removing packages by identifiers
type TrimPackagesJobHandler struct {
	repoID  string
	maxKeep int
}

// NewTrimPackagesJob will return a job suitable for adding to the job processor
func NewTrimPackagesJob(repoID string, maxKeep int) *JobEntry {
	return &JobEntry{
		sequential: true,
		Type:       TrimPackages,
		Params:     []string{repoID, fmt.Sprintf("%d", maxKeep)},
	}
}

// NewTrimPackagesJobHandler will create a job handler for the input job and ensure it validates
func NewTrimPackagesJobHandler(j *JobEntry) (*TrimPackagesJobHandler, error) {
	if len(j.Params) != 2 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	keep, err := strconv.ParseInt(j.Params[1], 10, 32)
	if err != nil {
		return nil, err
	}
	return &TrimPackagesJobHandler{
		repoID:  j.Params[0],
		maxKeep: int(keep),
	}, nil
}

// Execute will attempt removal of excessive packages in the index
func (j *TrimPackagesJobHandler) Execute(_ *Processor, manager *core.Manager) error {
	if err := manager.TrimPackages(j.repoID, j.maxKeep); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"repo":    j.repoID,
		"maxKeep": j.maxKeep,
	}).Info("Trimmed packages in repository")
	return nil
}

// Describe returns a human readable description for this job
func (j *TrimPackagesJobHandler) Describe() string {
	return fmt.Sprintf("Trim packages to maximum of %d in '%s'", j.maxKeep, j.repoID)
}
