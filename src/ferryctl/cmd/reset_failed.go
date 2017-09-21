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

var resetFailedCmd = &cobra.Command{
	Use:   "failed",
	Short: "reset failed logs",
	Long:  "Purge the logs for failed jobs",
	Run:   resetFailed,
}

func init() {
	ResetCmd.AddCommand(resetFailedCmd)
}

func resetFailed(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "reset failed takes no arguments\n")
		return
	}

	client := libferry.NewClient(socketPath)
	defer client.Close()

	if err := client.ResetFailed(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
}
