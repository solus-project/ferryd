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
	Use:   "list  [repos] [pool]",
	Short: "list",
}

// RemoveCmd is the parent for remove type commands
var RemoveCmd = &cobra.Command{
	Use:   "remove [repo] [source]",
	Short: "remove",
}

// ResetCmd is the parent for reset type commands
var ResetCmd = &cobra.Command{
	Use:   "reset [failed] [completed]",
	Short: "reset job logs",
}

// CopyCmd is the parent for copy type commands
var CopyCmd = &cobra.Command{
	Use:   "copy [source]",
	Short: "copy",
}

// TrimCmd is the parent for trim type commands
var TrimCmd = &cobra.Command{
	Use:   "trim [packages] [obsoletes]",
	Short: "trim",
}

var (
	// Default location for the unix socket
	socketPath = "/run/ferryd.sock"
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&socketPath, "socket", "s", "/run/ferryd.sock", "Set the socket path to talk to ferryd")

	RootCmd.AddCommand(CopyCmd)
	RootCmd.AddCommand(ListCmd)
	RootCmd.AddCommand(RemoveCmd)
	RootCmd.AddCommand(ResetCmd)
	RootCmd.AddCommand(TrimCmd)
}
