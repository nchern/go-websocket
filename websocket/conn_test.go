// Copyright 2012 Gary Burd
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package websocket

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"testing"
	"testing/iotest"
	"time"
)

type fakeNetConn struct {
	io.Reader
	io.Writer
}

func (c fakeNetConn) Close() error                       { return nil }
func (c fakeNetConn) LocalAddr() net.Addr                { return nil }
func (c fakeNetConn) RemoteAddr() net.Addr               { return nil }
func (c fakeNetConn) SetDeadline(t time.Time) error      { return nil }
func (c fakeNetConn) SetReadDeadline(t time.Time) error  { return nil }
func (c fakeNetConn) SetWriteDeadline(t time.Time) error { return nil }

func TestFraming(t *testing.T) {
	frameSizes := []int{0, 1, 2, 124, 125, 126, 127, 128, 129, 65534, 65535, 65536, 65537}
	var readChunkers = []struct {
		name string
		f    func(io.Reader) io.Reader
	}{
		{"half", iotest.HalfReader},
		{"one", iotest.OneByteReader},
		{"asis", func(r io.Reader) io.Reader { return r }},
	}

	writeBuf := make([]byte, 65537)
	for i := range writeBuf {
		writeBuf[i] = byte(i)
	}

	for _, isServer := range []bool{true, false} {
		for _, chunker := range readChunkers {

			var connBuf bytes.Buffer
			wc := newConn(fakeNetConn{Reader: nil, Writer: &connBuf}, isServer, 1024, 1024)
			rc := newConn(fakeNetConn{Reader: chunker.f(&connBuf), Writer: nil}, !isServer, 1024, 1024)

			for _, n := range frameSizes {
				name := fmt.Sprintf("s:%b, r:%s, n:%d", isServer, chunker.name, n)

				w, err := wc.NextWriter(OpText)
				if err != nil {
					t.Errorf("%s: wc.NextWriter() returned %v", name, err)
					continue
				}
				nn, err := w.Write(writeBuf[:n])
				if err != nil || nn != n {
					t.Errorf("%s: w.Write(writeBuf[:n]) returned %d, %v", name, nn, err)
					continue
				}
				err = w.Close()
				if err != nil {
					t.Errorf("%s: w.Close() returned %v", name, err)
					continue
				}

				opCode, r, err := rc.NextReader()
				if err != nil || opCode != OpText {
					t.Errorf("%s: NextReader() returned %d, r, %v", name, opCode, err)
					continue
				}
				rbuf, err := ioutil.ReadAll(r)
				if err != nil {
					t.Errorf("%s: ReadFull() returned rbuf, %v", name, err)
					continue
				}

				if len(rbuf) != n {
					t.Errorf("%s: len(rbuf) is %d, want %d", len(rbuf), n)
					continue
				}

				for i, b := range rbuf {
					if byte(i) != b {
						t.Errorf("%s: bad byte at offset %d", name, i)
						break
					}
				}
			}
		}
	}
}