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

func TestLogfmtTextParser(t *testing.T) {
	count := 10000
	result := make(chan *ParseStructuredLogResult, count)
	input := `msg="Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`
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
	parser := NewLogfmtTextParser()
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

func BenchmarkLogfmtTextParser(b *testing.B) {
	input := `msg="Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`
	parser := NewLogfmtTextParser()
	for i := 0; i < b.N; i++ {
		parser.TryParse(input)
	}
}

func TestLogfmtTextParserWorker_Parse(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *ParseStructuredLogResult
	}{
		{
			desc:  "simple logfmt message",
			input: `msg="Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415 severity=info`,
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
			desc:  "logfmt message with escaping in main message",
			input: `msg="Main \"message\"" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					MainMessageStructuredFieldKey: "Main \"message\"",
					"msg":                         "Main \"message\"",
					"fieldWithQuotes":             "foo",
					"fieldWithEscape":             `bar "qux"`,
					"fieldWithoutQuotes":          "3.1415",
				},
			},
		},
		{
			desc:  "logfmt message with no quotes around value",
			input: `msg=MainMessage fieldWithQuotes=foo fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					MainMessageStructuredFieldKey: "MainMessage",
					"msg":                         "MainMessage",
					"fieldWithQuotes":             "foo",
					"fieldWithEscape":             `bar "qux"`,
					"fieldWithoutQuotes":          "3.1415",
				},
			},
		},
		{
			desc:  "logfmt message with no quotes around value and escape non quoted value must be ignored",
			input: `msg=MainMessage fieldWithQuotes=f\oo fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					MainMessageStructuredFieldKey: "MainMessage",
					"msg":                         "MainMessage",
					"fieldWithQuotes":             "f\\oo",
					"fieldWithEscape":             `bar "qux"`,
					"fieldWithoutQuotes":          "3.1415",
				},
			},
		},
		{
			desc:  "logfmt message with no main message field",
			input: `fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					"fieldWithQuotes":    "foo",
					"fieldWithEscape":    `bar "qux"`,
					"fieldWithoutQuotes": "3.1415",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.want.Fields[OriginalMessageFieldKey] = tc.input
			worker := newLogfmtTextParserWorker()
			got, err := worker.parse(tc.input)
			if err != nil {
				t.Fatalf("parse() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
