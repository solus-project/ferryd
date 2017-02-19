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
	"fmt"
	"os"
	"testing"
)

const (
	eopkgTestFile = "testdata/nano-2.7.1-63-1-x86_64.eopkg"
)

// TestPackageOpen will validate simple open of package files
func TestPackageOpen(t *testing.T) {
	if _, err := Open("NoSuchFile.txt"); err == nil {
		t.Fatal("Mysteriously opened a file that doesn't exist!")
	}
	if _, err := Open("package_test.go"); err == nil {
		t.Fatal("Opened an invalid archive!")
	}
	pkg, err := Open(eopkgTestFile)
	if err != nil {
		t.Fatalf("Error opening valid .eopkg file: %v", err)
	}
	defer pkg.Close()

	meta := pkg.FindFile("metadata.xml")
	if meta == nil {
		t.Fatal("Good archive is missing metadata.xml")
	}
	if meta.Name != "metadata.xml" {
		t.Fatalf("Incorrect metadata.xml file returned: %v", meta.Name)
	}
	files := pkg.FindFile("files.xml")
	if files == nil {
		t.Fatal("Good archive is missing files.xml")
	}
	if files.Name != "files.xml" {
		t.Fatalf("Incorrect files.xml file returned: %v", files.Name)
	}
}

func TestPackageMeta(t *testing.T) {
	pkg, err := Open(eopkgTestFile)
	if err != nil {
		t.Fatalf("Error opening valid .eopkg file: %v", err)
	}
	defer pkg.Close()
	if err = pkg.ReadMetadata(); err != nil {
		t.Fatalf("Error reading metadata: %v", err)
	}
	metaPkg := pkg.Meta.Package
	fmt.Fprintf(os.Stderr, "Package: %s (%s-%d)\n", metaPkg.Name, metaPkg.History[0].Version, metaPkg.History[0].Release)
	fmt.Fprintf(os.Stderr, "Summary: %s\n", metaPkg.Summary)
}
