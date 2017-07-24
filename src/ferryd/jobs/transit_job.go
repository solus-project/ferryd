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
	"ferryd/core"
	log "github.com/sirupsen/logrus"
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

	log.WithFields(log.Fields{
		"target":   t.manifest.Manifest.Target,
		"manifest": t.manifest.ID(),
	}).Info("Successfully processed upload")
	return nil
}
