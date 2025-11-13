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

func TestKLogTextParser(t *testing.T) {
	count := 10000
	result := make(chan *ParseStructuredLogResult, count)
	input := `I0930 00:01:02.500000    1992 prober.go:116] "Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`
	want := &ParseStructuredLogResult{
		Fields: map[string]any{
			SeverityStructuredFieldKey:       enum.SeverityInfo,
			OriginalMessageFieldKey:          input,
			KLogHeaderDateFieldKey:           "0930",
			KLogHeaderTimeFieldKey:           "00:01:02.500000",
			KLogHeaderThreadIDFieldKey:       "1992",
			KLogHeaderSourceLocationFieldKey: "prober.go:116",
			MainMessageStructuredFieldKey:    "Main message",
			"fieldWithQuotes":                "foo",
			"fieldWithEscape":                `bar "qux"`,
			"fieldWithoutQuotes":             "3.1415",
		},
	}
	parser := NewKLogTextParser(true)
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

func BenchmarkKLogTextParser(b *testing.B) {
	input := `I0930 00:01:02.500000    1992 prober.go:116] "Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`
	parser := NewKLogTextParser(false)
	for i := 0; i < b.N; i++ {
		parser.TryParse(input)
	}
}

func TestKlogTextParserWorker_Parse(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *ParseStructuredLogResult
	}{
		{
			desc:  "simple klog message",
			input: `I0930 00:01:02.500000    1992 prober.go:116] "Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					SeverityStructuredFieldKey:       enum.SeverityInfo,
					KLogHeaderDateFieldKey:           "0930",
					KLogHeaderTimeFieldKey:           "00:01:02.500000",
					KLogHeaderThreadIDFieldKey:       "1992",
					KLogHeaderSourceLocationFieldKey: "prober.go:116",
					MainMessageStructuredFieldKey:    "Main message",
					"fieldWithQuotes":                "foo",
					"fieldWithEscape":                `bar "qux"`,
					"fieldWithoutQuotes":             "3.1415",
				},
			},
		},
		{
			desc:  "klog message with escaping",
			input: `I0930 00:01:02.500000    1992 prober.go:116] "Main \"message\"" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					SeverityStructuredFieldKey:       enum.SeverityInfo,
					KLogHeaderDateFieldKey:           "0930",
					KLogHeaderTimeFieldKey:           "00:01:02.500000",
					KLogHeaderThreadIDFieldKey:       "1992",
					KLogHeaderSourceLocationFieldKey: "prober.go:116",
					MainMessageStructuredFieldKey:    "Main \"message\"",
					"fieldWithQuotes":                "foo",
					"fieldWithEscape":                `bar "qux"`,
					"fieldWithoutQuotes":             "3.1415",
				},
			},
		},
		{
			desc:  "klog message with golang struct pointer in field",
			input: `I0930 00:01:02.500000    1992 prober.go:116] "SyncLoop (PLEG): event for pod" pod="kube-system/fluentbit-gke-bfkqc" event=&{ID:0043b37a-0001-48de-a6ed-60f8ea3151f2 Type:ContainerStarted Data:cbfd68440fe523435bdf9f68d0a0f45ab20af1f421dd8a060a10f4e106992c87}`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					SeverityStructuredFieldKey:       enum.SeverityInfo,
					KLogHeaderDateFieldKey:           "0930",
					KLogHeaderTimeFieldKey:           "00:01:02.500000",
					KLogHeaderThreadIDFieldKey:       "1992",
					KLogHeaderSourceLocationFieldKey: "prober.go:116",
					MainMessageStructuredFieldKey:    "SyncLoop (PLEG): event for pod",
					"pod":                            "kube-system/fluentbit-gke-bfkqc",
					"event":                          "&{ID:0043b37a-0001-48de-a6ed-60f8ea3151f2 Type:ContainerStarted Data:cbfd68440fe523435bdf9f68d0a0f45ab20af1f421dd8a060a10f4e106992c87}",
				},
			},
		},
		{
			desc:  "klog message with golang struct in field",
			input: `I0930 00:01:02.500000    1992 prober.go:116] "SyncLoop (PLEG): event for pod" pod="kube-system/fluentbit-gke-bfkqc" event={ID:0043b37a-0001-48de-a6ed-60f8ea3151f2 Type:ContainerStarted Data:cbfd68440fe523435bdf9f68d0a0f45ab20af1f421dd8a060a10f4e106992c87}`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					SeverityStructuredFieldKey:       enum.SeverityInfo,
					KLogHeaderDateFieldKey:           "0930",
					KLogHeaderTimeFieldKey:           "00:01:02.500000",
					KLogHeaderThreadIDFieldKey:       "1992",
					KLogHeaderSourceLocationFieldKey: "prober.go:116",
					MainMessageStructuredFieldKey:    "SyncLoop (PLEG): event for pod",
					"pod":                            "kube-system/fluentbit-gke-bfkqc",
					"event":                          "{ID:0043b37a-0001-48de-a6ed-60f8ea3151f2 Type:ContainerStarted Data:cbfd68440fe523435bdf9f68d0a0f45ab20af1f421dd8a060a10f4e106992c87}",
				},
			},
		},
		{
			desc:  "klog message with array in field",
			input: `I0929 08:30:44.541804    1949 kubelet.go:2458] "SyncLoop DELETE" source="api" pods=["foo/bar","baz/qux"]`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					SeverityStructuredFieldKey:       enum.SeverityInfo,
					KLogHeaderDateFieldKey:           "0929",
					KLogHeaderTimeFieldKey:           "08:30:44.541804",
					KLogHeaderThreadIDFieldKey:       "1949",
					KLogHeaderSourceLocationFieldKey: "kubelet.go:2458",
					MainMessageStructuredFieldKey:    "SyncLoop DELETE",
					"source":                         "api",
					"pods":                           `["foo/bar","baz/qux"]`,
				},
			},
		},
		{
			desc:  "klog message with no main message and fields",
			input: `I0929 08:30:44.541804    1949 kubelet.go:2458] Some plain text message`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					SeverityStructuredFieldKey:       enum.SeverityInfo,
					KLogHeaderDateFieldKey:           "0929",
					KLogHeaderTimeFieldKey:           "08:30:44.541804",
					KLogHeaderThreadIDFieldKey:       "1949",
					KLogHeaderSourceLocationFieldKey: "kubelet.go:2458",
					MainMessageStructuredFieldKey:    "Some plain text message",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.want.Fields[OriginalMessageFieldKey] = tc.input
			worker := newKLogTextParserWorker(true)
			got := worker.parse(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestKlogTextParserWorker_Parse_WithoutHeader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *ParseStructuredLogResult
	}{
		{
			desc:  "simple klog message",
			input: `"Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					MainMessageStructuredFieldKey: "Main message",
					"fieldWithQuotes":             "foo",
					"fieldWithEscape":             `bar "qux"`,
					"fieldWithoutQuotes":          "3.1415",
				},
			},
		},
		{
			desc:  "klog message with no main message and fields",
			input: `Some plain text message`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					MainMessageStructuredFieldKey: "Some plain text message",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.want.Fields[OriginalMessageFieldKey] = tc.input
			worker := newKLogTextParserWorker(false)
			got := worker.parse(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
