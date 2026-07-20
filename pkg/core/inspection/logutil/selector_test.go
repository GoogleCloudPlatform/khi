// Copyright 2026 Google LLC
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

package logutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type testContext struct {
	Component string
}

func TestSelectorLogParser_TryParse(t *testing.T) {
	klogParser := NewKLogTextParser(true)
	jsonlParser := NewJsonlTextParser()
	fallbackParser := &FallbackRawTextLogParser{}

	selector := NewSelectorLogParser[testContext](
		fallbackParser,
		ParserRule[testContext]{
			Match:  func(ctx testContext) bool { return ctx.Component == "kubelet" },
			Parser: klogParser,
		},
		ParserRule[testContext]{
			Match:  func(ctx testContext) bool { return ctx.Component == "json-app" },
			Parser: jsonlParser,
		},
	)

	testCases := []struct {
		name    string
		ctx     testContext
		message string
		wantMsg string
	}{
		{
			name:    "matched rule for kubelet (klog)",
			ctx:     testContext{Component: "kubelet"},
			message: `I0929 08:20:24.205299    1949 kubelet.go:100] "Pod started" pod="ns/pod1"`,
			wantMsg: "Pod started",
		},
		{
			name:    "matched rule for json-app (jsonl)",
			ctx:     testContext{Component: "json-app"},
			message: `{"level":"info","msg":"json log message","app":"demo"}`,
			wantMsg: "json log message",
		},
		{
			name:    "fallback to default parser when no rule matches",
			ctx:     testContext{Component: "unknown-component"},
			message: `raw unstructured text log line`,
			wantMsg: "raw unstructured text log line",
		},
		{
			name:    "fallback to default parser when matching parser returns nil",
			ctx:     testContext{Component: "kubelet"},
			message: `not a valid klog message`,
			wantMsg: "not a valid klog message",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := selector.TryParse(tc.ctx, tc.message)
			if got == nil {
				t.Fatalf("TryParse() returned nil, expected result")
			}
			msg, err := got.MainMessage()
			if err != nil {
				t.Fatalf("MainMessage() failed: %v", err)
			}
			if diff := cmp.Diff(tc.wantMsg, msg); diff != "" {
				t.Errorf("MainMessage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
