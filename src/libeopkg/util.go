//
// Copyright © 2016-2017 Ikey Doherty <ikey@solus-project.com>
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

package libeopkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DISCLAIMER: This stuff is just supporting the existing eopkg stuff.
// We know it's not ideal. When sol comes we'll have a much improved
// package format with a hash-indexed self deduplicating archive to
// mitigate delta issues, reduce sizes, and ensure verification at all stages

// ComputeDeltaName will determine the target name for the delta eopkg
func ComputeDeltaName(oldPackage, newPackage *MetaPackage) string {
	return fmt.Sprintf("%s-%d-%d-%s-%s.delta.eopkg",
		newPackage.Name,
		oldPackage.GetRelease(),
		newPackage.GetRelease(),
		newPackage.DistributionRelease,
		newPackage.Architecture)
}

// IsDeltaPossible will compare the two input packages and determine if it
// is possible for a delta to be considered. Note that we do not compare the
// distribution _name_ because Solus already had to do a rename once, and that
// broke delta updates. Let's not do that again. eopkg should in reality determine
// delta applicability based on repo origin + upgrade path, not names
func IsDeltaPossible(oldPackage, newPackage *MetaPackage) bool {
	return oldPackage.GetRelease() < newPackage.GetRelease() &&
		oldPackage.Name == newPackage.Name &&
		oldPackage.DistributionRelease == newPackage.DistributionRelease &&
		oldPackage.Architecture == newPackage.Architecture
}

// ProduceDelta will take two input packages and attempt to cook a delta package for
// them. This may fail due to the differences being two large
func ProduceDelta(oldPackage *MetaPackage, newPackage *MetaPackage, baseDir, deltaPath string) error {
	oldPackagePath := filepath.Join(baseDir, oldPackage.PackageURI)
	newPackagePath := filepath.Join(baseDir, newPackage.PackageURI)

	deltaDir := filepath.Dir(deltaPath)

	// eopkg is inefficient in generating delta packages, and will first extract the
	// newest package. The delta is then reproduced from the exploded new package, which
	// may result in permission violations (i.e. not being able to restore setuid
	// Consequently we pipe the call via fakeroot to "fix" this.
	cmd := []string{
		"fakeroot", "eopkg", "delta",
		oldPackagePath, newPackagePath,
		"-t", newPackagePath,
		"-o", deltaDir,
	}
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Dir = deltaDir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return err
	}

	// eopkg might decide to be a prat and not error even if the delta wasn't
	// generated, so we'll check after that the path even exists.
	if _, err := os.Stat(deltaPath); err != nil {
		return err
	}

	// In theory, have a .delta.eopkg now
	return nil
}
