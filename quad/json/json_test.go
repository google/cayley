// Copyright 2014 The Cayley Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package json

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cayleygraph/cayley/quad"
)

var readTests = []struct {
	message string
	input   string
	expect  []quad.Quad
	err     error
}{
	{
		message: "parse correct JSON",
		input: `[
			{"subject": "foo", "predicate": "bar", "object": "baz"},
			{"subject": "foo", "predicate": "bar", "object": "baz", "label": "graph"}
		]`,
		expect: []quad.Quad{
			quad.MakeRaw("foo", "bar", "baz", ""),
			quad.MakeRaw("foo", "bar", "baz", "graph"),
		},
		err: nil,
	},
	{
		message: "parse correct JSON with extra field",
		input: `[
			{"subject": "foo", "predicate": "bar", "object": "foo", "something_else": "extra data"}
		]`,
		expect: []quad.Quad{
			quad.MakeRaw("foo", "bar", "foo", ""),
		},
		err: nil,
	},
	{
		message: "reject incorrect JSON",
		input: `[
			{"subject": "foo", "predicate": "bar"}
		]`,
		expect: nil,
		err:    fmt.Errorf("invalid quad at index %d. %v", 0, quad.MakeRaw("foo", "bar", "", "")),
	},
}

func TestReadJSON(t *testing.T) {
	for _, test := range readTests {
		qr := NewReader(strings.NewReader(test.input))
		got, err := quad.ReadAll(qr)
		qr.Close()
		if fmt.Sprint(err) != fmt.Sprint(test.err) {
			t.Errorf("Failed to %v with unexpected error, got:%v expected %v", test.message, err, test.err)
		}
		if !reflect.DeepEqual(got, test.expect) {
			t.Errorf("Failed to %v, got:%v expect:%v", test.message, got, test.expect)
		}
	}
}

var writeTests = []struct {
	message string
	input   []quad.Quad
	expect  string
	err     error
}{
	{
		message: "write empty JSON",
		input:   []quad.Quad{},
		expect:  "null\n",
		err:     nil,
	},
	{
		message: "write JSON",
		input: []quad.Quad{
			quad.MakeRaw("foo", "bar", "baz", ""),
			quad.MakeRaw("foo", "bar", "baz", "graph"),
		},
		expect: `[
	{"subject":"foo","predicate":"bar","object":"baz"},
	{"subject":"foo","predicate":"bar","object":"baz","label":"graph"}
]
`,
		err: nil,
	},
}

func TestWriteJSON(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	for _, test := range writeTests {
		buf.Reset()
		qw := NewWriter(buf)
		_, err := quad.Copy(qw, quad.NewReader(test.input))
		if err != nil {
			t.Errorf("Failed to %v: %v", test.message, err)
			continue
		}
		qw.Close()
		if fmt.Sprint(err) != fmt.Sprint(test.err) {
			t.Errorf("Failed to %v with unexpected error, got:%v expected %v", test.message, err, test.err)
		}
		if got := buf.String(); got != test.expect {
			t.Errorf("Failed to %v, got:%v expect:%v", test.message, got, test.expect)
		}
	}
}
