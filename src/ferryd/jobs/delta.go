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
	"os"
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
	nDeltas     int // Track how many deltas we actually produce
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
		nDeltas:     0,
	}, nil
}

// executeInternal is the common code shared in the delta jobs, and is
// split out to save duplication.
func (j *DeltaJobHandler) executeInternal(manager *core.Manager) error {
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

		// Don't need to report that it failed, we know this from history
		if manager.GetDeltaFailed(deltaID) {
			continue
		}

		hasDelta, err := manager.HasDelta(j.repoID, j.packageName, deltaID)
		if err != nil {
			return err
		}

		// Package has this delta already? Continue.
		if hasDelta {
			continue
		}

		mapping := &core.DeltaInformation{
			FromID:      old.GetID(),
			ToID:        tip.GetID(),
			FromRelease: old.GetRelease(),
			ToRelease:   tip.GetRelease(),
		}

		// Before we go off creating it - does the delta package exist already?
		// If so, just re-ref it for usage within the new repo
		entry, err := manager.GetPoolEntry(deltaID)
		if entry != nil && err == nil {
			if err := manager.RefDelta(j.repoID, deltaID, mapping); err != nil {
				fields["error"] = err
				log.WithFields(fields).Error("Failed to ref existing delta")
				return err
			}
			log.WithFields(fields).Info("Reused existing delta")
			continue
		}

		deltaPath, err := manager.CreateDelta(j.repoID, old, tip)
		if err != nil {
			fields["error"] = err
			if err == libeopkg.ErrDeltaPointless {
				// Non-fatal, ask the manager to record this delta as a no-go
				log.WithFields(fields).Info("Delta not possible, marked permanently")
				if err := manager.MarkDeltaFailed(deltaID, mapping); err != nil {
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

		j.nDeltas++

		fields["path"] = deltaPath
		// Produced a delta!
		log.WithFields(fields).Info("Successfully producing delta package")

		// Let's get it included now.
		if err = j.includeDelta(manager, mapping, deltaPath); err != nil {
			fields["error"] = err
			log.WithFields(fields).Error("Failed to include delta package")
			return err
		}
	}

	return nil
}

// includeDelta will wrap up the basic functionality to get a delta package
// imported into a target repository.
func (j *DeltaJobHandler) includeDelta(manager *core.Manager, mapping *core.DeltaInformation, deltaPath string) error {
	// Try to insert the delta
	if err := manager.AddDelta(j.repoID, deltaPath, mapping); err != nil {
		return err
	}

	// Delete the deltaPath if the add is successful
	return os.Remove(deltaPath)
}

// Execute will delta the target package within the target repository.
func (j *DeltaJobHandler) Execute(_ *Processor, manager *core.Manager) error {
	err := j.executeInternal(manager)
	if err != nil {
		return err
	}
	if !j.indexRepo {
		return nil
	}
	// Ask that our repository now be reindexed because we've added deltas but
	// only if we've successfully produced some delta packages
	if j.nDeltas < 0 {
		return nil
	}

	if err := manager.Index(j.repoID); err != nil {
		log.WithFields(log.Fields{
			"repo":  j.repoID,
			"error": err,
		}).Error("Failed to index repository")
		return err
	}

	return nil
}

// Describe returns a human readable description for this job
func (j *DeltaJobHandler) Describe() string {
	if j.indexRepo {
		return fmt.Sprintf("Delta package '%s' on '%s', then re-index", j.packageName, j.repoID)
	}
	return fmt.Sprintf("Delta package '%s' on '%s'", j.packageName, j.repoID)
}
