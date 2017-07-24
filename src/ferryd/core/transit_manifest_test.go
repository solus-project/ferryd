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

package core

import (
	"testing"
)

const (
	transitTestFile = "testdata/nano.tram"
)

func TestTransitManifest(t *testing.T) {
	tm, err := NewTransitManifest(transitTestFile)
	if err != nil {
		t.Fatalf("Failed to load valid tram file: %v", err)
	}
	if tm.Manifest.Version != "1.0" {
		t.Fatalf("Invalid header version: %s", tm.Manifest.Version)
	}
	if tm.Manifest.Target != "unstable" {
		t.Fatalf("Invalid header target: %s", tm.Manifest.Target)
	}
	if len(tm.File) != 2 {
		t.Fatalf("Invalid number of files in payload: %d", len(tm.File))
	}
	if tm.File[1].Path != "nano-dbginfo-2.7.5-68-1-x86_64.eopkg" {
		t.Fatalf("Invalid path in tram file :%s", tm.File[1].Path)
	}
	if tm.File[0].Sha256 != "1810f4d36d42a9d41a37bcd31a70c2279c4cb7b02627bcab981f94f3a24bfcc5" {
		t.Fatalf("Invalid sha in tram file: %s", tm.File[0].Sha256)
	}
}
