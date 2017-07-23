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
	"testing"
)

// Test a real eopkg and make sure we get the right path from it
func TestMetadataSourcePackage(t *testing.T) {
	pkg, err := Open(eopkgTestFile)
	if err != nil {
		t.Fatalf("Error opening valid .eopkg file: %v", err)
	}
	defer pkg.Close()
	if err := pkg.ReadMetadata(); err != nil {
		t.Fatalf("Failed to read metadata: %v", err)
	}
	expPath := "n/nano"
	gotPath := pkg.Meta.Package.GetPathComponent()
	if expPath != gotPath {
		t.Fatalf("Expected source path '%s', got '%s'", expPath, gotPath)
	}
}

// Test a variety of source names and ensure we get the right component
// each time for the subpath we expect to see in the repository
func TestMetadataSourceDummy(t *testing.T) {
	metaDatas := []MetaPackage{
		{
			Source: Source{
				Name: "libreoffice",
			},
		},
		{
			Source: Source{
				Name: "lib",
			},
		},
		{
			Source: Source{
				Name: "alsa-lib",
			},
		},
		{
			Source: Source{
				Name: "NANO",
			},
		},
	}
	expected := []string{
		"libr/libreoffice",
		"l/lib",
		"a/alsa-lib",
		"n/nano",
	}
	for i := range metaDatas {
		exp := expected[i]
		got := (&metaDatas[i]).GetPathComponent()
		if exp != got {
			t.Fatalf("Expected source path '%s', got '%s'", exp, got)
		}
	}
}
