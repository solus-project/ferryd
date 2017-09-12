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
)

// TransitProcessJob is a sequential job that will process the incoming uploads
// directory, dealing with each .tram upload
type TransitProcessJob struct {
	path     string
	manifest *core.TransitManifest
}

// NewTransitProcessJob will create a new job for the given .tram path
func NewTransitProcessJob(path string) *TransitProcessJob {
	return &TransitProcessJob{path: path}
}

// Init is unused for this job
func (t *TransitProcessJob) Init(jproc *Processor) {}

// IsSequential will return true as we're going to need to index after
func (t *TransitProcessJob) IsSequential() bool {
	return true
}

// Perform will invoke the operation
func (t *TransitProcessJob) Perform(manager *core.Manager) error {
	tram, err := core.NewTransitManifest(t.path)
	if err != nil {
		return err
	}

	if err = tram.ValidatePayload(); err != nil {
		return err
	}

	t.manifest = tram

	// Sanity.
	repo := t.manifest.Manifest.Target
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
		"id":     t.manifest.ID(),
	}).Info("Successfully processed manifest upload")

	// Append the manifest path because now we'll want to delete these
	pkgs = append(pkgs, t.path)
	for _, p := range pkgs {
		if err := os.Remove(p); err != nil {
			log.WithFields(log.Fields{
				"file":  p,
				"id":    t.manifest.ID(),
				"error": err,
			}).Error("Failed to remove manifest file upload")
		}
	}
	return nil
}

// Describe will explain the purpose of this job
func (t *TransitProcessJob) Describe() string {
	if t.manifest == nil {
		return fmt.Sprintf("Process manifest '%s'", t.path)
	}

	return fmt.Sprintf("Process manifest '%s' for target '%s'", t.manifest.ID(), t.manifest.Manifest.Target)
}
