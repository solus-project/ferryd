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
	"os"
)

// IncludeDeltaJobHandler is responsible for indexing repositories and should only
// ever be used in sequential queues.
type IncludeDeltaJobHandler struct {
	repoID    string
	sourceID  string
	targetID  string
	deltaPath string
}

// NewIncludeDeltaJob will return a job suitable for adding to the job processor
func NewIncludeDeltaJob(repoID, sourceID, targetID, deltaPath string) *JobEntry {
	return &JobEntry{
		sequential: true,
		Type:       IncludeDelta,
		Params:     []string{repoID, sourceID, targetID, deltaPath},
	}
}

// NewIncludeDeltaJobHandler will create a job handler for the input job and ensure it validates
func NewIncludeDeltaJobHandler(j *JobEntry) (*IncludeDeltaJobHandler, error) {
	if len(j.Params) != 4 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	return &IncludeDeltaJobHandler{
		repoID:    j.Params[0],
		sourceID:  j.Params[1],
		targetID:  j.Params[2],
		deltaPath: j.Params[3],
	}, nil
}

// Execute will index the given repository if possible
func (j *IncludeDeltaJobHandler) Execute(_ *Processor, manager *core.Manager) error {
	// Populate basic mapping, core.Pool populates the rest
	mapping := &core.DeltaInformation{
		FromID: j.sourceID,
		ToID:   j.targetID,
	}

	// Try to insert the delta
	if err := manager.AddDelta(j.repoID, j.deltaPath, mapping); err != nil {
		return err
	}

	// Delete the deltaPath if the add is successful
	return os.Remove(j.deltaPath)
}

// Describe returns a human readable description for this job
func (j *IncludeDeltaJobHandler) Describe() string {
	return fmt.Sprintf("Include delta '%s' into repository '%s'", j.deltaPath, j.repoID)
}
