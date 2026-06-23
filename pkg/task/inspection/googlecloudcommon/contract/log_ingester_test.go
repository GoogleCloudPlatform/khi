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

package googlecloudcommon_contract

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestGCPOperationLogIngester_ProcessLog(t *testing.T) {
	testTime := time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC)
	dummyTaskRef := taskid.NewTaskReference[[]*log.Log]("dummy")
	dummyLogType := style.MustRegisterLogType("dummy", "Dummy", style.MustForceConvertSRGBHex("#123456"), style.ColorWhite)

	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name: "operation starting log",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				&GCPAuditLogFieldSet{
					MethodName:     "compute.instances.insert",
					OperationFirst: true,
					OperationLast:  false,
					Status:         -1,
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("Start: compute.instances.insert").
					HasLogType(dummyLogType).
					HasTimestamp(testTime).
					HasSeverity(inspectioncore_contract.SeverityInfo)
			},
		},
		{
			name: "operation ending succeeded log",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				&GCPAuditLogFieldSet{
					MethodName:     "compute.instances.insert",
					OperationFirst: false,
					OperationLast:  true,
					Status:         0,
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("Succeeded: compute.instances.insert").
					HasLogType(dummyLogType).
					HasTimestamp(testTime)
			},
		},
		{
			name: "operation ending failed log",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityError,
				},
				&GCPAuditLogFieldSet{
					MethodName:     "compute.instances.insert",
					OperationFirst: false,
					OperationLast:  true,
					Status:         13,
					StatusMessage:  "Internal error",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("Failed: [13: Internal error] compute.instances.insert").
					HasLogType(dummyLogType).
					HasTimestamp(testTime)
			},
		},
		{
			name: "immediate operation succeeded log",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				&GCPAuditLogFieldSet{
					MethodName:     "compute.instances.get",
					OperationFirst: true,
					OperationLast:  true,
					Status:         0,
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("Succeeded: compute.instances.get").
					HasLogType(dummyLogType).
					HasTimestamp(testTime)
			},
		},
		{
			name: "immediate operation failed log",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityError,
				},
				&GCPAuditLogFieldSet{
					MethodName:     "compute.instances.delete",
					OperationFirst: true,
					OperationLast:  true,
					Status:         7,
					StatusMessage:  "Permission denied",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("Failed: [7: Permission denied] compute.instances.delete").
					HasLogType(dummyLogType).
					HasTimestamp(testTime)
			},
		},
		{
			name: "default log (neither starting nor ending)",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				&GCPAuditLogFieldSet{
					MethodName:     "compute.instances.list",
					OperationFirst: false,
					OperationLast:  false,
					Status:         -1,
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("compute.instances.list").
					HasLogType(dummyLogType).
					HasTimestamp(testTime)
			},
		},
	}

	ingester := NewGCPOperationLogIngester(dummyTaskRef, dummyLogType)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := ingester.ProcessLog(t.Context(), tc.input)
			if err != nil {
				t.Fatalf("ProcessLog() returned unexpected error: %v", err)
			}
			tc.assert(t, cs)
		})
	}
}
