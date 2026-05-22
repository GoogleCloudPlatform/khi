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
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

var mockGroupedRawTaskID = taskid.NewDefaultImplementationID[[]*log.Log]("mock-grouped-raw")
var mockGroupedLogTaskID = taskid.NewDefaultImplementationID[LogGroupMap]("mock-grouped-log")

type testState struct {
	Prefix string
}

type mockGroupedLogIngester struct {
	SinglePassGroupedIngesterBase[testState]
	rawTask   taskid.TaskReference[[]*log.Log]
	groupTask taskid.TaskReference[LogGroupMap]
	processFn func(ctx context.Context, l *log.Log, state testState) (*khifilev6.LogChangeSet, testState, error)
}

func (m *mockGroupedLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return m.rawTask
}

func (m *mockGroupedLogIngester) GroupedLogTask() taskid.TaskReference[LogGroupMap] {
	return m.groupTask
}

func (m *mockGroupedLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return nil
}

func (m *mockGroupedLogIngester) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData testState) (*khifilev6.LogChangeSet, testState, error) {
	if m.processFn != nil {
		return m.processFn(ctx, l, prevGroupData)
	}
	return nil, prevGroupData, nil
}

var _ GroupedLogIngesterV2[testState] = (*mockGroupedLogIngester)(nil)

func TestNewGroupedLogIngesterTaskV2(t *testing.T) {
	testCases := []struct {
		name       string
		rawLogs    []*log.Log
		groups     LogGroupMap
		setup      func(t *testing.T, rawLogs []*log.Log) (func(ctx context.Context, l *log.Log, state testState) (*khifilev6.LogChangeSet, testState, error), func(t *testing.T))
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "successfully processes logs with state",
			rawLogs: []*log.Log{
				mustNewLogFromYAML(t, `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-1"}`),
				mustNewLogFromYAML(t, `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-2"}`),
			},
			groups: LogGroupMap{
				"group1": {
					Group: "group1",
					Logs: []*log.Log{
						mustNewLogFromYAML(t, `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-1"}`),
						mustNewLogFromYAML(t, `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-2"}`),
					},
				},
			},
			setup: func(t *testing.T, rawLogs []*log.Log) (func(ctx context.Context, l *log.Log, state testState) (*khifilev6.LogChangeSet, testState, error), func(t *testing.T)) {
				processedStates := make(map[string]string)
				processFn := func(ctx context.Context, l *log.Log, state testState) (*khifilev6.LogChangeSet, testState, error) {
					newState := testState{Prefix: state.Prefix + "a"}
					processedStates[l.ID] = newState.Prefix
					cs, err := khifilev6.NewLogChangeSet(l)
					if err != nil {
						return nil, state, err
					}
					cs.SetSummary(newState.Prefix)
					severityID := uint32(1)
					logTypeID := uint32(2)
					cs.SetSeverity(&pb.Severity{Id: &severityID})
					cs.SetLogType(&pb.LogType{Id: &logTypeID})
					return cs, newState, nil
				}
				verifyFn := func(t *testing.T) {
					if diff := cmp.Diff("a", processedStates[rawLogs[0].ID]); diff != "" {
						t.Errorf("rawLogs[0] state mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff("aa", processedStates[rawLogs[1].ID]); diff != "" {
						t.Errorf("rawLogs[1] state mismatch (-want +got):\n%s", diff)
					}
				}
				return processFn, verifyFn
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, group := range tc.groups {
				for i := range group.Logs {
					if i < len(tc.rawLogs) {
						group.Logs[i] = tc.rawLogs[i]
					}
				}
			}

			var processFn func(ctx context.Context, l *log.Log, state testState) (*khifilev6.LogChangeSet, testState, error)
			var verifyFn func(t *testing.T)
			if tc.setup != nil {
				processFn, verifyFn = tc.setup(t, tc.rawLogs)
			}

			ingester := &mockGroupedLogIngester{
				rawTask:   mockGroupedRawTaskID.Ref(),
				groupTask: mockGroupedLogTaskID.Ref(),
				processFn: processFn,
			}

			tid := taskid.NewDefaultImplementationID[[]*log.Log]("test-grouped-ingester")
			task := NewGroupedLogIngesterTaskV2(tid, ingester)

			ctx := context.Background()
			ctx = inspectiontest.WithDefaultTestInspectionTaskContext(ctx)
			builder := khifilev6.NewBuilder()
			ctx = khictx.WithValue(ctx, inspectioncore_contract.Builder, builder)

			result, _, err := inspectiontest.RunInspectionTask(
				ctx,
				task,
				inspectioncore_contract.TaskModeRun,
				map[string]any{},
				tasktest.NewTaskDependencyValuePair(mockGroupedRawTaskID.Ref(), tc.rawLogs),
				tasktest.NewTaskDependencyValuePair(mockGroupedLogTaskID.Ref(), tc.groups),
			)

			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error state: got error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				if err.Error() != tc.wantErrMsg {
					t.Fatalf("unexpected error message: got %q, want %q", err.Error(), tc.wantErrMsg)
				}
				return
			}

			if len(result) != len(tc.rawLogs) {
				t.Fatalf("unexpected result count: got %d, want %d", len(result), len(tc.rawLogs))
			}

			// Verify ingestion.
			for _, l := range tc.rawLogs {
				id, ok := builder.LogAccumulator.ResolveLogID(l.ID)
				if !ok {
					t.Errorf("expected log %s to be resolved", l.ID)
				}
				if id == 0 {
					t.Errorf("expected log %s to have valid ID", l.ID)
				}
			}

			if verifyFn != nil {
				verifyFn(t)
			}
		})
	}
}

func TestNewGroupedLogIngesterTaskV2_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	ctx = inspectiontest.WithDefaultTestInspectionTaskContext(ctx)
	rawLogs := []*log.Log{
		mustNewLogFromYAML(t, `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-1"}`),
	}
	groups := LogGroupMap{
		"group1": {
			Group: "group1",
			Logs:  rawLogs,
		},
	}
	expectedErr := errors.New("mock process error")

	ingester := &mockGroupedLogIngester{
		rawTask:   mockGroupedRawTaskID.Ref(),
		groupTask: mockGroupedLogTaskID.Ref(),
		processFn: func(ctx context.Context, l *log.Log, state testState) (*khifilev6.LogChangeSet, testState, error) {
			return nil, state, expectedErr
		},
	}

	tid := taskid.NewDefaultImplementationID[[]*log.Log]("test-grouped-ingester-error")
	task := NewGroupedLogIngesterTaskV2(tid, ingester)

	builder := khifilev6.NewBuilder()
	ctx = khictx.WithValue(ctx, inspectioncore_contract.Builder, builder)

	_, _, err := inspectiontest.RunInspectionTask(
		ctx,
		task,
		inspectioncore_contract.TaskModeRun,
		map[string]any{},
		tasktest.NewTaskDependencyValuePair(mockGroupedRawTaskID.Ref(), rawLogs),
		tasktest.NewTaskDependencyValuePair(mockGroupedLogTaskID.Ref(), groups),
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("unexpected error: got %v, want %v", err, expectedErr)
	}
}
