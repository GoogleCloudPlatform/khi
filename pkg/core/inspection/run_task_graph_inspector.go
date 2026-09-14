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
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"google.golang.org/protobuf/proto"
)

// ErrInspectionNotFound is returned when the requested inspection ID is not registered on the server.
var ErrInspectionNotFound = errors.New("inspection not found")

// InspectRunTaskGraph returns the execution graph of an inspection run together with the current
// phase of each task, so clients can visualize how far the run has progressed.
func InspectRunTaskGraph(server *InspectionTaskServer, inspectionID string) (*apiv1.InspectionRunTaskGraphSnapshot, error) {
	runner := server.GetInspection(inspectionID)
	if runner == nil {
		return nil, fmt.Errorf("inspection %s: %w", inspectionID, ErrInspectionNotFound)
	}

	snapshot, err := runner.TakeRunTaskGraphSnapshot()
	if err != nil {
		return nil, err
	}

	return &apiv1.InspectionRunTaskGraphSnapshot{
		Dag:                  snapshot.DAG,
		NodeStatuses:         convertTaskRunStatuses(snapshot.TaskRunStatuses),
		IsRunFinished:        proto.Bool(snapshot.IsRunFinished),
		SnapshotTimeUnixNano: proto.Int64(time.Now().UnixNano()),
	}, nil
}

// convertTaskRunStatuses formats task run statuses in a stable order so repeated snapshots stay diff-friendly.
func convertTaskRunStatuses(statuses map[string]coretask.TaskRunStatus) []*apiv1.TaskRunNodeStatus {
	nodeStatuses := make([]*apiv1.TaskRunNodeStatus, 0, len(statuses))
	for _, taskImplID := range slices.Sorted(maps.Keys(statuses)) {
		status := statuses[taskImplID]
		nodeStatuses = append(nodeStatuses, &apiv1.TaskRunNodeStatus{
			TaskImplementationId: proto.String(taskImplID),
			Phase:                convertTaskRunPhase(status.Phase),
			StartTimeUnixNano:    proto.Int64(unixNanoOrZero(status.StartTime)),
			EndTimeUnixNano:      proto.Int64(unixNanoOrZero(status.EndTime)),
		})
	}
	return nodeStatuses
}

// unixNanoOrZero converts a time to Unix nanoseconds, mapping the zero time to 0 instead of a large negative value.
func unixNanoOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

// convertTaskRunPhase maps the core task run phase to its API enum, falling back to UNSPECIFIED for unknown phases.
func convertTaskRunPhase(phase coretask.TaskRunPhase) *apiv1.TaskRunPhase {
	var val apiv1.TaskRunPhase
	switch phase {
	case coretask.TaskRunPhaseWaiting:
		val = apiv1.TaskRunPhase_TASK_RUN_PHASE_WAITING
	case coretask.TaskRunPhaseRunning:
		val = apiv1.TaskRunPhase_TASK_RUN_PHASE_RUNNING
	case coretask.TaskRunPhaseDone:
		val = apiv1.TaskRunPhase_TASK_RUN_PHASE_DONE
	case coretask.TaskRunPhaseError:
		val = apiv1.TaskRunPhase_TASK_RUN_PHASE_ERROR
	default:
		val = apiv1.TaskRunPhase_TASK_RUN_PHASE_UNSPECIFIED
	}
	return val.Enum()
}
