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

package jobs

import (
    "bytes"
    "encoding/gob"
)

// JobType is a numerical representation of a kind of job
type JobType uint8

const (
	BulkAdd JobType = iota
	CreateRepo
	Delta
	DeltaRepo
	TransitProcess
)

// JobEntry is an entry in the JobQueue
type JobEntry struct {
	Type    JobType
	Claimed bool
    Params  interface{}
}

// Serialize uses Gob encoding to convert a JobEntry to a byte slice
func (j *JobEntry) Serialize() (result []byte, err error) {
    buff := bytes.NewBuffer(make([]byte,0))
    enc := gob.NewEncoder(buff)
    err = enc.Encode(*j)
    if err != nil {
        return
    }
    result = buff.Bytes()
    return
}

func Deserialize(serial []byte) (j JobEntry, err error) {
    buff := bytes.NewBuffer(serial)
    dec := gob.NewDecoder(buff)
    err = dec.Decode(j)
    return
}