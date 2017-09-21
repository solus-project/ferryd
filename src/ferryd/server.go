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
	"github.com/coreos/go-systemd/activation"
	"github.com/coreos/go-systemd/daemon"
	"github.com/julienschmidt/httprouter"
	"github.com/radu-munteanu/fsnotify"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// Server sits on a unix socket accepting connections from authenticated
// client, i.e. root or those in the "ferry" group
type Server struct {
	srv     *http.Server
	running bool
	router  *httprouter.Router
	socket  net.Listener

	// We store a global lock file ..
	lockFile *LockFile
	lockPath string

	// When we first started up.
	timeStarted time.Time

	manager    *core.Manager     // heart of the story
	store      *jobs.JobStore    // Storage for jobs processor
	jproc      *jobs.Processor   // Allow scheduling jobs
	watcher    *fsnotify.Watcher // Monitor incoming uploads
	watchChan  chan bool         // Allow terminating the watcher
	watchGroup *sync.WaitGroup   // Allow blocking watch terminate.
	socketPath string
}

// NewServer will return a newly initialised Server which is currently unbound
func NewServer() (*Server, error) {
	router := httprouter.New()
	s := &Server{
		srv: &http.Server{
			Handler: router,
		},
		running:     false,
		router:      router,
		timeStarted: time.Now().UTC(),
		watchGroup:  &sync.WaitGroup{},
	}

	// Before we can actually bind the socket, we must lock the file
	s.lockPath = filepath.Join(baseDir, LockFilePath)
	lfile, err := NewLockFile(s.lockPath)
	s.lockFile = lfile

	if err != nil {
		return nil, err
	}

	// Try to lock our lockfile now
	if err := s.lockFile.Lock(); err != nil {
		return nil, err
	}

	// Set up the API bits
	router.GET("/api/v1/status", s.GetStatus)

	// Repo management
	router.GET("/api/v1/create/repo/:id", s.CreateRepo)
	router.GET("/api/v1/remove/repo/:id", s.DeleteRepo)
	router.GET("/api/v1/delta/repo/:id", s.DeltaRepo)
	router.GET("/api/v1/index/repo/:id", s.IndexRepo)

	// Client sends us data
	router.POST("/api/v1/import/:id", s.ImportPackages)
	router.POST("/api/v1/clone/:id", s.CloneRepo)
	router.POST("/api/v1/copy/source/:id", s.CopySource)
	router.POST("/api/v1/pull/:id", s.PullRepo)

	// Removal
	router.POST("/api/v1/remove/source/:id", s.RemoveSource)
	router.POST("/api/v1/trim/packages/:id", s.TrimPackages)
	router.GET("/api/v1/trim/obsoletes/:id", s.TrimObsolete)

	// List commands
	router.GET("/api/v1/list/repos", s.GetRepos)
	router.GET("/api/v1/list/pool", s.GetPoolItems)
	return s, nil
}

// killHandler will ensure we cleanly tear down on a ctrl+c/sigint
func (s *Server) killHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		log.Warning("ferryd shutting down")
		s.Close()
		// Stop any mainLoop defers here
		os.Exit(1)
	}()
}

// Bind will attempt to set up the listener on the unix socket
// prior to serving.
func (s *Server) Bind() error {
	var listener net.Listener

	// Set from global CLI flag
	s.socketPath = socketPath

	// Check if we're systemd activated.
	if _, b := os.LookupEnv("LISTEN_FDS"); b {
		listeners, err := activation.Listeners(true)
		if err != nil {
			return err
		}
		if len(listeners) != 1 {
			return errors.New("expected a single unix socket")
		}
		// listener will be sockets[0], now we'll need to follow systemd activation path
		listener = listeners[0]
		// Mustn't delete!
		if unix, ok := listener.(*net.UnixListener); ok {
			unix.SetUnlinkOnClose(false)
		} else {
			return errors.New("expected unix socket")
		}
		systemdEnabled = true
	} else {
		l, e := net.Listen("unix", s.socketPath)
		if e != nil {
			return e
		}
		listener = l
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

	s.jproc = jobs.NewProcessor(s.manager, s.store, backgroundJobCount)

	// Set up watching the manager's incoming directory
	if err := s.InitWatcher(); err != nil {
		return err
	}

	uid := os.Getuid()
	gid := os.Getgid()
	if !systemdEnabled {
		// Avoid umask issues
		if e = os.Chown(s.socketPath, uid, gid); e != nil {
			return e
		}
		// Fatal if we cannot chmod the socket to be ours only
		if e = os.Chmod(s.socketPath, 0660); e != nil {
			return e
		}
	}
	s.socket = listener
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

	if systemdEnabled {
		daemon.SdNotify(false, "READY=1")
	}

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
	if s.lockFile != nil {
		s.lockFile.Unlock()
		s.lockFile.Clean()
		s.lockFile = nil
	}
	s.StopWatching()
	s.jproc.Close()
	s.store.Close()
	s.manager.Close()
	s.running = false
	s.srv.Shutdown(nil)

	// We don't technically fully own it if systemd created it
	if !systemdEnabled {
		os.Remove(s.socketPath)
	}
}
