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
	"github.com/radu-munteanu/fsnotify"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

// InitWatcher will set up the watcher for the first time
func (s *Server) InitWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	// Monitor the incoming dir
	if err = watcher.Add(s.manager.IncomingPath); err != nil {
		return err
	}
	s.watchChan = make(chan bool)
	s.watcher = watcher
	return nil
}

// WatchIncoming will wait for events on the incoming directory
// and process incoming .tram files
func (s *Server) WatchIncoming() {
	s.watchGroup.Add(1)
	go func() {
		defer s.watchGroup.Done()
		for {
			select {
			case event := <-s.watcher.Events:
				// Not interested in subdirs
				if filepath.Dir(event.Name) != s.manager.IncomingPath {
					continue
				}
				if event.Op&fsnotify.Write|fsnotify.Close == fsnotify.Write|fsnotify.Close {
					if strings.HasSuffix(event.Name, core.TransitManifestSuffix) {
						s.processTransitManifest(filepath.Base(event.Name))
					}
				}
			case <-s.watchChan:
				return
			}
		}
	}()
}

// StopWatching will force the fsnotify code to shut down
func (s *Server) StopWatching() {
	s.watchChan <- true
	s.watchGroup.Wait()
}

// processTransitManifest is invoked when a .tram file is closed in our incoming
// directory. We'll now push it for further processing
func (s *Server) processTransitManifest(name string) {
	fullpath := filepath.Join(s.manager.IncomingPath, name)

	st, err := os.Stat(fullpath)
	if err != nil {
		return
	}

	if !st.Mode().IsRegular() {
		return
	}

	log.WithFields(log.Fields{
		"id": name,
	}).Info("Received transit manifest upload")
	// s.jproc.PushJob(jobs.NewTransitProcessJob(fullpath))
}
