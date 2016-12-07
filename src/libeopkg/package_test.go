//
// Copyright © 2016 Ikey Doherty <ikey@solus-project.com>
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
	"testing"
)

const (
	eopkgTestFile = "testdata/nano-2.7.1-63-1-x86_64.eopkg"
)

// TestPackageOpen will validate simple open of package files
func TestPackageOpen(t *testing.T) {
	pkg, err := Open(eopkgTestFile)
	if err != nil {
		t.Fatalf("Error opening valid .eopkg file: %v", err)
	}
	defer pkg.Close()
}
