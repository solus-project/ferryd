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
	"libeopkg"
	"sort"
)

// PackageSet provides sorting capabilities for a slice of packages
type PackageSet []*libeopkg.MetaPackage

func (p PackageSet) Len() int {
	return len(p)
}

func (p PackageSet) Less(a, b int) bool {
	return p[a].GetRelease() < p[b].GetRelease()
}

func (p PackageSet) Swap(a, b int) {
	p[a], p[b] = p[b], p[a]
}

// DeltaJobHandler is responsible for indexing repositories and should only
// ever be used in async queues. Deltas may take some time to produce and
// shouldn't be allowed to block the sequential processing queue.
type DeltaJobHandler struct {
	repoID      string
	packageName string
}

// NewDeltaJob will return a job suitable for adding to the job processor
func NewDeltaJob(repoID, packageID string) *JobEntry {
	return &JobEntry{
		sequential: false,
		Type:       Delta,
		Params:     []string{repoID, packageID},
	}
}

// NewDeltaJobHandler will create a job handler for the input job and ensure it validates
func NewDeltaJobHandler(j *JobEntry) (*DeltaJobHandler, error) {
	if len(j.Params) != 2 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	return &DeltaJobHandler{
		repoID:      j.Params[0],
		packageName: j.Params[1],
	}, nil
}

// Execute will delta the target package within the target repository.
func (j *DeltaJobHandler) Execute(_ *Processor, manager *core.Manager) error {
	pkgs, err := manager.GetPackages(j.repoID, j.packageName)
	if err != nil {
		return err
	}

	// Need at least 2 packages for a delta op.
	if len(pkgs) < 2 {
		log.WithFields(log.Fields{
			"repo":    j.repoID,
			"package": j.packageName,
		}).Debug("No delta is possible")
		return nil
	}

	sort.Sort(PackageSet(pkgs))
	tip := pkgs[len(pkgs)-1]

	// TODO: Record new deltas, invalidate old deltas
	// TODO: Consider spawning an async for *each* individual delta eopkg which
	// could speed things up considerably.
	for i := 0; i < len(pkgs)-1; i++ {
		old := pkgs[i]
		if err := manager.CreateDelta(j.repoID, old, tip); err != nil {
			log.WithFields(log.Fields{
				"old":   old.GetID(),
				"new":   tip.GetID(),
				"error": err,
				"repo":  j.repoID,
			}).Error("Error producing delta package")
			return err
		}
		log.WithFields(log.Fields{
			"old":  old.GetID(),
			"new":  tip.GetID(),
			"repo": j.repoID,
		}).Info("Successfully producing delta package")
	}

	return nil
}

// Describe returns a human readable description for this job
func (j *DeltaJobHandler) Describe() string {
	return fmt.Sprintf("Delta package '%s' on '%s'", j.packageName, j.repoID)
}
