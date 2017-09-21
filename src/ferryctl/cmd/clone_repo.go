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
)

var (
	fullClone bool
)

var cloneRepoCmd = &cobra.Command{
	Use:   "clone [from] [newName]",
	Short: "clone an existing repository",
	Long:  "Clone an existing repository into a new repository",
	Run:   cloneRepo,
}

func init() {
	cloneRepoCmd.PersistentFlags().BoolVarP(&fullClone, "full", "f", false, "Perform a deep clone")
	RootCmd.AddCommand(cloneRepoCmd)
}

func cloneRepo(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "clone takes exactly 2 arguments\n")
		return
	}

	client := libferry.NewClient(socketPath)
	defer client.Close()

	if err := client.CloneRepo(args[0], args[1], fullClone); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
}
