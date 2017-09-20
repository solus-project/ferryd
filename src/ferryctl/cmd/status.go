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
	"github.com/olekukonko/tablewriter"
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

func printJobs(js []*libferry.Job) {
	header := []string{
		"Queued",
		"Completed",
		"Duration",
		"Description",
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetBorder(false)

	for _, j := range js {
		timeStart := j.QueuedSince().String()

		// Is it actually complete ?
		if j.Timing.End.IsZero() {
			table.Append([]string{
				timeStart,
				"-",
				"-",
				j.Description,
			})
		} else {
			table.Append([]string{
				timeStart,
				j.Executed().String(),
				j.ExecutionTime().String(),
				j.Description,
			})
		}
	}
	table.Render()
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
	fmt.Printf(" - Daemon uptime: %v\n", status.Uptime())
	fmt.Printf(" - Daemon version: %v\n", status.Version)

	// Show failing
	if len(status.FailedJobs) > 0 {
		fmt.Printf("Failed jobs: \n\n")
		printJobs(status.FailedJobs)
	}

	// Show current
	if len(status.CurrentJobs) > 0 {
		fmt.Printf("Current jobs: \n\n")
		printJobs(status.CurrentJobs)
	}
}
