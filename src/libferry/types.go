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
	"time"
)

// Response is the base portion for all ferryd responses, and will
// include any relevant information on errors
type Response struct {
	Error       bool   // Whether this response is indication of an error
	ErrorString string // The associated error message
}

// A VersionRequest allows the client to request the current version string
// from the running daemon
type VersionRequest struct {
	Response
	Version string `json:"version"`
}

// An ImportRequest is given to ferryd to ask for the given packages to be
// included into the repository
type ImportRequest struct {
	Response
	Path []string `json:"path"`
}

// RepoListingRequest allows us to ask the remote what repositories it
// currently knows about.
type RepoListingRequest struct {
	Response
	Repository []string `json:"repos"`
}

// A PoolItem simply has an ID and a refcount, allowing us to examine our
// local storage efficiency.
type PoolItem struct {
	ID       string `json:"id"`
	RefCount int    `json:"refCount"`
}

// A PoolListingRequest is sent to get a listing of all pool items
type PoolListingRequest struct {
	Response
	Item []PoolItem `json:"items"`
}

// CloneRepoRequest is given to ferryd to ask it to clone one repo into another
type CloneRepoRequest struct {
	Response
	CloneName string `json:"cloneName"`
	CopyAll   bool   `json:"copyAll"` // Full clone
}

// PullRepoRequest is given to ferryd to ask it to from from one repo into another
type PullRepoRequest struct {
	Response
	Source string `json:"source"`
}

// RemoveSourceRequest is used to ask ferryd to remove all packages matching the
// given source and relno parameters
type RemoveSourceRequest struct {
	Response
	Source  string `json:"source"`
	Release int    `json:"relno"`
}

// CopySourceRequest is used to ask ferryd to copy all packages matching the
// given source and relno parameters
type CopySourceRequest struct {
	Response
	Target  string `json:"target"`
	Source  string `json:"source"`
	Release int    `json:"relno"`
}

// TrimPackagesRequest is sent when trimming excessive fat from a repository.
type TrimPackagesRequest struct {
	Response
	MaxKeep int `json:"maxPackages"`
}

// Job is used to represent status items in the backend
type Job struct {
	Description string
	// TODO: Add timing data!
}

// StatusRequest is used to grab information from the daemon, including its
// uptime
type StatusRequest struct {
	Response

	// When the daemon was first started, to work out uptime
	TimeStarted time.Time `json:"timeStarted"`

	FailedJobs  []Job // Known failed jobs
	CurrentJobs []Job // Currently registered jobs
}

// Uptime will determine the uptime of the daemon
func (s *StatusRequest) Uptime() time.Duration {
	return time.Now().UTC().Sub(s.TimeStarted)
}
