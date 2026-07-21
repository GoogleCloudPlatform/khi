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

	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestZapConsoleTextParser_TryParse(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  *ParseStructuredLogResult
	}{
		{
			name:  "standard zap console log with caller and json fields",
			input: "2026-02-18T06:58:06.999Z\tinfo\tkubernetes/kubernetes.go:282\tUsing pod service account via in-cluster config\t{\"kind\": \"receiver\", \"name\": \"prometheus\", \"discovery\": \"kubernetes\"}",
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					OriginalMessageFieldKey:       "2026-02-18T06:58:06.999Z\tinfo\tkubernetes/kubernetes.go:282\tUsing pod service account via in-cluster config\t{\"kind\": \"receiver\", \"name\": \"prometheus\", \"discovery\": \"kubernetes\"}",
					MainMessageStructuredFieldKey: "Using pod service account via in-cluster config",
					SeverityStructuredFieldKey:    inspectioncore_contract.SeverityInfo,
					ZapConsoleTimestampFieldKey:   "2026-02-18T06:58:06.999Z",
					ZapConsoleCallerFieldKey:      "kubernetes/kubernetes.go:282",
					"kind":                        "receiver",
					"name":                        "prometheus",
					"discovery":                   "kubernetes",
				},
			},
		},
		{
			name:  "zap console log with caller, empty message, and json fields",
			input: "2026-02-18T06:58:06.999Z\tinfo\tkubernetes/kubernetes.go:282\t\t{\"kind\": \"receiver\"}",
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					OriginalMessageFieldKey:     "2026-02-18T06:58:06.999Z\tinfo\tkubernetes/kubernetes.go:282\t\t{\"kind\": \"receiver\"}",
					SeverityStructuredFieldKey:  inspectioncore_contract.SeverityInfo,
					ZapConsoleTimestampFieldKey: "2026-02-18T06:58:06.999Z",
					ZapConsoleCallerFieldKey:    "kubernetes/kubernetes.go:282",
					"kind":                      "receiver",
				},
			},
		},
		{
			name:  "zap console log with caller and no json fields",
			input: "2026-02-18T06:58:06.999Z\tinfo\tkubernetes/kubernetes.go:282\tUsing pod service account via in-cluster config",
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					OriginalMessageFieldKey:       "2026-02-18T06:58:06.999Z\tinfo\tkubernetes/kubernetes.go:282\tUsing pod service account via in-cluster config",
					MainMessageStructuredFieldKey: "Using pod service account via in-cluster config",
					SeverityStructuredFieldKey:    inspectioncore_contract.SeverityInfo,
					ZapConsoleTimestampFieldKey:   "2026-02-18T06:58:06.999Z",
					ZapConsoleCallerFieldKey:      "kubernetes/kubernetes.go:282",
				},
			},
		},
		{
			name:  "zap console log without caller (rejected)",
			input: "2026-02-18T06:58:06.999Z\twarn\tUsing pod service account via in-cluster config\t{\"kind\": \"receiver\"}",
			want:  nil,
		},
		{
			name:  "zap console log without timestamp header (rejected)",
			input: "info\tkubernetes/kubernetes.go:282\tUsing pod service account via in-cluster config",
			want:  nil,
		},
		{
			name:  "plain text message without tabs (rejected)",
			input: "This is a simple plain text log message",
			want:  nil,
		},
		{
			name:  "tab separated text with invalid timestamp (rejected)",
			input: "invalid_time\tinfo\tkubernetes/kubernetes.go:282\tUsing pod service account",
			want:  nil,
		},
	}

	parser := NewZapConsoleTextParser()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parser.TryParse(tc.input)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("TryParse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
