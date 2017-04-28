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

// Package slip provides the Ferry Slip implementation.
//
// This portion of ferryd is responsible for the management of management
// of the repositories, and receives packages from the builders.
// In the ferryd design, packages are ferried to the slip, where it is then
// organised into the repositories.
package slip

const (
	// DatabasePathComponent is the suffix applied to a working directory
	// for the database file itself.
	DatabasePathComponent = "ferry.db"
)
