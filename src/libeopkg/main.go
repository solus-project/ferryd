//
// Copyright © 2016-2017 Solus Project
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

// Package libeopkg provides Go-native access to `.eopkg` files, allowing
// ferryd to read and manipulate them without having a host-side eopkg
// tool.
//
// It should also be noted that `eopkg` is implemented in Python, so calling
// out to the host-side tool just isn't acceptable for the performance we
// require.
// In time, `sol` will replace eopkg and it is very likely that we'll base
// the new `libsol` component on the C library using cgo.
package libeopkg

import (
	"errors"
)

var (
	// ErrNotYetImplemented is a placeholder during development
	ErrNotYetImplemented = errors.New("Not yet implemented")

	// ErrEopkgCorrupted is provided when a file does not conform to eopkg spec
	ErrEopkgCorrupted = errors.New(".eopkg file is corrupted or invalid")
)

// LocalisedField is used in various parts of the eopkg metadata to provide
// a field value with an xml:lang attribute describing the language
type LocalisedField struct {
	Value string `xml:",cdata"`
	Lang  string `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
}

// FixMissingLocalLanguage should be used on a set of LocalisedField to restore
// the missing "en" that is required in the very first field set.
func FixMissingLocalLanguage(fields *[]LocalisedField) {
	if fields == nil {
		return
	}
	field := &(*fields)[0]
	if field.Lang == "" {
		field.Lang = "en"
	}
}
