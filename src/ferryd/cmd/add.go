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

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"libferry"
	"os"
)

var addCmd = &cobra.Command{
	Use:   "add [repo-name] [packages]",
	Short: "Add package(s) to repository",
	Long:  "Add a list of packages to the given repository",
	Run:   addPackages,
}

func init() {
	RootCmd.AddCommand(addCmd)
}

func addPackages(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "add takes at least 2 arguments")
		return
	}

	repoName := args[0]
	repoDir := "./ferry"

	// TODO: Get the right cwd always ..
	manager, err := libferry.NewManager(repoDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening repo manager: %v\n", err)
		return
	}
	defer manager.Close()

	packages := args[1:]

	if err := manager.AddPackages(repoName, packages); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("Added packages to: %s\n", repoName)
}
