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

package main

import (
	"ferryd/core"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var (
	// If systemd is enabled, we'll talk to it.
	systemdEnabled = false

	// baseDir is where we expect to operate
	baseDir = "/var/lib/ferryd"

	// Default socket path we expect to use
	socketPath = "/run/ferryd.sock"
)

// RootCmd is the main entry point into ferry
var RootCmd = &cobra.Command{
	Use:   "ferryd",
	Short: "ferry is the Solus package repository daemon",
}

// Set up the main logger formatting used in USpin
func init() {
	form := &log.TextFormatter{}
	form.FullTimestamp = true
	form.TimestampFormat = "15:04:05"
	log.SetFormatter(form)
	RootCmd.PersistentFlags().StringVarP(&baseDir, "base", "d", "/var/lib/ferryd", "Set the base directory for ferryd")
	RootCmd.PersistentFlags().StringVarP(&socketPath, "socket", "s", "/run/ferryd.sock", "Set the socket path for ferryd")
}

func mainLoop() {
	log.Info("Initialising server")

	srv := NewServer()
	defer srv.Close()
	if e := srv.Bind(); e != nil {
		log.WithFields(log.Fields{
			"socket": srv.socketPath,
			"error":  e,
		}).Error("Error in binding server socket")
		return
	}
	if e := srv.Serve(); e != nil {
		log.WithFields(log.Fields{
			"socket": srv.socketPath,
			"error":  e,
		}).Error("Error in serving on socket")
		return
	}
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	// Must have a valid baseDir
	if !core.PathExists(baseDir) {
		log.WithFields(log.Fields{
			"directory": baseDir,
		}).Error("Base directory does not exist")
		os.Exit(1)
	}

	mainLoop()
}
