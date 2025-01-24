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

package k8s

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/log/schema"
	"github.com/google/go-cmp/cmp"
)

func TestParseKLogHeader(t *testing.T) {
	testCases := []struct {
		InputLog string
		Expected *klogHeader
	}{
		{
			InputLog: `I0930 00:01:02.500000    1992 prober.go:116] "Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`,
			Expected: &klogHeader{
				Severity: schema.SeverityInfo,
				Message:  `"Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`,
			},
		},
		{
			InputLog: `time="2024-01-09T23:18:46.683566491Z" level=info msg="foo bar" fieldWithQuotes="foo" fieldWithEscape="foo \"bar\"" fieldWithoutQuotes=qux1234`,
			Expected: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.InputLog, func(t *testing.T) {
			actual := parseKLogHeader(testCase.InputLog)
			if diff := cmp.Diff(testCase.Expected, actual); diff != "" {
				t.Errorf("Result is not matching with the expected outcome.\n%s", diff)
			}
		})
	}
}

func TestParseKLogMessageFragment(t *testing.T) {
	testCases := []struct {
		Name             string
		InputLogFragment string
		Expected         map[string]string
	}{
		{
			Name:             "parse with basic types and escape",
			InputLogFragment: `"message" quote="foo" escape="bar \"qux\"" nonQuote=3.1415`,
			Expected: map[string]string{
				"msg":      "message",
				"quote":    "foo",
				"escape":   `bar "qux"`,
				"nonQuote": "3.1415",
			},
		},
		{
			Name:             "parse message not beginning with the main message",
			InputLogFragment: `time="2024-01-09T23:18:46.683566491Z" level=info msg="foo bar" fieldWithQuotes="foo" fieldWithEscape="foo \"bar\"" fieldWithoutQuotes=qux1234`,
			Expected: map[string]string{
				"time":               "2024-01-09T23:18:46.683566491Z",
				"level":              "info",
				"msg":                "foo bar",
				"fieldWithEscape":    `foo "bar"`,
				"fieldWithQuotes":    "foo",
				"fieldWithoutQuotes": "qux1234",
			},
		},
		{
			Name:             "parse message containing Golang brace",
			InputLogFragment: `time="2024-01-01T09:36:50.190865579Z" level=info msg="CreateContainer within sandbox \"0967f41e43a74987c232c287ddf4eb5291cea903aed457eea197b6a0fc8f51a9\" for &ContainerMetadata{Name:config-reloader,Attempt:0,} returns container id \"31569ae45ccb2fe6d89eb298cd1fa124e6522c3c1683b944e4168ee19488430c\""`,
			Expected: map[string]string{
				"time":  "2024-01-01T09:36:50.190865579Z",
				"level": "info",
				"msg":   `CreateContainer within sandbox "0967f41e43a74987c232c287ddf4eb5291cea903aed457eea197b6a0fc8f51a9" for &ContainerMetadata{Name:config-reloader,Attempt:0,} returns container id "31569ae45ccb2fe6d89eb298cd1fa124e6522c3c1683b944e4168ee19488430c"`,
			},
		},
		{
			Name:             "parse message containing Golang struct directly in the field",
			InputLogFragment: `"SyncLoop (PLEG): event for pod" pod="kube-system/fluentbit-gke-bfkqc" event=&{ID:0043b37a-0001-48de-a6ed-60f8ea3151f2 Type:ContainerStarted Data:cbfd68440fe523435bdf9f68d0a0f45ab20af1f421dd8a060a10f4e106992c87}`,
			Expected: map[string]string{
				"msg":   "SyncLoop (PLEG): event for pod",
				"pod":   "kube-system/fluentbit-gke-bfkqc",
				"event": "&{ID:0043b37a-0001-48de-a6ed-60f8ea3151f2 Type:ContainerStarted Data:cbfd68440fe523435bdf9f68d0a0f45ab20af1f421dd8a060a10f4e106992c87}",
			},
		},
		{
			Name:             "parse message with complex escapes",
			InputLogFragment: `"SyncLoop (PLEG): event for pod" pod="kube-system/fluentbit-gke-bfkqc" event=&{ID:0043b37a-0001-48de-a6ed-60f8ea3151f2 Type:ContainerStarted Data:cbfd68440fe523435bdf9f68d0a0f45ab20af1f421dd8a060a10f4e106992c87}`,
			Expected: map[string]string{
				"msg":   "SyncLoop (PLEG): event for pod",
				"pod":   "kube-system/fluentbit-gke-bfkqc",
				"event": "&{ID:0043b37a-0001-48de-a6ed-60f8ea3151f2 Type:ContainerStarted Data:cbfd68440fe523435bdf9f68d0a0f45ab20af1f421dd8a060a10f4e106992c87}",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			actual := parseKLogMessageFragment(testCase.InputLogFragment)
			if diff := cmp.Diff(testCase.Expected, actual); diff != "" {
				t.Errorf("Result is not matching with the expected outcome.\n%s", diff)
			}
		})
	}
}

