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

package inspectioncore_impl

import (
	"testing"
	"time"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// TestTimeZoneShiftInputTask verifies that TimeZoneShiftInputTask parses the timezone offset
// parameter and returns the corresponding time.Location or falls back to UTC.
func TestTimeZoneShiftInputTask(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		name              string
		input             map[string]any
		wantZoneName      string
		wantOffsetSeconds int
		wantUTC           bool
	}{
		{
			name: "positive offset +9 hours",
			input: map[string]any{
				inspectioncore_contract.TaskInputKeyTimezoneShiftHours: float64(9),
			},
			wantZoneName:      "Unknown",
			wantOffsetSeconds: 9 * 3600,
			wantUTC:           false,
		},
		{
			name: "negative offset -7 hours",
			input: map[string]any{
				inspectioncore_contract.TaskInputKeyTimezoneShiftHours: float64(-7),
			},
			wantZoneName:      "Unknown",
			wantOffsetSeconds: -7 * 3600,
			wantUTC:           false,
		},
		{
			name: "zero offset falls back to UTC",
			input: map[string]any{
				inspectioncore_contract.TaskInputKeyTimezoneShiftHours: float64(0),
			},
			wantZoneName:      "UTC",
			wantOffsetSeconds: 0,
			wantUTC:           true,
		},
		{
			name:              "missing timezone key defaults to UTC",
			input:             map[string]any{},
			wantZoneName:      "UTC",
			wantOffsetSeconds: 0,
			wantUTC:           true,
		},
		{
			name: "non float64 value defaults to UTC",
			input: map[string]any{
				inspectioncore_contract.TaskInputKeyTimezoneShiftHours: "9",
			},
			wantZoneName:      "UTC",
			wantOffsetSeconds: 0,
			wantUTC:           true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.RunInspectionTask(ctx, TimeZoneShiftInputTask, inspectioncore_contract.TaskModeRun, tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantUTC && got != time.UTC {
				t.Errorf("got %v, want time.UTC", got)
			}
			gotZoneName, gotOffset := baseTime.In(got).Zone()
			if gotZoneName != tc.wantZoneName {
				t.Errorf("zone name = %q, want %q", gotZoneName, tc.wantZoneName)
			}
			if gotOffset != tc.wantOffsetSeconds {
				t.Errorf("offset = %d, want %d", gotOffset, tc.wantOffsetSeconds)
			}
		})
	}
}
