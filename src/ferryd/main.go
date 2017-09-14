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
	log "github.com/sirupsen/logrus"
)

// Set up the main logger formatting used in USpin
func init() {
	form := &log.TextFormatter{}
	form.FullTimestamp = true
	form.TimestampFormat = "15:04:05"
	log.SetFormatter(form)

	// Temp
	log.SetLevel(log.DebugLevel)
}

func mainLoop() {
	srv := NewServer()
	defer srv.Close()
	if e := srv.Bind(); e != nil {
		log.WithFields(log.Fields{
			"socket": UnixSocketPath,
			"error":  e,
		}).Error("Error in binding server socket")
		return
	}
	if e := srv.Serve(); e != nil {
		log.WithFields(log.Fields{
			"socket": UnixSocketPath,
			"error":  e,
		}).Error("Error in serving on socket")
		return
	}
}

func main() {
	log.Info("Initialising server")
	mainLoop()
}
