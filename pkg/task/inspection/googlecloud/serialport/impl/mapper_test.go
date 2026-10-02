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

package serialport_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/serialport"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestProcessSerialPortLog(t *testing.T) {
	testTime := time.Date(2025, 9, 29, 6, 39, 24, 0, time.UTC)
	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name: "successful log ingestion",
			input: testlog.NewMockLog(
				testTime,
				inspectioncore.DefaultSeverityFieldSet{
					Severity: inspectioncore.SeverityError,
				},
				serialport.GCESerialPortLogFieldSet{
					Message:  "foo payload",
					NodeName: "node-name-bar",
					Port:     "serial_port_output_qux",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("foo payload").
					HasTimestamp(testTime).
					HasSeverity(inspectioncore.SeverityError).
					HasLogType(serialport.LogTypeSerialPort)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			cs, err := processSerialPortLog(ctx, tc.input)
			if err != nil {
				t.Fatalf("processSerialPortLog() returned unexpected error: %v", err)
			}
			tc.assert(t, cs)
		})
	}
}

func TestSerialPortLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())
	ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
	wantSerialPortPath := serialport.MustSerialPortTimeline(ctx, "test-cluster", "node-name-bar", "serial_port_output_qux")

	testCases := []struct {
		name     string
		inputLog *log.Log
		assert   func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "create timeline event",
			inputLog: testlog.NewMockLog(
				serialport.GCESerialPortLogFieldSet{
					Message:  "foo payload",
					NodeName: "node-name-bar",
					Port:     "serial_port_output_qux",
				},
			),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantSerialPortPath)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
			cs, err := mapSerialPortLog(ctx, tc.inputLog, "test-cluster")
			if err != nil {
				t.Fatalf("mapSerialPortLog() returned unexpected error: %v", err)
			}

			tc.assert(t, ctx, cs)
		})
	}
}
