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

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"manager"
	"os"
	"sort"
)

var listCmd = &cobra.Command{
	Use:   "list-repos",
	Short: "list known repositories",
	Long:  "List all of the repositories currently known to binman",
	Run:   listRepos,
}

func init() {
	RootCmd.AddCommand(listCmd)
}

// createRepo will use the manager to create a new repository with the
// specified name.
func listRepos(cmd *cobra.Command, args []string) {
	man, err := manager.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to instantiate the manager: %v\n", err)
		return
	}
	defer man.Cleanup()

	repos, err := man.ListRepos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to list repositories: %v\n", err)
		return
	}
	if len(repos) < 1 {
		fmt.Printf("No repositories currently known\nCreate one with 'create-repo'\n")
		return
	}
	fmt.Print("Registered repositories:\n\n")
	sort.Strings(repos)
	for _, repoName := range repos {
		fmt.Printf(" - %s\n", repoName)
	}
}
