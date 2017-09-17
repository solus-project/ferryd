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

package libdb

import (
	"testing"
)

type embeddedGob struct {
	ID int
}

type gobTester struct {
	Name     string
	Age      int
	AllocGob *embeddedGob
	EmbedGob embeddedGob
}

func TestGobTranscoder(t *testing.T) {
	encoder := NewGobTranscoder()

	tmpGob := &gobTester{
		Name: "John",
		Age:  57,
		AllocGob: &embeddedGob{
			ID: 12,
		},
		EmbedGob: embeddedGob{
			ID: 14,
		},
	}

	b, err := encoder.EncodeType(tmpGob)
	if err != nil {
		t.Fatalf("Failed to encode basic type: %v", err)
	}
	var decoded gobTester
	if err = encoder.DecodeType(b, &decoded); err != nil {
		t.Fatalf("Failed to decode basic type: %v", err)
	}
	if decoded.Name != "John" {
		t.Fatalf("Expected name 'John', got '%s'", decoded.Name)
	}
	if decoded.Age != 57 {
		t.Fatalf("Expected age '57', got '%d'", decoded.Age)
	}
	if decoded.AllocGob == nil {
		t.Fatal("Missing AllocGob in decode")
	}
	if decoded.AllocGob.ID != 12 {
		t.Fatalf("Expected AllocGob ID of '12', got '%d'", decoded.AllocGob.ID)
	}
	if decoded.EmbedGob.ID != 14 {
		t.Fatalf("Expected AllocGob ID of '14', got '%d'", decoded.EmbedGob.ID)
	}
}
