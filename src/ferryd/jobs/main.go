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

package jobs

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"ferryd/core"
	"fmt"
)

// JobType is a numerical representation of a kind of job
type JobType string

const (

	// BulkAdd is a sequential job which will attempt to add all of the packages
	BulkAdd JobType = "BulkAdd"

	// CreateRepo is a sequential job which will attempt to create a new repo
	CreateRepo = "CreateRepo"

	// Delta is a parallel job which will attempt the construction of deltas for
	// a given package name + repo
	Delta = "Delta"

	// DeltaRepo is a sequential job which creates Delta jobs for every package in
	// a repo
	DeltaRepo = "DeltaRepo"

	// IndexRepo is a sequential job that requests the repository be re-indexed
	IndexRepo = "IndexRepo"

	// TransitProcess is a sequential job that will process the incoming uploads
	// directory, dealing with each .tram upload
	TransitProcess = "TransitProcess"
)

// A JobHandler is created for each JobEntry, to provide specialised handling
// of the job type
type JobHandler interface {

	// Execute will attempt to execute the given job
	Execute(proc *Processor, m *core.Manager) error

	// Describe will return an appropriate description for the job
	Describe() string
}

// JobEntry is an entry in the JobQueue
type JobEntry struct {
	id         []byte // Unique ID for this job
	sequential bool   // Private to the job implementation
	Type       JobType
	Claimed    bool
	Params     []string
}

// Serialize uses Gob encoding to convert a JobEntry to a byte slice
func (j *JobEntry) Serialize() (result []byte, err error) {
	buff := &bytes.Buffer{}
	enc := gob.NewEncoder(buff)
	err = enc.Encode(j)
	if err != nil {
		return
	}
	result = buff.Bytes()
	return
}

// Deserialize use Gob decoding to convert a byte slice to a JobEntry
func Deserialize(serial []byte) (*JobEntry, error) {
	ret := &JobEntry{}
	buff := bytes.NewBuffer(serial)
	dec := gob.NewDecoder(buff)
	err := dec.Decode(ret)
	return ret, err
}

// GetID gets the true numerical ID for this job entry
func (j *JobEntry) GetID() uint64 {
	return binary.BigEndian.Uint64(j.id)
}

// NewJobHandler will return a handler that is loaded only during the execution
// of a previously serialised job
func NewJobHandler(j *JobEntry) (JobHandler, error) {
	switch j.Type {
	case BulkAdd:
		return NewBulkAddJobHandler(j)
	case CreateRepo:
		return NewCreateRepoJobHandler(j)
	case IndexRepo:
		return NewIndexRepoJobHandler(j)
	default:
		return nil, fmt.Errorf("unknown job type '%s'", j.Type)
	}
}
