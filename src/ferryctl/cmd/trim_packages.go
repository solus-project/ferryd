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
	"strconv"
)

var trimPackagesCmd = &cobra.Command{
	Use:   "packages [repoName] [max to keep]",
	Short: "trim packages back to a maximum of [max to keep]",
	Long:  "Trim excessive back versions for packages in the repository",
	Run:   trimPackages,
}

func init() {
	TrimCmd.AddCommand(trimPackagesCmd)
}

func trimPackages(cmd *cobra.Command, args []string) {
	// TODO: Support -1 implicitly set
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "trim packages takes exactly 2 arguments\n")
		return
	}

	maxKeep, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid integer: %v\n", err)
		return
	}

	if maxKeep < 1 {
		fmt.Fprintf(os.Stderr, "maxKeep should be higher than 1\n")
		return
	}

	client := libferry.NewClient("./ferryd.sock")
	defer client.Close()

	repoID := args[0]

	if err := client.TrimPackages(repoID, int(maxKeep)); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
}
