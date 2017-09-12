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

package libeopkg

import (
	"testing"
)

const (
	distTestFile = "testdata/distribution.xml"
)

func TestDistribution(t *testing.T) {
	dist, err := NewDistribution(distTestFile)
	if err != nil {
		t.Fatalf("Failed to load good file: %s", err)
	}
	if dist == nil {
		t.Fatalf("Failed to get distribution")
	}
	if dist.SourceName != "Solus" {
		t.Fatalf("Invalid source name %s", dist.SourceName)
	}
	if dist.Version != "1" {
		t.Fatalf("Invalid version %s", dist.Version)
	}
	if len(dist.Description) != 23 {
		t.Fatalf("Invalid number of descriptions: %d", len(dist.Description))
	}
	if dist.Description[0].Lang != "" {
		t.Fatalf("First element should not have language: %s", dist.Description[0].Lang)
	}
	if dist.Description[22].Lang != "zh_CN" {
		t.Fatalf("Lang on last element is wrong: %s", dist.Description[21].Lang)
	}
	if dist.Type != "main" {
		t.Fatalf("Invalid repo type: %s", dist.Type)
	}
	if dist.BinaryName != "Solus" {
		t.Fatalf("Invalid binary name: %s", dist.BinaryName)
	}

	if dist.Obsoletes[0] != "pcre" {
		t.Fatalf("Invalid first obsolete: %s", dist.Obsoletes[0])
	}
}
