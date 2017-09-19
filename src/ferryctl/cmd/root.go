//
// Copyright © 2016-2017 Solus Project
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
	"github.com/spf13/cobra"
)

// RootCmd is the main entry point into ferry
var RootCmd = &cobra.Command{
	Use:   "ferry",
	Short: "ferry is the Solus package repository tool",
}

// ListCmd is a parent for list type commands
var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "list [repos] [pool]",
}

// RemoveCmd is the parent for remove type commands
var RemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "remove [repo]",
}

func init() {
	RootCmd.AddCommand(ListCmd)
	RootCmd.AddCommand(RemoveCmd)
}
