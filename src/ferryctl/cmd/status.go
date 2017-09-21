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
	"sort"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "display ferryd status",
	Long:  "Show an overview of currently registered jobs, and any failures",
	Run:   getStatus,
}

var (
	// How many jobs we print by default.
	maxPrintJobs = 10
	allJobs      = false
)

func init() {
	statusCmd.PersistentFlags().BoolVarP(&allJobs, "all", "a", false, "Show all jobs (limits to 10 by default)")
	RootCmd.AddCommand(statusCmd)
}

func printActiveJobs(js []*libferry.Job) {
	header := []string{
		"Status",
		"Queued",
		"Waited",
		"Description",
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetBorder(false)

	i := 0

	for _, j := range js {
		if i >= maxPrintJobs && !allJobs {
			break
		}
		i++
		var runType string
		if j.Timing.Begin.IsZero() {
			runType = "queued"
		} else {
			runType = "running"
		}
		table.Append([]string{
			runType,
			j.Timing.Queued.Format("2006-01-02 15:04:05"),
			j.QueuedSince().String(),
			j.Description,
		})
	}
	table.Render()
}

// Print out all the failed jobs
func printFailedJobs(js []*libferry.Job) {
	header := []string{
		"Status",
		"Completed",
		"Duration",
		"Description",
		"Error",
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetBorder(false)

	i := 0

	for _, j := range js {
		if i >= maxPrintJobs && !allJobs {
			break
		}
		i++
		table.Append([]string{
			"failed",
			j.Timing.End.Format("2006-01-02 15:04:05"),
			j.ExecutionTime().String(),
			j.Description,
			j.Error,
		})
	}
	table.Render()
}

// Print all successfully completed jobs
func printCompletedJobs(js []*libferry.Job) {
	header := []string{
		"Status",
		"Completed",
		"Duration",
		"Execution time",
		"Description",
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetBorder(false)

	i := 0

	for _, j := range js {
		if i >= maxPrintJobs && !allJobs {
			break
		}
		i++
		table.Append([]string{
			"success",
			j.Timing.End.Format("2006-01-02 15:04:05"),
			j.TotalTime().String(),
			j.ExecutionTime().String(),
			j.Description,
		})
	}
	table.Render()
}

func getStatus(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "status takes no arguments\n")
		return
	}

	client := libferry.NewClient(socketPath)
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
		sort.Sort(sort.Reverse(status.FailedJobs))
		fmt.Printf("Failed jobs: (%d tracked)\n\n", len(status.FailedJobs))
		printFailedJobs(status.FailedJobs)
	}

	// Show current
	if len(status.CurrentJobs) > 0 {
		sort.Sort(status.CurrentJobs)
		fmt.Printf("Current/Active jobs: (%d tracked)\n\n", len(status.CurrentJobs))
		printActiveJobs(status.CurrentJobs)
	}

	if len(status.CompletedJobs) > 0 {
		sort.Sort(sort.Reverse(status.CompletedJobs))
		fmt.Printf("Completed jobs: (%d tracked)\n\n", len(status.CompletedJobs))
		printCompletedJobs(status.CompletedJobs)
	}
}