func TestExtractKLogField(t *testing.T) {
	type match struct {
		Field    string
		Expected string
	}
	testCases := []struct {
		InputLog string
		Matches  []match
	}{
		{
			InputLog: `I0930 00:01:02.030000    1992 prober.go:116] "Main message" fieldWithQuotes="foo" fieldWithEscape="bar \"qux\"" fieldWithoutQuotes=3.1415`,
			Matches: []match{
				{
					Field:    "",
					Expected: "Main message",
				},
				{
					Field:    "fieldWithQuotes",
					Expected: "foo",
				},
				{
					Field:    "fieldWithEscape",
					Expected: `bar "qux"`,
				},
				{
					Field:    "fieldWithoutQuotes",
					Expected: "3.1415",
				},
				{
					Field:    "non-existing-field",
					Expected: "",
				},
				{
					Field:    KLogSeverityFieldAlias,
					Expected: "INFO",
				},
			},
		},
		{
			InputLog: `I1125 05:07:10.533544    1679 flags.go:64] FLAG: --container-runtime-endpoint="unix:///run/containerd/containerd.sock"`,
			Matches: []match{
				{
					Field:    "",
					Expected: `FLAG: --container-runtime-endpoint="unix:///run/containerd/containerd.sock"`,
				}, {
					Field:    KLogSeverityFieldAlias,
					Expected: "INFO",
				},
			},
		},
		{
			InputLog: `Error foo" fieldWithQuotes="foo" fieldWithEscape="foo \"bar\"" fieldWithoutQuotes=qux1234`,
			Matches: []match{
				{
					Field:    "",
					Expected: `Error foo`,
				}, {
					Field:    "fieldWithQuotes",
					Expected: "foo",
				}, {
					Field:    "fieldWithEscape",
					Expected: `foo "bar"`,
				},
				{
					Field:    "fieldWithoutQuotes",
					Expected: "qux1234",
				},
			},
		},
		{
			InputLog: `time="2024-01-09T23:18:46.683566491Z" level=info msg="foo bar" fieldWithQuotes="foo" fieldWithEscape="foo \"bar\"" fieldWithoutQuotes=qux1234`,
			Matches: []match{
				{
					Field:    "",
					Expected: `foo bar`,
				}, {
					Field:    "fieldWithQuotes",
					Expected: "foo",
				}, {
					Field:    "fieldWithEscape",
					Expected: `foo "bar"`,
				},
				{
					Field:    "fieldWithoutQuotes",
					Expected: "qux1234",
				},
				{
					Field:    KLogSeverityFieldAlias,
					Expected: "INFO",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.InputLog, func(t *testing.T) {
			for _, match := range testCase.Matches {
				t.Run(match.Field, func(t *testing.T) {
					actual, err := ExtractKLogField(testCase.InputLog, match.Field)
					if err != nil {
						t.Errorf("unexpected error\n%v", err)
					}
					if diff := cmp.Diff(match.Expected, actual); diff != "" {
						t.Errorf("Result is not matching with the expected outcome.\n%s", diff)
					}
				})
			}
		})
	}
}
