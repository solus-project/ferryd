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
	"errors"
	"net"
	"net/http"
	"time"
)

const (
	// Version of the ferry client library
	Version = "0.0.1"
)

// A FerryClient is used to communicate with the system ferryd
type FerryClient struct {
	client *http.Client
}

// NewClient will return a new FerryClient for the local unix socket, suitable
// for communicating with the daemon.
func NewClient(address string) *FerryClient {
	return &FerryClient{
		client: &http.Client{
			Transport: &http.Transport{
				Dial: func(protocol, address string) (net.Conn, error) {
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
func (f *FerryClient) Close() {
	trans := f.client.Transport.(*http.Transport)
	trans.CloseIdleConnections()
}

// GetVersion will return the version of the remote daemon
func (f *FerryClient) GetVersion() (string, error) {
	return "", errors.New("Not yet implemented")
}
