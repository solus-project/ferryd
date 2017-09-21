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
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
)

var (
	// If systemd is enabled, we'll talk to it.
	systemdEnabled = false

	// baseDir is where we expect to operate
	baseDir = "/var/lib/ferryd"

	// Default socket path we expect to use
	socketPath = "/run/ferryd.sock"

	// How many jobs we're allowed to use. By default, half of the system cores (xz -T 2)
	backgroundJobCount = -1
)

const (
	// LockFilePath is created within the baseDir to assert ferryd instance ownership
	LockFilePath = "ferryd.lock"
)

func mainLoop() {
	pflag.StringVarP(&baseDir, "base", "d", "/var/lib/ferryd", "Set the base directory for ferryd")
	pflag.StringVarP(&socketPath, "socket", "s", "/run/ferryd.sock", "Set the socket path for ferryd")
	pflag.IntVarP(&backgroundJobCount, "jobs", "j", -1, "Number of jobs to use (-1 is 50% of cores)")
	pflag.Parse()

	// We write to a logfile..
	form := &log.TextFormatter{
		DisableColors: true,
	}

	form.FullTimestamp = true
	form.TimestampFormat = "15:04:05"
	log.SetFormatter(form)

	// Ensure all joined directories are correct
	b, err := filepath.Abs(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot resolve directory %v: %v\n", baseDir, err)
		os.Exit(1)
	}
	baseDir = b

	// Must have a valid baseDir
	if !core.PathExists(baseDir) {
		fmt.Fprintf(os.Stderr, "Base directory does not exist: %s\n", baseDir)
		os.Exit(1)
	}

	// Need to get a lock file before we can even grab the log file
	srv, err := NewServer()
	if err != nil {
		lockPath := filepath.Join(baseDir, LockFilePath)
		fmt.Fprintf(os.Stderr, "Failed to start ferryd: %v (lockfile: %v)\n", err, lockPath)
		os.Exit(1)
	}
	defer srv.Close()

	// We'll just keep logging for ever, don't expect rotation..
	logPath := filepath.Join(baseDir, "ferryd.log")
	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 00644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %s %v\n", logPath, err)
		os.Exit(1)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	// Now we can safely use logrus..
	log.Info("Initialising server")

	if err := srv.Bind(); err != nil {
		log.WithFields(log.Fields{
			"socket": srv.socketPath,
			"error":  err,
		}).Error("Error in binding server socket")
		fmt.Fprintf(os.Stderr, "Fatal error in socket bind, check logs: %v\n", err)
		return
	}
	if err := srv.Serve(); err != nil {
		log.WithFields(log.Fields{
			"socket": srv.socketPath,
			"error":  err,
		}).Error("Error in serving on socket")
		fmt.Fprintf(os.Stderr, "Fatal error in runtime execution, check logs: %v\n", err)
		return
	}
}

func main() {
	mainLoop()
}
