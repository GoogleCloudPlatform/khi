// Copyright 2024 Google LLC
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

package log

import (
	"fmt"
	"sync"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/adapter"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
)

func logFromYaml(yaml string) (*log.LogEntity, error) {
	readerFactory := structure.NewReaderFactory(&structuredatastore.OnMemoryStructureDataStore{})
	reader, err := readerFactory.NewReader(adapter.Yaml(yaml))
	if err != nil {
		return nil, err
	}
	return log.NewLogEntity(reader, GCPCommonFieldExtractor{}), nil
}

func TestID(t *testing.T) {
	t.Run("generates unique ids", func(t *testing.T) {
		count := 10000
		ids := sync.Map{}
		for i := 0; i < count; i++ {
			randomInsertId := generateLogId()
			l, err := logFromYaml(fmt.Sprintf(`insertId: %s
timestamp: "2024-01-01T00:00:00Z"`, randomInsertId))
			if err != nil {
				t.Error(err)
			}
			id := l.ID()

			if _, found := ids.LoadOrStore(id, struct{}{}); found {
				t.Errorf("id %s is duplicated. ID is not unique enough", id)
			}
		}
	})

	t.Run("should generate the same Id for same insertId and timestamp", func(t *testing.T) {
		l1, err := logFromYaml(`insertId: foo
timestamp: bar`)
		if err != nil {
			t.Error(err)
		}
		l2, err := logFromYaml(`insertId: foo
timestamp: bar`)
		if err != nil {
			t.Error(err)
		}
		if l1.ID() != l2.ID() {
			t.Error("same id should be generated for same timestamp and insertId but didn't")
		}
	})
}

func TestMainMessage(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "parsing from textPayload",
			input:    `textPayload: foo bar`,
			expected: "foo bar",
		},
		{
			name: "parsing from MESSAGE",
			input: `jsonPayload:
  MESSAGE: foo bar
`,
			expected: "foo bar",
		},
		{
			name: "parsing from message",
			input: `jsonPayload:
  message: foo bar
`,
			expected: "foo bar",
		},
		{
			name: "parsing from msg",
			input: `jsonPayload:
  msg: foo bar
`,
			expected: "foo bar",
		},
		{
			name: "parsing from log",
			input: `jsonPayload:
  log: foo bar
`,
			expected: "foo bar",
		},
		{
			name: "parsing from httpRequest",
			input: `httpRequest:
  latency: 0.02s
  protocol: grpc
  remoteIp: 100.200.100.200:80
  requestMethod: POST
  requestSize: "1200"
  requestUrl: https://foo.bar/action
  responseSize: "1000"
  serverIp: 100.200.100.200:80
  status: 200
  userAgent: grpc-go/1.61.0
`,
			expected: "【200】GRPC https://foo.bar/action",
		},
		{
			name: "parsing from httpRequest",
			input: `httpRequest:
  latency: 0.02s
  protocol: http
  remoteIp: 100.200.100.200:80
  requestMethod: DELETE
  requestSize: "1200"
  requestUrl: https://foo.bar/action
  responseSize: "1000"
  serverIp: 100.200.100.200:80
  status: 200
  userAgent: grpc-go/1.61.0
`,
			expected: "【200】DELETE https://foo.bar/action",
		},
		{
			name: "fallback to json",
			input: `jsonPayload:
  foo: bar
  baz: qux
`,
			expected: "{\"foo\":\"bar\",\"baz\":\"qux\"}",
		},
		{
			name: "fallback to labels",
			input: `logName: foo
labels:
  foo: foo_val
  bar: bar_bal`,
			expected: `{"foo":"foo_val","bar":"bar_bal"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l, err := logFromYaml(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			msg, err := l.MainMessage()
			if err != nil {
				t.Fatal(err)
			}
			if msg != tc.expected {
				t.Errorf("expected %s,\n actual %s", tc.expected, msg)
			}
		})
	}
}
