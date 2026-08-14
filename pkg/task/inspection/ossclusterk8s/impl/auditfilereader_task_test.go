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

package ossclusterk8s_impl

import (
	"io"
	"strings"
	"testing"
	"time"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	ossclusterk8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/ossclusterk8s/contract"
	"github.com/google/go-cmp/cmp"
)

type mockUploadStoreProvider struct {
	data string
}

func (m *mockUploadStoreProvider) GetUploadToken(id string) upload.UploadToken {
	return &upload.DirectUploadToken{ID: id}
}

func (m *mockUploadStoreProvider) Read(token upload.UploadToken) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(m.data)), nil
}

func TestAuditLogFileReaderTask(t *testing.T) {
	testCases := []struct {
		name         string
		taskMode     inspectioncore_contract.InspectionTaskModeType
		inputData    string
		wantLogCount int
		wantStages   []string
		wantTimes    []time.Time
	}{
		{
			name:     "dry run mode returns empty list",
			taskMode: inspectioncore_contract.TaskModeDryRun,
			inputData: `{"stage":"ResponseComplete","stageTimestamp":"2026-05-25T12:00:00.000000Z","verb":"get"}
`,
			wantLogCount: 0,
		},
		{
			name:     "filters non-ResponseComplete stage and sorts by timestamp",
			taskMode: inspectioncore_contract.TaskModeRun,
			inputData: `{"stage":"ResponseStarted","stageTimestamp":"2026-05-25T12:00:01.000000Z","verb":"get"}
{"stage":"ResponseComplete","stageTimestamp":"2026-05-25T12:00:05.000000Z","verb":"create"}

{"stage":"ResponseComplete","stageTimestamp":"2026-05-25T12:00:02.000000Z","verb":"update"}
`,
			wantLogCount: 2,
			wantStages:   []string{"ResponseComplete", "ResponseComplete"},
			wantTimes: []time.Time{
				time.Date(2026, 5, 25, 12, 0, 2, 0, time.UTC),
				time.Date(2026, 5, 25, 12, 0, 5, 0, time.UTC),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			baseCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())

			gotLogs, _, err := inspectiontest.RunInspectionTask(
				baseCtx,
				AuditLogFileReaderTask,
				tc.taskMode,
				nil,
				tasktest.NewTaskDependencyValuePair(
					ossclusterk8s_contract.InputAuditLogFilesFormTaskID.Ref(),
					upload.UploadResult{
						Status:        upload.UploadStatusCompleted,
						StoreProvider: &mockUploadStoreProvider{data: tc.inputData},
						Token:         &upload.DirectUploadToken{ID: "token-1"},
					},
				),
			)
			if err != nil {
				t.Fatalf("AuditLogFileReaderTask returned unexpected error: %v", err)
			}

			if len(gotLogs) != tc.wantLogCount {
				t.Fatalf("AuditLogFileReaderTask log count mismatch (-want +got):\n%s", cmp.Diff(tc.wantLogCount, len(gotLogs)))
			}

			var gotStages []string
			var gotTimes []time.Time
			for _, l := range gotLogs {
				gotStages = append(gotStages, l.ReadStringOrDefault("stage", ""))
				commonField := log.MustGetFieldSet(l, &log.CommonFieldSet{})
				gotTimes = append(gotTimes, commonField.Timestamp)
			}

			if diff := cmp.Diff(tc.wantStages, gotStages); diff != "" {
				t.Errorf("AuditLogFileReaderTask stages mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantTimes, gotTimes); diff != "" {
				t.Errorf("AuditLogFileReaderTask timestamps mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
