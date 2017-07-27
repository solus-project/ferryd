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

// DeltaPackageJob is a parallel job which will attempt the construction of
// deltas for a given package name + repo
type DeltaPackageJob struct {
	packageName string
	repoID      string
}

// NewDeltaPackageJob will create a new delta job for the given repo + package name
func NewDeltaPackageJob(repoID, packageName string) *DeltaPackageJob {
	return &DeltaPackageJob{repoID: repoID, packageName: packageName}
}

// Init is unused for this job
func (d *DeltaPackageJob) Init(jproc *Processor) {}

// IsSequential will return false as we're happy to batch up multiple delta
// operations provided they're parented with an indexing job
func (d *DeltaPackageJob) IsSequential() bool {
	return false
}

// Perform will invoke the indexing operation
func (d *DeltaPackageJob) Perform(manager *core.Manager) error {
	pkgs, err := manager.GetPackages(d.repoID, d.packageName)
	if err != nil {
		return err
	}

	// Duh.
	if len(pkgs) < 2 {
		return nil
	}

	sort.Sort(PackageSet(pkgs))
	tip := pkgs[len(pkgs)-1]

	for i := 0; i < len(pkgs)-1; i++ {
		old := pkgs[i]
		if err := manager.CreateDelta(d.repoID, old, tip); err != nil {
			log.WithFields(log.Fields{
				"old":   old.GetID(),
				"new":   tip.GetID(),
				"error": err,
				"repo":  d.repoID,
			}).Error("Error producing delta package")
			return err
		}
		log.WithFields(log.Fields{
			"old":  old.GetID(),
			"new":  tip.GetID(),
			"repo": d.repoID,
		}).Info("Successfully producing delta package")
	}

	return nil
}

// Describe will explain the purpose of this job
func (d *DeltaPackageJob) Describe() string {
	return fmt.Sprintf("Delta package '%s' on '%s'", d.packageName, d.repoID)
}
