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

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"libferry"
	"os"
	"path/filepath"
)

var importCmd = &cobra.Command{
	Use:   "import [repo] [packages]",
	Short: "Bulk import packages into repository",
	Long:  "Add packages in bulk to the named repository",
	Run:   importEx,
}

func init() {
	RootCmd.AddCommand(importCmd)
}

func importEx(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "import takes exactly at least 1 argument\n")
		return
	}

	client := libferry.NewClient(socketPath)
	defer client.Close()

	repoID := args[0]
	var packages []string
	for i := 1; i < len(args); i++ {
		f, err := filepath.Abs(args[i])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to probe: %s: %v\n", f, err)
			return
		}
		if _, err := os.Stat(f); err != nil {
			fmt.Fprintf(os.Stderr, "File does not exist: %s (%v)\n", f, err)
			return
		}
		packages = append(packages, f)
	}

	if err := client.ImportPackages(repoID, packages); err != nil {
		fmt.Fprintf(os.Stderr, "Import error: %v\n", err)
		return
	}
}
