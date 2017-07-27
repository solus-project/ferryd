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

package libeopkg

import (
	"os"
	"testing"
)

// Our test files, known to produce a valid delta package
const (
	deltaOldPkg = "testdata/delta/nano-2.8.5-75-1-x86_64.eopkg"
	deltaNewPkg = "testdata/delta/nano-2.8.6-76-1-x86_64.eopkg"
)

func TestBasicDelta(t *testing.T) {
	producer, err := NewDeltaProducer(deltaOldPkg, deltaNewPkg)
	if err != nil {
		t.Fatalf("Failed to create delta producer for existing pkgs: %v", err)
	}
	defer producer.Close()
	path, err := producer.Commit()
	if err != nil {
		t.Fatalf("Failed to produce delta packages: %v", err)
	}
	defer os.Remove(path)

	pkg, err := Open(path)
	if err != nil {
		t.Fatalf("Failed to open our delta package: %v", err)
	}
	defer pkg.Close()
	if err = pkg.ReadAll(); err != nil {
		t.Fatalf("Failed to read metadata on delta: %v", err)
	}
	if pkg.Meta.Package.Name != "nano" {
		t.Fatalf("Invalid delta name: %s", pkg.Meta.Package.Name)
	}
	if pkg.Meta.Package.GetRelease() != 76 {
		t.Fatalf("Invalid release number in delta: %d", pkg.Meta.Package.GetRelease())
	}

}
