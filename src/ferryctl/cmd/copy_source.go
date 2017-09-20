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

var copySourceCmd = &cobra.Command{
	Use:   "source [fromRepo] [targetRepo] [sourceName] [releaseNumber]",
	Short: "copy packages by source name",
	Long:  "Remove an existing package set in the ferryd instance",
	Run:   copySource,
}

func init() {
	CopyCmd.AddCommand(copySourceCmd)
}

func copySource(cmd *cobra.Command, args []string) {
	// TODO: Support -1 implicitly to copy *all* by source ID
	if len(args) != 4 {
		fmt.Fprintf(os.Stderr, "copy source takes exactly 4 arguments\n")
		return
	}

	release, err := strconv.ParseInt(args[3], 10, 32)
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
	targetID := args[1]
	sourceID := args[2]

	if err := client.CopySource(repoID, targetID, sourceID, int(release)); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
}
