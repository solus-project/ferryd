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
	"bytes"
	"encoding/json"
	"ferryd/jobs"
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

// GetRepos will attempt to serialise our known repositories into a response
func (s *Server) GetRepos(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req := libferry.RepoListingRequest{}
	repos, err := s.manager.GetRepos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, repo := range repos {
		req.Repository = append(req.Repository, repo.ID)
	}
	buf := bytes.Buffer{}
	if err := json.NewEncoder(&buf).Encode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(buf.Bytes())
}

// GetPoolItems will handle responding with the currently known pool items
func (s *Server) GetPoolItems(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req := libferry.PoolListingRequest{}
	pools, err := s.manager.GetPoolItems()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, pool := range pools {
		req.Item = append(req.Item, libferry.PoolItem{
			ID:       pool.Name,
			RefCount: int(pool.RefCount),
		})
	}
	buf := bytes.Buffer{}
	if err := json.NewEncoder(&buf).Encode(&req); err != nil {
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
	s.jproc.PushJob(jobs.NewCreateRepoJob(id))
}

// DeleteRepo will handle remote requests for repository deletion
func (s *Server) DeleteRepo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")
	log.WithFields(log.Fields{
		"id": id,
	}).Info("Repository deletion requested")
	s.jproc.PushJob(jobs.NewDeleteRepoJob(id))
}

// DeltaRepo will handle remote requests for repository deltaing
func (s *Server) DeltaRepo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")
	log.WithFields(log.Fields{
		"id": id,
	}).Info("Repository delta requested")
	s.jproc.PushJob(jobs.NewDeltaRepoJob(id))
}

// IndexRepo will handle remote requests for repository indexing
func (s *Server) IndexRepo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")
	log.WithFields(log.Fields{
		"id": id,
	}).Info("Repository indexing requested")
	s.jproc.PushJob(jobs.NewIndexRepoJob(id))
}

// ImportPackages will bulk-import the packages in the request
func (s *Server) ImportPackages(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")

	req := libferry.ImportRequest{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"id":        id,
		"npackages": len(req.Path),
	}).Info("Repository bulk import requested")

	s.jproc.PushJob(jobs.NewBulkAddJob(id, req.Path))
}

// CloneRepo will proxy a job to clone an existing repository
func (s *Server) CloneRepo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")

	req := libferry.CloneRepoRequest{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"source":    id,
		"target":    req.CloneName,
		"fullClone": req.CopyAll,
	}).Info("Repository clone requested")

	s.jproc.PushJob(jobs.NewCloneRepoJob(id, req.CloneName, req.CopyAll))
}

// PullRepo will proxy a job to pull an existing repository
func (s *Server) PullRepo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	target := p.ByName("id")

	req := libferry.PullRepoRequest{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"source": req.Source,
		"target": target,
	}).Info("Repository pull requested")

	s.jproc.PushJob(jobs.NewPullRepoJob(req.Source, target))
}

// RemoveSource will proxy a job to remove an existing set of packages by source name + relno
func (s *Server) RemoveSource(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	target := p.ByName("id")

	req := libferry.RemoveSourceRequest{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"source":  req.Source,
		"release": req.Release,
		"repo":    target,
	}).Info("Source removal requested")

	s.jproc.PushJob(jobs.NewRemoveSourceJob(target, req.Source, req.Release))
}
