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

package ferry

import (
	"context"
	"encoding/json"
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
func (f *Client) Close() {
	trans := f.client.Transport.(*http.Transport)
	trans.CloseIdleConnections()
}

func (f *Client) formURI(part string) string {
	return fmt.Sprintf("http://localhost.localdomain:0/%s", part)
}

// GetVersion will return the version of the remote daemon
func (f *Client) GetVersion() (string, error) {
	var vq VersionRequest
	resp, e := f.client.Get(f.formURI("api/v1/version"))
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	if e = json.NewDecoder(resp.Body).Decode(&vq); e != nil {
		return "", e
	}
	return vq.Version, nil
}
