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

// A Group as seen through the eyes of XML
type Group struct {
	Name string // ID of this group, i.e. "multimedia"

	// Translated short name
	LocalName []struct {
		Value string `xml:",cdata"`
		Lang  string `xml:"lang,attr,omitempty"`
	}

	Icon string // Display icon for this Group
}

// Groups is a simple helper wrapper for loading from components.xml files
type Groups struct {
	Groups []Group `xml:"Groups>Group"`
}

// NewGroups will load the Groups data from the XML file
func NewGroups(xmlfile string) (*Groups, error) {
	fi, err := os.Open(xmlfile)
	if err != nil {
		return nil, err
	}
	defer fi.Close()
	grp := &Groups{}
	dec := xml.NewDecoder(fi)
	if err = dec.Decode(grp); err != nil {
		return nil, err
	}
	return grp, nil
}
