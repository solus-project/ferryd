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

var removeSourceCmd = &cobra.Command{
	Use:   "source [repoName] [sourceName] [releaseNumber]",
	Short: "remove packages by source name",
	Long:  "Remove an existing package set in the ferryd instance",
	Run:   removeSource,
}

func init() {
	RemoveCmd.AddCommand(removeSourceCmd)
}

func removeSource(cmd *cobra.Command, args []string) {
	// TODO: Support -1 implicitly to remove *all* by source ID
	if len(args) != 3 {
		fmt.Fprintf(os.Stderr, "remove source takes exactly 3 arguments\n")
		return
	}

	release, err := strconv.ParseInt(args[2], 10, 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid integer: %v\n", err)
		return
	}

	if release < 1 {
		fmt.Fprintf(os.Stderr, "Release should be higher than 1\n")
		return
	}

	client := libferry.NewClient("./ferryd.sock")
	defer client.Close()

	repoID := args[0]
	sourceID := args[1]

	if err := client.RemoveSource(repoID, sourceID, int(release)); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
}
