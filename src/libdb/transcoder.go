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
	"bytes"
	"encoding/gob"
	"io"
)

// GobEncoderLight is a helper for encoding gob
type GobEncoderLight struct {
	bytes   *bytes.Buffer
	encoder *gob.Encoder
}

// GobDecoderLight is a helper for decoding gob
type GobDecoderLight struct {
	bytes   *bytes.Buffer
	decoder *gob.Decoder
}

// NewGobEncoderLight returns a new lock-free encoder
func NewGobEncoderLight() *GobEncoderLight {
	ret := &GobEncoderLight{
		bytes: &bytes.Buffer{},
	}
	ret.encoder = gob.NewEncoder(ret.bytes)
	return ret
}

// NewGobDecoderLight returns a new lock-free decoder
func NewGobDecoderLight() *GobDecoderLight {
	ret := &GobDecoderLight{
		bytes: &bytes.Buffer{},
	}
	ret.decoder = gob.NewDecoder(ret.bytes)
	return ret
}

// EncodeType will convert give given pointer into a gob encoded
// byte set, and return them
func (g *GobEncoderLight) EncodeType(t interface{}) ([]byte, error) {
	defer func() {
		g.bytes.Reset()
	}()
	err := g.encoder.Encode(t)
	if err != nil {
		return nil, err
	}
	return g.bytes.Bytes(), nil
}

// DecodeType will attempt to decode the buffer into the pointer outT
func (g *GobDecoderLight) DecodeType(buf []byte, outT interface{}) error {
	defer func() {
		g.bytes.Reset()
	}()
	reader := bytes.NewReader(buf)
	if _, err := io.Copy(g.bytes, reader); err != nil {
		return err
	}
	return g.decoder.Decode(outT)
}
