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

var createCmd = &cobra.Command{
	Use:   "create-repo [name]",
	Short: "create a repository",
	Long: `Create a new repository with the given name, This will be an empty
repository, and must have a unique name.`,
	Example: `binman create-repo myCustomRepo

Creates a new repository with the name "myCustomRepo".`,
	RunE: createRepo,
}

func init() {
	RootCmd.AddCommand(createCmd)
}

// createRepo will use the manager to create a new repository with the
// specified name.
func createRepo(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("You must supply the name of a repository")
	}
	man, err := manager.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to instaniate the manager: %v\n", err)
		return nil
	}
	defer man.Cleanup()

	// Create the repo now
	if err := man.CreateRepo(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create repository: %v\n", err)
		return nil
	}
	return nil
}
