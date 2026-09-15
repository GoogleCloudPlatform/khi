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

package coreinspection

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestInspectRunTaskGraph(t *testing.T) {
	logger.InitGlobalKHILogger()

	testCases := []struct {
		name           string
		queryUnknownID bool
		runInspection  bool
		failTask       bool
		wantErrIs      error
		wantPhase      apiv1.TaskRunPhase
	}{
		{
			name:           "reports a not found error for an unregistered inspection",
			queryUnknownID: true,
			wantErrIs:      ErrInspectionNotFound,
		},
		{
			name:      "reports an error for an inspection that has not started",
			wantErrIs: ErrRunTaskGraphNotStarted,
		},
		{
			name:          "reports the done phase after a successful run",
			runInspection: true,
			wantPhase:     apiv1.TaskRunPhase_TASK_RUN_PHASE_DONE,
		},
		{
			name:          "reports the error phase after a failed run",
			runInspection: true,
			failTask:      true,
			wantPhase:     apiv1.TaskRunPhase_TASK_RUN_PHASE_ERROR,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var taskErr error
			if tc.failTask {
				taskErr = fmt.Errorf("simulated failure")
			}
			server, inspectionID, taskImplID := newTestInspectionServer(t, taskErr)

			if tc.runInspection {
				runner := server.GetInspection(inspectionID)
				req := &inspectioncore.InspectionRequest{Values: map[string]any{}}
				if err := runner.Run(context.Background(), req); err != nil {
					t.Fatalf("Run failed: %v", err)
				}
				<-runner.Wait()
			}
			if tc.queryUnknownID {
				inspectionID = "unknown-inspection-id"
			}

			snapshot, err := InspectRunTaskGraph(server, inspectionID)
			if tc.wantErrIs != nil {
				if err == nil {
					t.Fatalf("InspectRunTaskGraph() = %v, want an error wrapping %v", snapshot, tc.wantErrIs)
				}
				if !errors.Is(err, tc.wantErrIs) {
					t.Errorf("InspectRunTaskGraph() error = %v, want it to wrap %v", err, tc.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("InspectRunTaskGraph() failed: %v", err)
			}

			if !snapshot.GetIsRunFinished() {
				t.Errorf("IsRunFinished = false, want true")
			}
			if snapshot.GetSnapshotTimeUnixNano() == 0 {
				t.Errorf("SnapshotTimeUnixNano = 0, want a non-zero capture time")
			}
			if !snapshot.GetDag().GetIsSuccess() {
				t.Errorf("Dag.IsSuccess = false, want true. error_message=%s", snapshot.GetDag().GetErrorMessage())
			}

			nodeStatus := findNodeStatus(snapshot.GetNodeStatuses(), taskImplID)
			if nodeStatus == nil {
				t.Fatalf("NodeStatuses has no entry for %s", taskImplID)
			}
			if nodeStatus.GetPhase() != tc.wantPhase {
				t.Errorf("NodeStatuses[%s].Phase = %v, want %v", taskImplID, nodeStatus.GetPhase(), tc.wantPhase)
			}
			if nodeStatus.GetStartTimeUnixNano() == 0 {
				t.Errorf("NodeStatuses[%s].StartTimeUnixNano = 0, want a non-zero start time", taskImplID)
			}
			if nodeStatus.GetEndTimeUnixNano() == 0 {
				t.Errorf("NodeStatuses[%s].EndTimeUnixNano = 0, want a non-zero end time", taskImplID)
			}
		})
	}
}

func TestInspectRunTaskGraphSortsNodeStatusesByTaskImplementationID(t *testing.T) {
	logger.InitGlobalKHILogger()

	server, inspectionID, _ := newTestInspectionServer(t, nil)
	runner := server.GetInspection(inspectionID)
	if err := runner.Run(context.Background(), &inspectioncore.InspectionRequest{Values: map[string]any{}}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	<-runner.Wait()

	snapshot, err := InspectRunTaskGraph(server, inspectionID)
	if err != nil {
		t.Fatalf("InspectRunTaskGraph() failed: %v", err)
	}

	nodeStatuses := snapshot.GetNodeStatuses()
	if len(nodeStatuses) < 2 {
		t.Fatalf("len(NodeStatuses) = %d, want at least 2 to verify the ordering", len(nodeStatuses))
	}
	for i := 1; i < len(nodeStatuses); i++ {
		previous := nodeStatuses[i-1].GetTaskImplementationId()
		current := nodeStatuses[i].GetTaskImplementationId()
		if previous >= current {
			t.Errorf("NodeStatuses is not sorted: index %d is %q but index %d is %q", i-1, previous, i, current)
		}
	}
}

func TestConvertTaskRunStatuses(t *testing.T) {
	startTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	endTime := startTime.Add(time.Second)

	testCases := []struct {
		name     string
		statuses map[string]coretask.TaskRunStatus
		want     []*apiv1.TaskRunNodeStatus
	}{
		{
			name: "keeps both timestamps at zero while the task waits",
			statuses: map[string]coretask.TaskRunStatus{
				"task#default": {Phase: coretask.TaskRunPhaseWaiting},
			},
			want: []*apiv1.TaskRunNodeStatus{
				{
					TaskImplementationId: proto.String("task#default"),
					Phase:                apiv1.TaskRunPhase_TASK_RUN_PHASE_WAITING.Enum(),
					StartTimeUnixNano:    proto.Int64(0),
					EndTimeUnixNano:      proto.Int64(0),
				},
			},
		},
		{
			name: "keeps the end timestamp at zero while the task runs",
			statuses: map[string]coretask.TaskRunStatus{
				"task#default": {
					Phase:     coretask.TaskRunPhaseRunning,
					StartTime: startTime,
				},
			},
			want: []*apiv1.TaskRunNodeStatus{
				{
					TaskImplementationId: proto.String("task#default"),
					Phase:                apiv1.TaskRunPhase_TASK_RUN_PHASE_RUNNING.Enum(),
					StartTimeUnixNano:    proto.Int64(startTime.UnixNano()),
					EndTimeUnixNano:      proto.Int64(0),
				},
			},
		},
		{
			name: "reports both timestamps after a successful task",
			statuses: map[string]coretask.TaskRunStatus{
				"task#default": {
					Phase:     coretask.TaskRunPhaseDone,
					StartTime: startTime,
					EndTime:   endTime,
				},
			},
			want: []*apiv1.TaskRunNodeStatus{
				{
					TaskImplementationId: proto.String("task#default"),
					Phase:                apiv1.TaskRunPhase_TASK_RUN_PHASE_DONE.Enum(),
					StartTimeUnixNano:    proto.Int64(startTime.UnixNano()),
					EndTimeUnixNano:      proto.Int64(endTime.UnixNano()),
				},
			},
		},
		{
			name: "reports both timestamps after a failed task",
			statuses: map[string]coretask.TaskRunStatus{
				"task#default": {
					Phase:     coretask.TaskRunPhaseError,
					StartTime: startTime,
					EndTime:   endTime,
				},
			},
			want: []*apiv1.TaskRunNodeStatus{
				{
					TaskImplementationId: proto.String("task#default"),
					Phase:                apiv1.TaskRunPhase_TASK_RUN_PHASE_ERROR.Enum(),
					StartTimeUnixNano:    proto.Int64(startTime.UnixNano()),
					EndTimeUnixNano:      proto.Int64(endTime.UnixNano()),
				},
			},
		},
		{
			name: "falls back to the unspecified phase for an unknown phase",
			statuses: map[string]coretask.TaskRunStatus{
				"task#default": {Phase: coretask.TaskRunPhase(99)},
			},
			want: []*apiv1.TaskRunNodeStatus{
				{
					TaskImplementationId: proto.String("task#default"),
					Phase:                apiv1.TaskRunPhase_TASK_RUN_PHASE_UNSPECIFIED.Enum(),
					StartTimeUnixNano:    proto.Int64(0),
					EndTimeUnixNano:      proto.Int64(0),
				},
			},
		},
		{
			name: "sorts multiple statuses deterministically by task implementation id",
			statuses: map[string]coretask.TaskRunStatus{
				"c#default": {Phase: coretask.TaskRunPhaseDone},
				"a#default": {Phase: coretask.TaskRunPhaseWaiting},
				"b#default": {Phase: coretask.TaskRunPhaseRunning},
			},
			want: []*apiv1.TaskRunNodeStatus{
				{
					TaskImplementationId: proto.String("a#default"),
					Phase:                apiv1.TaskRunPhase_TASK_RUN_PHASE_WAITING.Enum(),
					StartTimeUnixNano:    proto.Int64(0),
					EndTimeUnixNano:      proto.Int64(0),
				},
				{
					TaskImplementationId: proto.String("b#default"),
					Phase:                apiv1.TaskRunPhase_TASK_RUN_PHASE_RUNNING.Enum(),
					StartTimeUnixNano:    proto.Int64(0),
					EndTimeUnixNano:      proto.Int64(0),
				},
				{
					TaskImplementationId: proto.String("c#default"),
					Phase:                apiv1.TaskRunPhase_TASK_RUN_PHASE_DONE.Enum(),
					StartTimeUnixNano:    proto.Int64(0),
					EndTimeUnixNano:      proto.Int64(0),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := convertTaskRunStatuses(tc.statuses)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("convertTaskRunStatuses() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// findNodeStatus returns the status of the given task implementation ID, or nil when it is absent.
func findNodeStatus(nodeStatuses []*apiv1.TaskRunNodeStatus, taskImplID string) *apiv1.TaskRunNodeStatus {
	for _, nodeStatus := range nodeStatuses {
		if nodeStatus.GetTaskImplementationId() == taskImplID {
			return nodeStatus
		}
	}
	return nil
}
