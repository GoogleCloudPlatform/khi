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

package inspectiontaskbase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var mockLogIngesterPrevTaskID = taskid.NewDefaultImplementationID[[]*log.Log]("mock-log-ingester-v2-prev")

type mockLogIngester struct {
	cancel context.CancelFunc
}

func (m *mockLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return mockLogIngesterPrevTaskID.Ref()
}

func (m *mockLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (m *mockLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	shouldCancel := l.ReadBoolOrDefault("cancel", false)
	if shouldCancel && m.cancel != nil {
		m.cancel()
		time.Sleep(50 * time.Millisecond)
		return nil, ctx.Err()
	}
	shouldErr := l.ReadBoolOrDefault("error", false)
	if shouldErr {
		return nil, fmt.Errorf("test error")
	}
	shouldSkip := l.ReadBoolOrDefault("skip", false)
	if shouldSkip {
		return nil, nil // Return nil changeset (skip)
	}

	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetSummary("hello summary")
	cs.SetTimestamp(time.Now())

	severityID := uint32(1)
	logTypeID := uint32(2)
	cs.SetSeverity(&pb.Severity{Id: &severityID})
	cs.SetLogType(&pb.LogType{Id: &logTypeID})
	return cs, nil
}

var _ LogIngester = (*mockLogIngester)(nil)

func TestLogIngesterTaskV2(t *testing.T) {
	type testLog struct {
		yaml         string
		shouldIngest bool
	}

	testCases := []struct {
		desc          string
		taskMode      inspectioncore_contract.InspectionTaskModeType
		prevLogs      []testLog
		wantError     bool
		cancelContext bool
		cancelMidWay  bool
	}{
		{
			desc:     "DryRun mode",
			taskMode: inspectioncore_contract.TaskModeDryRun,
			prevLogs: []testLog{
				{
					yaml:         `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-1"}`,
					shouldIngest: false,
				},
			},
			wantError: false,
		},
		{
			desc:     "Normal execution with some skipped logs",
			taskMode: inspectioncore_contract.TaskModeRun,
			prevLogs: []testLog{
				{
					yaml:         `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-1"}`,
					shouldIngest: true,
				},
				{
					yaml:         `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-2", "skip": true}`,
					shouldIngest: false,
				},
			},
			wantError: false,
		},
		{
			desc:     "Execution with error",
			taskMode: inspectioncore_contract.TaskModeRun,
			prevLogs: []testLog{
				{
					yaml:         `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-1", "error": true}`,
					shouldIngest: false,
				},
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tid := taskid.NewDefaultImplementationID[[]*log.Log]("mock-log-ingester-v2")

			ctx := context.Background()
			ctx = inspectiontest.WithDefaultTestInspectionTaskContext(ctx)

			var cancel context.CancelFunc
			if tc.cancelContext || tc.cancelMidWay {
				ctx, cancel = context.WithCancel(ctx)
			}
			if tc.cancelContext {
				cancel()
			}

			builder := khifilev6.NewBuilder()
			ctx = khictx.WithValue(ctx, inspectioncore_contract.Builder, builder)

			var logs []*log.Log
			shouldIngestMap := make(map[string]bool)
			for _, tl := range tc.prevLogs {
				l := mustNewLogFromYAML(t, tl.yaml)
				logs = append(logs, l)
				shouldIngestMap[l.ID] = tl.shouldIngest
			}

			task := NewLogIngesterTask(tid, &mockLogIngester{})

			_, _, err := inspectiontest.RunInspectionTask(ctx, task, tc.taskMode, map[string]any{}, tasktest.NewTaskDependencyValuePair(mockLogIngesterPrevTaskID.Ref(), logs))
			if (err != nil) != tc.wantError {
				t.Fatalf("RunInspectionTask() error = %v, wantError %v", err, tc.wantError)
			}

			if tc.cancelContext {
				var expectedErr = context.Canceled
				if err == nil || err.Error() != expectedErr.Error() {
					t.Errorf("RunInspectionTask() error = %v, want %v", err, expectedErr)
				}
			}

			if !tc.wantError {
				// Verify whether logs were correctly accumulated in LogAccumulator or not
				for _, l := range logs {
					resolvedID, ok := builder.LogAccumulator.ResolveLogID(l.ID)
					shouldIngest := shouldIngestMap[l.ID]
					if shouldIngest {
						if !ok {
							t.Errorf("expected log %s to be ingested, but ResolveLogID returned false", l.ID)
						}
						if resolvedID == 0 {
							t.Errorf("expected log %s to have a valid resolved ID, got 0", l.ID)
						}
					} else if ok {
						t.Errorf("expected log %s NOT to be ingested, but ResolveLogID returned true with ID %d", l.ID, resolvedID)
					}
				}
			}
		})
	}
}
