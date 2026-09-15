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

package gcpcommon

import (
	"testing"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func TestParseGCPSeverity(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  *pb.Severity
	}{
		{
			name:  "DEFAULT returns Info",
			input: "DEFAULT",
			want:  inspectioncore.SeverityInfo,
		},
		{
			name:  "DEBUG returns Info",
			input: "DEBUG",
			want:  inspectioncore.SeverityInfo,
		},
		{
			name:  "INFO returns Info",
			input: "INFO",
			want:  inspectioncore.SeverityInfo,
		},
		{
			name:  "NOTICE returns Info",
			input: "NOTICE",
			want:  inspectioncore.SeverityInfo,
		},
		{
			name:  "WARNING returns Warning",
			input: "WARNING",
			want:  inspectioncore.SeverityWarning,
		},
		{
			name:  "ERROR returns Error",
			input: "ERROR",
			want:  inspectioncore.SeverityError,
		},
		{
			name:  "CRITICAL returns Fatal",
			input: "CRITICAL",
			want:  inspectioncore.SeverityFatal,
		},
		{
			name:  "ALERT returns Fatal",
			input: "ALERT",
			want:  inspectioncore.SeverityFatal,
		},
		{
			name:  "EMERGENCY returns Fatal",
			input: "EMERGENCY",
			want:  inspectioncore.SeverityFatal,
		},
		{
			name:  "UNKNOWN returns Unknown",
			input: "UNKNOWN",
			want:  inspectioncore.SeverityUnknown,
		},
		{
			name:  "invalid string returns Unknown",
			input: "INVALID_SEVERITY",
			want:  inspectioncore.SeverityUnknown,
		},
		{
			name:  "empty string returns Unknown",
			input: "",
			want:  inspectioncore.SeverityUnknown,
		},
		{
			name:  "case insensitive - lowercase info",
			input: "info",
			want:  inspectioncore.SeverityInfo,
		},
		{
			name:  "case insensitive - mixed case warning",
			input: "WaRnInG",
			want:  inspectioncore.SeverityWarning,
		},
		{
			name:  "surrounding spaces",
			input: "  INFO  ",
			want:  inspectioncore.SeverityInfo,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseGCPSeverity(tc.input)
			if got != tc.want {
				t.Errorf("ParseGCPSeverity(%q) = %v; want %v", tc.input, got, tc.want)
			}
		})
	}
}
