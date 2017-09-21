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

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "index the given repository",
	Long:  "Request the index be reconstructed in the given repository",
	Run:   index,
}

func init() {
	RootCmd.AddCommand(indexCmd)
}

func index(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "index takes exactly 1 argument\n")
		return
	}

	client := libferry.NewClient(socketPath)
	defer client.Close()

	if err := client.IndexRepo(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
}
