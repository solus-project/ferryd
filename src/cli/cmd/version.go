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
	"ferry"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "show version",
	Long:  "Print the ferry version and exit",
	Run:   printVersion,
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

func printVersion(cmd *cobra.Command, args []string) {
	// Print local version
	fmt.Printf("ferry %v\n\nCopyright © 2016-2017 Solus Project\n", ferry.Version)
	fmt.Printf("Licensed under the Apache License, Version 2.0\n\n")

	// Attempt to grab the local daemon version
	client := ferry.NewClient("./ferryd.sock")
	defer client.Close()
	version, err := client.GetVersion()
	if err != nil {
		log.WithFields(log.Fields{
			"socket": "./ferryd.sock",
			"error":  err,
		}).Error("Cannot determine ferryd version")
		return
	}
	fmt.Printf("ferryd version: %v\n", version)
}
