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
	"time"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestKLogTextParser(t *testing.T) {
	count := 10000
	result := make(chan *ParseStructuredLogResult, count)
	input := `I0930 00:01:02.500000    1992 prober.go:116] "Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`
	want := &ParseStructuredLogResult{
		Fields: map[string]any{
			SeverityStructuredFieldKey:       inspectioncore.SeverityInfo,
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
	parser := NewKLogTextParser()
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
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("parse() mismatch (-want +got):\n%s", diff)
		}
	}
}

func BenchmarkKLogTextParser(b *testing.B) {
	input := `I0930 00:01:02.500000    1992 prober.go:116] "Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`
	parser := NewKLogTextParser()
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
					SeverityStructuredFieldKey:       inspectioncore.SeverityInfo,
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
					SeverityStructuredFieldKey:       inspectioncore.SeverityInfo,
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
					SeverityStructuredFieldKey:       inspectioncore.SeverityInfo,
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
					SeverityStructuredFieldKey:       inspectioncore.SeverityInfo,
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
					SeverityStructuredFieldKey:       inspectioncore.SeverityInfo,
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
					SeverityStructuredFieldKey:       inspectioncore.SeverityInfo,
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
			worker := newKLogTextParserWorker()
			got := worker.parse(tc.input)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatKLogString(t *testing.T) {
	baseTime := time.Date(2025, 9, 29, 8, 20, 24, 205299000, time.UTC)
	testCases := []struct {
		desc      string
		severity  *pb.Severity
		timestamp time.Time
		threadID  int
		file      string
		line      int
		message   string
		want      string
	}{
		{
			desc:      "info severity with structured message",
			severity:  inspectioncore.SeverityInfo,
			timestamp: baseTime,
			file:      "scheduler.go",
			line:      100,
			message:   `"Starting watch" path="/apis"`,
			want:      `I0929 08:20:24.205299 0 scheduler.go:100] "Starting watch" path="/apis"`,
		},
		{
			desc:      "warning severity",
			severity:  inspectioncore.SeverityWarning,
			timestamp: baseTime,
			file:      "scheduler.go",
			line:      100,
			message:   "msg",
			want:      `W0929 08:20:24.205299 0 scheduler.go:100] msg`,
		},
		{
			desc:      "error severity",
			severity:  inspectioncore.SeverityError,
			timestamp: baseTime,
			file:      "scheduler.go",
			line:      100,
			message:   "msg",
			want:      `E0929 08:20:24.205299 0 scheduler.go:100] msg`,
		},
		{
			desc:      "fatal severity",
			severity:  inspectioncore.SeverityFatal,
			timestamp: baseTime,
			file:      "scheduler.go",
			line:      100,
			message:   "msg",
			want:      `F0929 08:20:24.205299 0 scheduler.go:100] msg`,
		},
		{
			desc:      "unknown severity falls back to info",
			severity:  inspectioncore.SeverityUnknown,
			timestamp: baseTime,
			file:      "scheduler.go",
			line:      100,
			message:   "msg",
			want:      `I0929 08:20:24.205299 0 scheduler.go:100] msg`,
		},
		{
			desc:      "nil severity falls back to info",
			severity:  nil,
			timestamp: baseTime,
			file:      "scheduler.go",
			line:      100,
			message:   "msg",
			want:      `I0929 08:20:24.205299 0 scheduler.go:100] msg`,
		},
		{
			desc:      "non-UTC timestamp is converted to UTC",
			severity:  inspectioncore.SeverityInfo,
			timestamp: time.Date(2025, 9, 30, 3, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
			file:      "scheduler.go",
			line:      100,
			message:   "msg",
			want:      `I0929 18:00:00.000000 0 scheduler.go:100] msg`,
		},
		{
			desc:      "empty file is replaced with unknown",
			severity:  inspectioncore.SeverityInfo,
			timestamp: baseTime,
			file:      "",
			line:      0,
			message:   "msg",
			want:      `I0929 08:20:24.205299 0 unknown:0] msg`,
		},
		{
			desc:      "thread ID is written as given",
			severity:  inspectioncore.SeverityInfo,
			timestamp: baseTime,
			threadID:  1949,
			file:      "scheduler.go",
			line:      100,
			message:   "msg",
			want:      `I0929 08:20:24.205299 1949 scheduler.go:100] msg`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := FormatKLogString(tc.severity, tc.timestamp, tc.threadID, tc.file, tc.line, tc.message)
			if got != tc.want {
				t.Errorf("FormatKLogString() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatKLogString_ParsableByKLogTextParser(t *testing.T) {
	line := FormatKLogString(inspectioncore.SeverityWarning, time.Date(2025, 9, 29, 8, 20, 24, 205299000, time.UTC), 0, "", 0, `"Starting watch" path="/apis" timeout="9m16s"`)
	want := &ParseStructuredLogResult{
		Fields: map[string]any{
			OriginalMessageFieldKey:          line,
			SeverityStructuredFieldKey:       inspectioncore.SeverityWarning,
			KLogHeaderDateFieldKey:           "0929",
			KLogHeaderTimeFieldKey:           "08:20:24.205299",
			KLogHeaderThreadIDFieldKey:       "0",
			KLogHeaderSourceLocationFieldKey: "unknown:0",
			MainMessageStructuredFieldKey:    "Starting watch",
			"path":                           "/apis",
			"timeout":                        "9m16s",
		},
	}

	got := NewKLogTextParser().TryParse(line)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("TryParse(FormatKLogString(...)) mismatch (-want +got):\n%s", diff)
	}
}
