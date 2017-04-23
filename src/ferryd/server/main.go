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

package server

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
)

const (
	// UnixSocketPath is the unique socket path on the system for the ferry daemon
	UnixSocketPath = "./ferryd.sock"
)

// Server sits on a unix socket accepting connections from authenticated
// client, i.e. root or those in the "ferry" group
type Server struct {
	socket  net.Listener
	srv     *http.Server
	running bool
}

// New will return a newly initialised Server which is currently unbound
func New() *Server {
	return &Server{
		srv:     &http.Server{},
		running: false,
	}
}

// killHandler will ensure we cleanly tear down on a ctrl+c/sigint
func (s *Server) killHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		fmt.Fprintf(os.Stderr, " -> shutting down due to ctrl+c\n")
		s.Close()
		// Stop any mainLoop defers here
		os.Exit(1)
	}()
}

// Serve will continuously serve on the unix socket until dead
func (s *Server) Serve() error {
	l, e := net.Listen("unix", UnixSocketPath)
	if e != nil {
		return e
	}
	s.socket = l
	s.running = true
	s.killHandler()
	defer func() {
		s.running = false
	}()
	e = s.srv.Serve(l)
	// Don't treat Shutdown/Close as an error, it's intended by us.
	if e != http.ErrServerClosed {
		return e
	}
	return nil
}

// Close will shut down and cleanup the socket
func (s *Server) Close() {
	if !s.running {
		return
	}
	s.running = false
	s.srv.Shutdown(nil)
	s.socket.Close()
	os.Remove(UnixSocketPath)
}
