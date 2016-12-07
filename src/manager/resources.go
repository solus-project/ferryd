//
// Copyright © 2016 Ikey Doherty <ikey@solus-project.com>
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

package manager

import (
	"errors"
)

var (
	// BucketNameRepos is the fixed name of the repositories bucket
	BucketNameRepos = []byte("repos")

	// BucketNamePool refers to our boltdb bucket in the Pool implementation
	BucketNamePool = []byte("pool")

	// ErrResourceExists is returned when the user attempts to create a
	// new resource, and a resource with the given name already exists.
	ErrResourceExists = errors.New("The specified resource already exists")

	// ErrUnknownResource is returned when the user attempts to delete a named
	// resource, but it was never stored to begin with.
	ErrUnknownResource = errors.New("The specified resource does not exist")
)
