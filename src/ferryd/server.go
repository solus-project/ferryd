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
	"errors"
	"ferryd/core"
	"ferryd/jobs"
	"github.com/julienschmidt/httprouter"
	"github.com/radu-munteanu/fsnotify"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
)

const (
	// UnixSocketPath is the unique socket path on the system for the ferry daemon
	UnixSocketPath = "./ferryd.sock"
)

// Server sits on a unix socket accepting connections from authenticated
// client, i.e. root or those in the "ferry" group
type Server struct {
	srv     *http.Server
	running bool
	router  *httprouter.Router
	socket  net.Listener

	manager    *core.Manager     // heart of the story
	store      *jobs.JobStore    // Storage for jobs processor
	jproc      *jobs.Processor   // Allow scheduling jobs
	watcher    *fsnotify.Watcher // Monitor incoming uploads
	watchChan  chan bool         // Allow terminating the watcher
	watchGroup *sync.WaitGroup   // Allow blocking watch terminate.
}

// NewServer will return a newly initialised Server which is currently unbound
func NewServer() *Server {
	router := httprouter.New()
	s := &Server{
		srv: &http.Server{
			Handler: router,
		},
		running:    false,
		router:     router,
		watchGroup: &sync.WaitGroup{},
	}
	// Set up the API bits
	router.GET("/api/v1/version", s.GetVersion)

	// Repo management
	router.GET("/api/v1/create_repo/:id", s.CreateRepo)
	router.GET("/api/v1/delta_repo/:id", s.DeltaRepo)
	router.GET("/api/v1/index_repo/:id", s.IndexRepo)
	router.POST("/api/v1/import/:id", s.ImportPackages)
	return s
}

// killHandler will ensure we cleanly tear down on a ctrl+c/sigint
func (s *Server) killHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		log.Warning("Shutting down due to CTRL+C")
		s.Close()
		// Stop any mainLoop defers here
		os.Exit(1)
	}()
}

// Bind will attempt to set up the listener on the unix socket
// prior to serving.
func (s *Server) Bind() error {
	l, e := net.Listen("unix", UnixSocketPath)
	if e != nil {
		return e
	}

	baseDir := "./ferry"

	// Create new Slip Manager for the "./ferry" repo
	if err := os.MkdirAll(baseDir, 00755); err != nil {
		return err
	}
	m, e := core.NewManager(baseDir)
	if e != nil {
		return e
	}
	s.manager = m

	st, e := jobs.NewStore(baseDir)
	if e != nil {
		return e
	}
	s.store = st

	// TODO: Expose setting for background job count
	s.jproc = jobs.NewProcessor(s.manager, s.store, -1)

	// Set up watching the manager's incoming directory
	if err := s.InitWatcher(); err != nil {
		return err
	}

	uid := os.Geteuid()
	gid := os.Getegid()
	// Avoid umask issues
	if e = os.Chown(UnixSocketPath, uid, gid); e != nil {
		return e
	}
	// Fatal if we cannot chmod the socket to be ours only
	if e = os.Chmod(UnixSocketPath, 0600); e != nil {
		return e
	}
	s.socket = l
	return nil
}

// Serve will continuously serve on the unix socket until dead
func (s *Server) Serve() error {
	if s.socket == nil {
		return errors.New("Cannot serve without a bound server socket")
	}
	s.running = true
	s.killHandler()
	defer func() {
		s.running = false
	}()
	// Serve the job queue
	s.jproc.Begin()
	s.WatchIncoming()
	// Don't treat Shutdown/Close as an error, it's intended by us.
	if e := s.srv.Serve(s.socket); e != http.ErrServerClosed {
		return e
	}
	return nil
}

// Close will shut down and cleanup the socket
func (s *Server) Close() {
	if !s.running {
		return
	}
	s.StopWatching()
	s.jproc.Close()
	s.store.Close()
	s.manager.Close()
	s.running = false
	s.srv.Shutdown(nil)
	os.Remove(UnixSocketPath)
}
