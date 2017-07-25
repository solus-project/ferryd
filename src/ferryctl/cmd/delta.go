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
	"os"
)

var deltaCmd = &cobra.Command{
	Use:   "delta [repo]",
	Short: "Create deltas",
	Long:  "Schedule that the repo has all deltas rebuilt",
	Run:   delta,
}

func init() {
	RootCmd.AddCommand(deltaCmd)
}

func delta(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "delta takes exactly 1 argument")
		return
	}

	client := libferry.NewClient("./ferryd.sock")
	defer client.Close()

	if err := client.DeltaRepo(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
}
