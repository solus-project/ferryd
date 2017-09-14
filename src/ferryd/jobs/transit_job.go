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
	"os"
	"path/filepath"
)

// TransitJobHandler is responsible for accepting new upload payloads in the repository
type TransitJobHandler struct {
	path     string
	manifest *core.TransitManifest
}

// NewTransitJob will return a job suitable for adding to the job processor
func NewTransitJob(path string) *JobEntry {
	return &JobEntry{
		sequential: true,
		Type:       TransitProcess,
		Params:     []string{path},
	}
}

// NewTransitJobHandler will create a job handler for the input job and ensure it validates
func NewTransitJobHandler(j *JobEntry) (*TransitJobHandler, error) {
	if len(j.Params) != 1 {
		return nil, fmt.Errorf("job has invalid parameters")
	}
	return &TransitJobHandler{
		path: j.Params[0],
	}, nil
}

// Execute will index the given repository if possible
func (j *TransitJobHandler) Execute(jproc *Processor, manager *core.Manager) error {
	tram, err := core.NewTransitManifest(j.path)
	if err != nil {
		return err
	}

	if err = tram.ValidatePayload(); err != nil {
		return err
	}

	j.manifest = tram

	// Sanity.
	repo := j.manifest.Manifest.Target
	if _, err := manager.GetRepo(repo); err != nil {
		return err
	}

	// Now try to merge into the repo
	pkgs := tram.GetPaths()
	if err = manager.AddPackages(repo, pkgs); err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"target": repo,
		"id":     j.manifest.ID(),
	}).Info("Successfully processed manifest upload")

	// Append the manifest path because now we'll want to delete these
	pkgs = append(pkgs, j.path)

	for _, p := range pkgs {
		if !core.PathExists(p) {
			continue
		}
		if err := os.Remove(p); err != nil {
			log.WithFields(log.Fields{
				"file":  p,
				"id":    j.manifest.ID(),
				"error": err,
			}).Error("Failed to remove manifest file upload")
		}
	}

	// At this point we should actually have valid pool entries so
	// we'll grab their names, and schedule that they be re-deltad.
	// It might be the case no delta is possible, but we'll let the
	// DeltaJobHandler decide on that.
	for _, pkg := range pkgs {
		pkgID := filepath.Base(pkg)
		p, ent := manager.GetPoolEntry(pkgID)
		if ent != nil {
			return err
		}
		jproc.PushJob(NewDeltaIndexJob(repo, p.Name))
	}

	return nil
}

// Describe returns a human readable description for this job
func (j *TransitJobHandler) Describe() string {
	if j.manifest == nil {
		return fmt.Sprintf("Process manifest '%s'", j.path)
	}

	return fmt.Sprintf("Process manifest '%s' for target '%s'", j.manifest.ID(), j.manifest.Manifest.Target)
}
