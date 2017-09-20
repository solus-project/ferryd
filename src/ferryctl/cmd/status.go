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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "display ferryd status",
	Long:  "Show an overview of currently registered jobs, and any failures",
	Run:   getStatus,
}

func init() {
	RootCmd.AddCommand(statusCmd)
}

// printJob pretty prints the job to the CLI
func printJob(j *libferry.Job) {
}

func getStatus(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "status takes no arguments")
		return
	}

	client := libferry.NewClient("./ferryd.sock")
	defer client.Close()

	status, err := client.GetStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	// Show uptime
	fmt.Printf("Daemon uptime: %v\n", status.Uptime())

	// Show failing
	if len(status.FailedJobs) > 0 {
		fmt.Printf("Failed jobs: \n\n")
		for i := range status.FailedJobs {
			printJob(&status.FailedJobs[i])
		}
	}

	// Show current
	if len(status.CurrentJobs) > 0 {
		fmt.Printf("Current jobs: \n\n")
		for i := range status.CurrentJobs {
			printJob(&status.CurrentJobs[i])
		}
	}
}
