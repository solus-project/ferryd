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

package libferry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	// Version of the ferry client library
	Version = "0.0.1"
)

// A Client is used to communicate with the system ferryd
type Client struct {
	client *http.Client
}

// NewClient will return a new Client for the local unix socket, suitable
// for communicating with the daemon.
func NewClient(address string) *Client {
	return &Client{
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", address)
				},
				DisableKeepAlives:     false,
				IdleConnTimeout:       30 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
			Timeout: 20 * time.Second,
		},
	}
}

// Close will kill any idle connections still in "keep-alive" and ensure we're
// not leaking file descriptors.
func (c *Client) Close() {
	trans := c.client.Transport.(*http.Transport)
	trans.CloseIdleConnections()
}

func (c *Client) formURI(part string) string {
	return fmt.Sprintf("http://localhost.localdomain:0/%s", part)
}

// GetVersion will return the version of the remote daemon
func (c *Client) GetVersion() (string, error) {
	var vq VersionRequest
	resp, err := c.client.Get(c.formURI("api/v1/version"))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&vq); err != nil {
		return "", err
	}
	return vq.Version, nil
}

// GetRepos will grab a list of repos from the daemon
func (c *Client) GetRepos() ([]string, error) {
	var lq RepoListingRequest
	resp, err := c.client.Get(c.formURI("api/v1/list/repos"))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&lq); err != nil {
		return nil, err
	}
	return lq.Repository, nil
}

// GetPoolItems will grab a list of pool items from the daemon
func (c *Client) GetPoolItems() ([]PoolItem, error) {
	var lq PoolListingRequest
	resp, err := c.client.Get(c.formURI("api/v1/list/pool"))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&lq); err != nil {
		return nil, err
	}
	return lq.Item, nil
}

// A helper to wrap the trivial functionality, chaining off
// the appropriate errors, etc.
func (c *Client) getBasicResponse(url string, outT interface{}) error {
	resp, e := c.client.Get(url)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.ContentLength > 0 {
		if e = json.NewDecoder(resp.Body).Decode(outT); e != nil {
			return e
		}
	}
	fc := outT.(*Response)
	if !fc.Error {
		return nil
	}
	return errors.New(fc.ErrorString)
}

// A helper to wrap the trivial functionality, chaining off
// the appropriate errors, etc.
func (c *Client) postBasicResponse(url string, inT interface{}, outT interface{}) error {
	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)
	if err := enc.Encode(inT); err != nil {
		return err
	}

	resp, e := c.client.Post(url, "application/json; charset=utf-8", b)
	if e != nil {
		return e
	}

	defer resp.Body.Close()
	if resp.ContentLength > 0 {
		if e = json.NewDecoder(resp.Body).Decode(outT); e != nil {
			return e
		}
	}

	fc := outT.(*Response)
	if !fc.Error {
		return nil
	}
	return errors.New(fc.ErrorString)
}

// CreateRepo will attempt to create a repository in the daemon
func (c *Client) CreateRepo(id string) error {
	uri := c.formURI("/api/v1/create/repo/" + id)
	return c.getBasicResponse(uri, &Response{})
}

// DeleteRepo will attempt to delete a remote repository
func (c *Client) DeleteRepo(id string) error {
	uri := c.formURI("/api/v1/remove/repo/" + id)
	return c.getBasicResponse(uri, &Response{})
}

// DeltaRepo will attempt to reproduce deltas in the given repo
func (c *Client) DeltaRepo(id string) error {
	uri := c.formURI("/api/v1/delta/repo/" + id)
	return c.getBasicResponse(uri, &Response{})
}

// IndexRepo will attempt to index a repository in the daemon
func (c *Client) IndexRepo(id string) error {
	uri := c.formURI("/api/v1/index/repo/" + id)
	return c.getBasicResponse(uri, &Response{})
}

// ImportPackages will ask ferryd to import the named packages with absolute
// paths
func (c *Client) ImportPackages(repoID string, pkgs []string) error {
	iq := ImportRequest{
		Path: pkgs,
	}
	return c.postBasicResponse(c.formURI("api/v1/import/"+repoID), &iq, &Response{})
}

// CloneRepo will ask the backend to clone an existing repository into a new repository
func (c *Client) CloneRepo(repoID, newClone string, copyAll bool) error {
	cq := CloneRepoRequest{
		CloneName: newClone,
		CopyAll:   copyAll,
	}
	return c.postBasicResponse(c.formURI("api/v1/clone/"+repoID), &cq, &Response{})
}

// PullRepo will ask the backend to pull from target into repoID
func (c *Client) PullRepo(sourceID, targetID string) error {
	pq := PullRepoRequest{
		Source: sourceID,
	}
	return c.postBasicResponse(c.formURI("api/v1/pull/"+targetID), &pq, &Response{})
}

// RemoveSource will ask the backend to remove packages by source name
func (c *Client) RemoveSource(repoID, sourceID string, relno int) error {
	sq := RemoveSourceRequest{
		Source:  sourceID,
		Release: relno,
	}
	return c.postBasicResponse(c.formURI("api/v1/remove/source/"+repoID), &sq, &Response{})
}

// TrimPackages will request that packages in the repo are trimmed to maxKeep
func (c *Client) TrimPackages(repoID string, maxKeep int) error {
	tq := TrimPackagesRequest{
		MaxKeep: maxKeep,
	}
	return c.postBasicResponse(c.formURI("api/v1/trim/packages/"+repoID), &tq, &Response{})
}

// TrimObsolete will request that all packages marked obsolete are removed
func (c *Client) TrimObsolete(repoID string) error {
	uri := c.formURI("/api/v1/trim/obsoletes/" + repoID)
	return c.getBasicResponse(uri, &Response{})
}
