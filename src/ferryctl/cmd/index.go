//
// Copyright © 2017 Ikey Doherty <ikey@solus-project.com>
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
	"libferry/jobs"
	"os"
)

var indexCmd = &cobra.Command{
	Use:   "index [repo-name]",
	Short: "Re-construct the index of the given repository",
	Long:  "Re-construct the index of the given repository",
	Run:   indexRepo,
}

func init() {
	RootCmd.AddCommand(indexCmd)
}

func indexRepo(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "index takes exactly one argument")
		return
	}

	repoName := args[0]
	repoDir := "./ferry"

	// TODO: Get the right cwd always ..
	manager, err := libferry.NewManager(repoDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening repo manager: %v\n", err)
		return
	}

	defer manager.Close()
	jproc := jobs.NewProcessor(manager, -1)
	jproc.Begin()
	jproc.PushJob(jobs.NewIndexJob(repoName))
	jproc.Close()
}
