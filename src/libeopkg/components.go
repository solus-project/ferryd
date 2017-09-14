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
	"encoding/xml"
	"os"
)

// A Component as seen through the eyes of XML
type Component struct {
	Name string // ID of this component, i.e. "system.base"

	// Translated short name
	LocalName []LocalisedField

	// Translated summary
	Summary []LocalisedField

	// Translated description
	Description []LocalisedField

	Group      string // Which group this component belongs to
	Maintainer struct {
		Name  string // Name of the component maintainer
		Email string // Contact e-mail address of component maintainer
	}
}

// Components is a simple helper wrapper for loading from components.xml files
type Components struct {
	Components []Component `xml:"Components>Component"`
}

// NewComponents will load the Components data from the XML file
func NewComponents(xmlfile string) (*Components, error) {
	fi, err := os.Open(xmlfile)
	if err != nil {
		return nil, err
	}
	defer fi.Close()
	components := &Components{}
	dec := xml.NewDecoder(fi)
	if err = dec.Decode(components); err != nil {
		return nil, err
	}

	// Ensure there are no empty Lang= fields
	for i := range components.Components {
		comp := &components.Components[i]
		FixMissingLocalLanguage(&comp.LocalName)
		FixMissingLocalLanguage(&comp.Summary)
		FixMissingLocalLanguage(&comp.Description)
	}
	return components, nil
}
