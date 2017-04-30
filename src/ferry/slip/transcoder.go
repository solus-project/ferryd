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

package slip

import (
	"bytes"
	"encoding/gob"
	"io"
	"sync"
)

// A GobTranscoder is a reusable encoder object which is designed to
// wrap the Gob encoding API, in a simple fashion for ferryd, while
// avoiding high usage costs.
type GobTranscoder struct {
	// Encoding
	encoder    *gob.Encoder
	outBytes   *bytes.Buffer
	encoderMut *sync.Mutex

	// Decoding
	decoder    *gob.Decoder
	inBytes    *bytes.Buffer
	decoderMut *sync.Mutex
}

// NewGobTranscoder will return a newly initialised transcoder to help
// with the mundane encoding/decoding operations
func NewGobTranscoder() *GobTranscoder {
	ret := &GobTranscoder{
		inBytes:    &bytes.Buffer{},
		outBytes:   &bytes.Buffer{},
		encoderMut: &sync.Mutex{},
		decoderMut: &sync.Mutex{},
	}
	ret.encoder = gob.NewEncoder(ret.outBytes)
	ret.decoder = gob.NewDecoder(ret.inBytes)
	return ret
}

// EncodeType will convert give given pointer into a gob encoded
// byte set, and return them
func (g *GobTranscoder) EncodeType(t interface{}) ([]byte, error) {
	g.encoderMut.Lock()
	defer func() {
		g.outBytes.Reset()
		g.encoderMut.Unlock()
	}()
	err := g.encoder.Encode(t)
	if err != nil {
		return nil, err
	}
	return g.outBytes.Bytes(), nil
}

// DecodeType will attempt to decode the buffer into the pointer outT
func (g *GobTranscoder) DecodeType(buf []byte, outT interface{}) error {
	g.decoderMut.Lock()
	defer func() {
		g.inBytes.Reset()
		g.decoderMut.Unlock()
	}()
	reader := bytes.NewReader(buf)
	if _, err := io.Copy(g.inBytes, reader); err != nil {
		return err
	}
	return g.decoder.Decode(outT)
}
