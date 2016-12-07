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

package libeopkg

// A Packager identifies the person responsible for maintaining the source
// package. In terms of ypkg builds, it will indicate the last person who
// made a change to the package, allowing a natural "blame" system to work
// much like git.
type Packager struct {
	Name  string // Packager's name
	Email string // Packager's email address
}

// Source provides the information relating to the source package within
// each binary package.
// This source identifies one or more packages coming from the same origin,
// i.e they have the same *source name*.
type Source struct {
	Name     string   // Source name
	Packager Packager // Who is responsible for packaging this source.
}

// A Dependency has various attributes which help determine what needs to
// be installed when updating or installing the package.
type Dependency struct {
	Name string `xml:",chardata"` // Value of the dependency

	// Release based dependencies
	ReleaseFrom int `xml:"releaseFrom,attr,omitempty"` // >= release
	ReleaseTo   int `xml:"releaseTo,attr,omitempty"`   // <= release
	Release     int `xml:"release,attr,omitempty"`     // == release

	// Version based dependencies
	VersionFrom string `xml:"versionFrom,attr,omitempty"` // >= version
	VersionTo   string `xml:"versionTo,attr,omitempty"`   // <= version
	Version     string `xml:"version,attr,omitempty"`     // == version
}

// An Update forms part of a package's history, describing the version, release,
// etc, for each release of the package.
type Update struct {
	Release int    `xml:"release,attr"`        // Relno for this update
	Type    string `xml:"type,attr,omitempty"` // i.e. security
	Date    string // When the update was issued
	Version string // Version of the package at the time of this update
	Comment string // Comment explaining  the update
	Name    string // Updater's name
	Email   string // Updater's email
}

// A MetaPackage is the Package section of the metadata file. It contains
// the main details that are important to users.
type MetaPackage struct {

	// Main details
	Name                string       // Name of this binary package
	Summary             string       // Brief description, one line, of the package functionality
	Description         string       // A full fleshed description of the package
	RuntimeDependencies []Dependency // Packages this package depends on at runtime
	PartOf              string       // Which component the package belongs to
	License             []string     // The package license(s)
	History             []Update     `xml:"History>Update"` // A series of updates to the package

	// Binary details
	BuildHost          string // Which build server produced the package
	Distributon        string // Identifier for the distribution, i.e. "Solus"
	DistributonRelease string // Name/ID if this distro release, i.e. "1"
	Architecture       string // i.e. x86_64
	InstalledSize      int64  // How much disk space this package takes up
	PackageFormat      string // Locked to 1.2 for eopkg

	// TODO: Investigate why this is present in the metadata.xml!
	Source Source // Duplicate source to satisfy indexing
}

// Metadata contains all of the information a package can provide to a user
// prior to installation. This includes the name, version, release, and so
// forth.
//
// Every Package contains Metadata, and during eopkg indexing, a reduced
// version of the Metadata is emitted.
type Metadata struct {
	Source  Source      // Source of this package
	Package MetaPackage `xml:"Package"` // Meta on the package itself
}
