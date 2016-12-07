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
)

var addCmd = &cobra.Command{
	Use:   "add [repo-name] [package1.eopkg] [package2.eopkg]",
	Short: "add packages to the given repository",
	Long: `Sideload the given packages into the given repository, without
using the normal processing methods`,
	Example: "binman add myCustomRepo *.eopkg",
	RunE:    addPackages,
}

func init() {
	RootCmd.AddCommand(addCmd)
}

// addPackages will use the manager to add packages to the specified repo
func addPackages(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("You must supply the name of a repository")
	}
	repoNom := args[0]
	pkgs := args[1:]
	if len(pkgs) < 1 {
		return fmt.Errorf("You must supply the path to .eopkg files")
	}

	man, err := manager.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to instantiate the manager: %v\n", err)
		return nil
	}
	defer man.Cleanup()

	if err := man.AddPackages(repoNom, pkgs); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to add packages: %v\n", err)
		return nil
	}
	fmt.Printf("Successfully added packages\n")

	return nil
}
