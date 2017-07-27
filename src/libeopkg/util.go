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
