// Copyright 2025 Google LLC
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

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/sync/errgroup"
)

func TestJsonlTextParser(t *testing.T) {
	count := 10000
	result := make(chan *ParseStructuredLogResult, count)
	input := `{"msg":"Main message","fieldWithQuotes":"foo","fieldWithEscape":"bar \"qux\"","fieldWithoutQuotes":3.1415}`
	want := &ParseStructuredLogResult{
		Fields: map[string]any{
			MainMessageStructuredFieldKey: "Main message",
			OriginalMessageFieldKey:       input,
			"msg":                         "Main message",
			"fieldWithQuotes":             "foo",
			"fieldWithEscape":             `bar "qux"`,
			"fieldWithoutQuotes":          "3.1415",
		},
	}
	parser := NewJsonlTextParser()
	errgrp := errgroup.Group{}
	for i := 0; i < count; i++ {
		errgrp.Go(func() error {
			result <- parser.TryParse(input)
			return nil
		})
	}
	errgrp.Wait()
	close(result)
	for got := range result {
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("parse() mismatch (-want +got):\n%s", diff)
		}
	}
}

func BenchmarkJsonlTextParser(b *testing.B) {
	input := `{"msg":"Main message","fieldWithQuotes":"foo","fieldWithEscape":"bar \"qux\"","fieldWithoutQuotes":3.1415}`
	parser := NewJsonlTextParser()
	for i := 0; i < b.N; i++ {
		parser.TryParse(input)
	}
}

func TestJsonlTextParser_TryParse(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *ParseStructuredLogResult
	}{
		{
			desc:  "simple jsonl message",
			input: `{"msg":"Main message","fieldWithQuotes":"foo","fieldWithEscape":"bar \"qux\"","fieldWithoutQuotes":3.1415,"severity":"info"}`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					MainMessageStructuredFieldKey: "Main message",
					SeverityStructuredFieldKey:    enum.SeverityInfo,
					"msg":                         "Main message",
					"severity":                    "info",
					"fieldWithQuotes":             "foo",
					"fieldWithEscape":             `bar "qux"`,
					"fieldWithoutQuotes":          "3.1415",
				},
			},
		},
		{
			desc:  "jsonl message with leading whitespace",
			input: `  {"msg":"Main message","fieldWithQuotes":"foo","fieldWithEscape":"bar \"qux\"","fieldWithoutQuotes":3.1415,"severity":"info"}`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					MainMessageStructuredFieldKey: "Main message",
					SeverityStructuredFieldKey:    enum.SeverityInfo,
					"msg":                         "Main message",
					"severity":                    "info",
					"fieldWithQuotes":             "foo",
					"fieldWithEscape":             `bar "qux"`,
					"fieldWithoutQuotes":          "3.1415",
				},
			},
		},
		{
			desc:  "jsonl message with level",
			input: `{"message":"Main \"message\"","fieldWithQuotes":"foo","level":"WARN"}`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					MainMessageStructuredFieldKey: `Main "message"`,
					SeverityStructuredFieldKey:    enum.SeverityWarning,
					"message":                     `Main "message"`,
					"level":                       "WARN",
					"fieldWithQuotes":             "foo",
				},
			},
		},
		{
			desc:  "jsonl message with nested object",
			input: `{"msg":"MainMessage","nested":{"foo":"bar"}}`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					MainMessageStructuredFieldKey: "MainMessage",
					"msg":                         "MainMessage",
					"nested":                      `{"foo":"bar"}`,
				},
			},
		},
		{
			desc:  "jsonl message with boolean and null",
			input: `{"msg":"System boot","success":true,"error":null}`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					MainMessageStructuredFieldKey: "System boot",
					"msg":                         "System boot",
					"success":                     "true",
					"error":                       "null",
				},
			},
		},
		{
			desc:  "jsonl message with int64 and float",
			input: `{"msg":"System boot","largeInt":1234567890123456789,"floatValue":3.14159}`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					MainMessageStructuredFieldKey: "System boot",
					"msg":                         "System boot",
					"largeInt":                    "1234567890123456789",
					"floatValue":                  "3.14159",
				},
			},
		},
		{
			desc:  "not a jsonl log",
			input: `msg=MainMessage fieldWithQuotes=foo`,
			want:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.want != nil {
				tc.want.Fields[OriginalMessageFieldKey] = tc.input
			}
			parser := NewJsonlTextParser()
			got := parser.TryParse(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
