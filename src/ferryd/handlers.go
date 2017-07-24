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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"libferry"
	"net/http"
	"runtime"
)

// getMethodOrigin helps us determine the caller so that we can print
// an appropriate method name into the log without tons of boilerplate
func getMethodCaller() string {
	n, _, _, ok := runtime.Caller(2)
	if !ok {
		return ""
	}
	if details := runtime.FuncForPC(n); details != nil {
		return details.Name()
	}
	return ""
}

// sendStockError is a utility to send a standard response to the ferry
// client that embeds the error message from ourside.
func (s *Server) sendStockError(err error, w http.ResponseWriter, r *http.Request) {
	response := libferry.Response{
		Error:       true,
		ErrorString: err.Error(),
	}
	log.WithFields(log.Fields{
		"error":  err,
		"method": getMethodCaller(),
	}).Error("Client communication error")
	buf := bytes.Buffer{}
	if e2 := json.NewEncoder(&buf).Encode(&response); e2 != nil {
		http.Error(w, e2.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	w.Write(buf.Bytes())
}

// GetVersion will return the current version of the ferryd
func (s *Server) GetVersion(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// For now return nothing and default to 200 OK
	fmt.Printf("Got a version request: %v\n", r.URL.Path)

	vq := libferry.VersionRequest{Version: libferry.Version}
	buf := bytes.Buffer{}
	if err := json.NewEncoder(&buf).Encode(&vq); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(buf.Bytes())
}

// CreateRepo will handle remote requests for repository creation
func (s *Server) CreateRepo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")
	log.WithFields(log.Fields{
		"id": id,
	}).Info("Repository creation requested")
	err := s.manager.CreateRepo(id)
	// TODO: Make this Moar Better..
	if err != nil {
		s.sendStockError(err, w, r)
	}
}
