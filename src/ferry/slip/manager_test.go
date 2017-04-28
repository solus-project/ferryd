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

package slip

import (
	"os"
	"path/filepath"
	"testing"
)

// initTestArea is a very simple helper to set up a database staging tree
func initTestArea(t *testing.T) string {
	dirName := filepath.Join(".", "testenv")
	if _, err := os.Stat(dirName); err == nil {
		if err = os.RemoveAll(dirName); err != nil {
			t.Fatalf("Cannot clean the test environment: %v", err)
		}
	}
	if err := os.MkdirAll(dirName, 00755); err != nil {
		t.Fatalf("Cannot mkdirs for test: %v", err)
	}
	return dirName
}

// TestManagerBasic will verify the most basic functionality
// within slip.
func TestManagerBasic(t *testing.T) {
	manager, err := NewManager(initTestArea(t))
	if err != nil {
		t.Fatalf("Failed to initialise a new manager for the current directory: %v", err)
	}
	manager.Close()
}
