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
	indexRepo   bool
}

// NewDeltaJob will return a job suitable for adding to the job processor
func NewDeltaJob(repoID, packageID string) *JobEntry {
	return &JobEntry{
		sequential: false,
		Type:       Delta,
		Params:     []string{repoID, packageID},
	}
}

// NewDeltaIndexJob will return a new job for creating delta packages as well
// as scheduling an index operation when complete.
func NewDeltaIndexJob(repoID, packageID string) *JobEntry {
	return &JobEntry{
		sequential: false,
		Type:       DeltaIndex,
		Params:     []string{repoID, packageID},
	}
}

// NewDeltaJobHandler will create a job handler for the input job and ensure it validates
func NewDeltaJobHandler(j *JobEntry, indexRepo bool) (*DeltaJobHandler, error) {
	if len(j.Params) != 2 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	return &DeltaJobHandler{
		repoID:      j.Params[0],
		packageName: j.Params[1],
		indexRepo:   indexRepo,
	}, nil
}

// executeInternal is the common code shared in the delta jobs, and is
// split out to save duplication.
func (j *DeltaJobHandler) executeInternal(jproc *Processor, manager *core.Manager) error {
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

	// TODO: Invalidate old deltas
	// TODO: Consider spawning an async for *each* individual delta eopkg which
	// could speed things up considerably.
	for i := 0; i < len(pkgs)-1; i++ {
		old := pkgs[i]
		fields := log.Fields{
			"old":  old.GetID(),
			"new":  tip.GetID(),
			"repo": j.repoID,
		}

		deltaID := libeopkg.ComputeDeltaName(old, tip)
		failed, err := manager.GetDeltaFailed(deltaID)
		if err != nil {
			return err
		}

		// Don't need to report that it failed, we know this from history
		if failed {
			continue
		}

		deltaPath, err := manager.CreateDelta(j.repoID, old, tip)
		if err != nil {
			fields["error"] = err
			if err == libeopkg.ErrDeltaPointless {
				// Non-fatal, ask the manager to record this delta as a no-go
				log.WithFields(fields).Info("Delta not possible, marked permanently")
				if err := manager.MarkDeltaFailed(deltaID); err != nil {
					fields["error"] = err
					log.WithFields(fields).Error("Failed to mark delta failure")
					return err
				}
				continue
			} else if err == libeopkg.ErrMismatchedDelta {
				log.WithFields(fields).Error("Package delta candidates do not match")
				continue
			} else {
				// Genuinely an issue now
				log.WithFields(fields).Error("Error in delta production")
				return err
			}
		}

		log.WithFields(log.Fields{
			"path": deltaPath,
			"old":  old.GetID(),
			"new":  tip.GetID(),
			"repo": j.repoID,
		}).Info("Successfully producing delta package")

		// Note if we push an index job, it's also on the sequential queue so it
		// still won't actually run until after we've included the deltas from our
		// own job run.
		jproc.PushJob(NewIncludeDeltaJob(j.repoID, old.GetID(), tip.GetID(), deltaPath))
	}

	return nil
}

// Execute will delta the target package within the target repository.
func (j *DeltaJobHandler) Execute(jproc *Processor, manager *core.Manager) error {
	err := j.executeInternal(jproc, manager)
	if err != nil {
		return err
	}
	if !j.indexRepo {
		return nil
	}
	// TODO: Only index if we've actually CREATED deltas!!
	// Ask that our repository now be reindexed because we've added deltas
	jproc.PushJob(NewIndexRepoJob(j.repoID))
	return nil
}

// Describe returns a human readable description for this job
func (j *DeltaJobHandler) Describe() string {
	if j.indexRepo {
		return fmt.Sprintf("Delta package '%s' on '%s', then re-index", j.packageName, j.repoID)
	}
	return fmt.Sprintf("Delta package '%s' on '%s'", j.packageName, j.repoID)
}
