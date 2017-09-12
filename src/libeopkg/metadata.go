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

package libeopkg

import (
	"path/filepath"
	"strings"
)

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
	Homepage string   `xml:"Homepage,omitempty"` // From whence it came
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

// Action represents an action to take upon applying an update, such as restarting
// the system.
type Action struct {
	Value   string `xml:",chardata"`              // i.e. "systemRestart
	Package string `xml:"package,attr,omitempty"` // i.e. package="kernel"
}

// An Update forms part of a package's history, describing the version, release,
// etc, for each release of the package.
type Update struct {
	Release int    `xml:"release,attr"`        // Relno for this update
	Type    string `xml:"type,attr,omitempty"` // i.e. security
	Date    string // When the update was issued
	Version string // Version of the package at the time of this update
	Comment struct {
		Value string `xml:",cdata"` // Comment explaining  the update
	}
	Name struct {
		Value string `xml:",cdata"` // Updater's name
	}
	Email string // Updater's email

	Requires *[]Action `xml:"Requires>Action,omitempty"`
}

// A COMAR script
type COMAR struct {
	Value  string `xml:",chardata"`             // Value of the COMAR name
	Script string `xml:"script,attr,omitempty"` // The type of script
}

// Provides defines special items that might be exported by a package
type Provides struct {
	COMAR       []COMAR  `xml:"COMAR,omitempty"`
	PkgConfig   []string `xml:"PkgConfig,omitempty"`
	PkgConfig32 []string `xml:"PkgConfig32,omitempty"`
}

// Delta describes a delta package that may be used for an update to save on bandwidth
// for the users.
//
// Delta upgrades are determined by placing the <DeltaPackages> section into the index, with
// each Delta listed with a releaseFrom. If the user is currently using one of the listed
// releaseFrom IDs in their installation, that delta package will be selected instead of the
// full package.
type Delta struct {
	ReleaseFrom int    `xml:"releaseFrom,attr,omitempty"` // Delta from specified release to this one
	PackageURI  string // Relative location to the package
	PackageSize int64  // Actual size on disk of the .eopkg
	PackageHash string // Sha1sum for this package
}

// A MetaPackage is the Package section of the metadata file. It contains
// the main details that are important to users.
type MetaPackage struct {

	// Main details
	Name string // Name of this binary package
	// Brief description, one line, of the package functionality
	Summary []struct {
		Value string `xml:",cdata"`
		Lang  string `xml:"lang,attr,omitempty"`
	}
	// A full fleshed description of the package
	Description []struct {
		Value string `xml:",cdata"`
		Lang  string `xml:"lang,attr,omitempty"`
	}
	IsA                 string        `xml:"IsA,omitempty"`    // Legacy construct defining type
	PartOf              string        `xml:"PartOf,omitempty"` // Which component the package belongs to
	License             []string      // The package license(s)
	RuntimeDependencies *[]Dependency `xml:"RuntimeDependencies>Dependency,omitempty"` // Packages this package depends on at runtime
	Conflicts           *[]string     `xml:"Conflicts>Package,omitempty"`              // Conflicts with some package
	Replaces            *[]string     `xml:"Replaces>Package,omitempty"`               // Replaces the named package
	Provides            *Provides     `xml:"Provides,omitempty"`
	History             []Update      `xml:"History>Update"` // A series of updates to the package

	// Binary details
	BuildHost           string // Which build server produced the package
	Distribution        string // Identifier for the distribution, i.e. "Solus"
	DistributionRelease string // Name/ID if this distro release, i.e. "1"
	Architecture        string // i.e. x86_64
	InstalledSize       int64  // How much disk space this package takes up
	PackageSize         int64  // Actual size on disk of the .eopkg
	PackageHash         string // Sha1sum for this package
	PackageURI          string // Relative location to the package

	// DeltaPackages are only emitted in the index itself
	DeltaPackages *[]Delta `xml:"DeltaPackages>Delta,omitempty"`

	PackageFormat string // Locked to 1.2 for eopkg

	// TODO: Investigate why this is present in the metadata.xml!
	Source Source // Duplicate source to satisfy indexing
}

// GetID will return the package ID for ferryd
func (m *MetaPackage) GetID() string {
	return filepath.Base(m.PackageURI)
}

// GetRelease is a helpful wrapper to return the package's current release
func (m *MetaPackage) GetRelease() int {
	return m.History[0].Release
}

// GetVersion is a helpful wrapper to return the package's current version
func (m *MetaPackage) GetVersion() string {
	return m.History[0].Version
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

// GetPathComponent will get the source part of the string which is used
// in all subdirectories of the repository.
//
// For all packages with a source name of 4 or more characters, the path
// component will be split on this, i.e.:
//
//      libr/libreoffice
//
// For all other packages, the first letter of the source name is used, i.e.:
//
//      n/nano
//
func (m *MetaPackage) GetPathComponent() string {
	nom := strings.ToLower(m.Source.Name)
	letter := nom[0:1]
	var path string
	if strings.HasPrefix(nom, "lib") && len(nom) > 3 {
		path = filepath.Join(nom[0:4], nom)
	} else {
		path = filepath.Join(letter, nom)
	}
	return path
}
