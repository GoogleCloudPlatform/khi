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

package mcp

import (
	"context"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// WaitInspectionInput defines the input parameters for the wait_inspection tool.
type WaitInspectionInput struct {
	InspectionID string `json:"inspectionId" jsonschema:"The unique identifier of the inspection."`
	// TimeoutSeconds is a pointer to distinguish an omitted value (default wait) from an explicit 0 (no wait).
	TimeoutSeconds *int `json:"timeoutSeconds,omitempty" jsonschema:"Maximum seconds to wait for completion. Defaults to 60 and is capped at 300. Pass 0 to return the current status without waiting."`
}

// CancelInspectionInput defines the input parameters for the cancel_inspection tool.
type CancelInspectionInput struct {
	InspectionID string `json:"inspectionId" jsonschema:"The unique identifier of the inspection."`
}

func inspectionNotStartedResult() (*mcpsdk.CallToolResult, any, error) {
	return mdtemplate.ErrorResult("INSPECTION_NOT_STARTED",
		"Call `run_inspection` to start the inspection.")
}

func inspectionAlreadyFinishedResult() (*mcpsdk.CallToolResult, any, error) {
	return mdtemplate.ErrorResult("INSPECTION_ALREADY_FINISHED",
		"Call `wait_inspection` to read the final status.")
}

// Wait timeouts stay below the tool call timeout of common MCP clients.
const (
	defaultWaitTimeoutSeconds = 60
	maxWaitTimeoutSeconds     = 300
	cancelWaitTimeout         = 10 * time.Second
)

type runningTaskRow struct {
	Label    string
	Progress string
	Message  string
}

// failedTaskRow is a failed task in the wait_inspection error table.
// Title is empty when the task has no progress title, and the task ID is shown instead.
type failedTaskRow struct {
	Title  string
	TaskID string
	Error  string
}

type waitInspectionData struct {
	ID            string
	Phase         string
	ProgressRatio float64
	RunningTasks  []runningTaskRow
	FailedTasks   []failedTaskRow
}

// buildWaitInspectionData builds the wait_inspection template data from the current metadata of runner.
// Running inspections list the progress of running tasks, and failed inspections list the failed
// tasks built by failedTaskRows.
func (h *InspectionHandler) buildWaitInspectionData(id string, runner *coreinspection.InspectionTaskRunner) (waitInspectionData, error) {
	metadata, err := runner.GetCurrentMetadata()
	if err != nil {
		return waitInspectionData{}, err
	}

	progressMeta, found := typedmap.Get(metadata, inspectionmetadata.ProgressMetadataKey)
	if !found || progressMeta == nil {
		// Imported inspections carry no progress metadata and are always complete,
		// matching the DONE fallback of handleListInspections.
		return waitInspectionData{ID: id, Phase: string(inspectionmetadata.TaskPhaseDone)}, nil
	}

	snap := progressMeta.Snapshot()
	data := waitInspectionData{
		ID:    id,
		Phase: string(snap.Phase),
	}

	switch snap.Phase {
	case inspectionmetadata.TaskPhaseRunning:
		var totalRatio float64
		if snap.TotalProgress != nil {
			totalRatio = float64(snap.TotalProgress.Ratio)
		}
		data.ProgressRatio = totalRatio
		if len(snap.TaskProgresses) > 0 {
			tasks := make([]runningTaskRow, 0, len(snap.TaskProgresses))
			for _, tp := range snap.TaskProgresses {
				progressStr := mdtemplate.Percent(float64(tp.Ratio))
				if tp.Indeterminate {
					progressStr = "-"
				}
				tasks = append(tasks, runningTaskRow{
					Label:    tp.Label,
					Progress: progressStr,
					Message:  tp.Message,
				})
			}
			data.RunningTasks = tasks
		}
	case inspectionmetadata.TaskPhaseDone:
		// Done phase does not need additional fields for template
	case inspectionmetadata.TaskPhaseError:
		graphSnap, err := runner.TakeRunTaskGraphSnapshot()
		if err != nil {
			return waitInspectionData{}, err
		}
		data.FailedTasks = failedTaskRows(graphSnap)
	case inspectionmetadata.TaskPhaseCancelled:
		// Cancelled phase does not need additional fields
	}

	return data, nil
}

// failedTaskRows lists each failed task of snap sorted by task ID, with its progress title and the error
// returned by that task.
// A failure cancels the rest of the graph, so running tasks that stopped with context.Canceled are omitted
// unless no task failed for another reason.
func failedTaskRows(snap *coreinspection.RunTaskGraphSnapshot) []failedTaskRow {
	// Titles come from the DAG captured when the run started, so they describe the graph that
	// actually ran even if the feature selection or registered tasks changed afterwards.
	titles := make(map[string]string, len(snap.DAG.GetNodes()))
	for _, node := range snap.DAG.GetNodes() {
		if title := node.GetLabels()[coretask.LabelKeyTaskTitle.Key()]; title != "" {
			titles[node.GetTaskImplementationId()] = title
		}
	}

	var failedRows, cancelledRows []failedTaskRow
	for _, tid := range slices.Sorted(maps.Keys(snap.TaskRunStatuses)) {
		status := snap.TaskRunStatuses[tid]
		if status.Phase != coretask.TaskRunPhaseError {
			continue
		}
		row := failedTaskRow{Title: titles[tid], TaskID: tid, Error: status.Error.Error()}
		if errors.Is(status.Error, context.Canceled) {
			cancelledRows = append(cancelledRows, row)
		} else {
			failedRows = append(failedRows, row)
		}
	}
	if len(failedRows) == 0 {
		return cancelledRows
	}
	return failedRows
}

func (h *InspectionHandler) handleWaitInspection(ctx context.Context, req *mcpsdk.CallToolRequest, in WaitInspectionInput) (*mcpsdk.CallToolResult, any, error) {
	runner := h.server.GetInspection(in.InspectionID)
	if runner == nil {
		return inspectionNotFoundResult(in.InspectionID)
	}

	if !runner.Started() {
		return inspectionNotStartedResult()
	}

	timeout := defaultWaitTimeoutSeconds
	if in.TimeoutSeconds != nil {
		timeout = min(max(*in.TimeoutSeconds, 0), maxWaitTimeoutSeconds)
	}

	if timeout > 0 {
		timer := time.NewTimer(time.Duration(timeout) * time.Second)
		defer timer.Stop()
		select {
		case <-runner.Wait():
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-timer.C:
		}
	}

	data, err := h.buildWaitInspectionData(in.InspectionID, runner)
	if err != nil {
		return nil, nil, err
	}

	return h.templates.ToolResult("wait_inspection.md.tmpl", data)
}

func (h *InspectionHandler) handleCancelInspection(ctx context.Context, req *mcpsdk.CallToolRequest, in CancelInspectionInput) (*mcpsdk.CallToolResult, any, error) {
	runner := h.server.GetInspection(in.InspectionID)
	if runner == nil {
		return inspectionNotFoundResult(in.InspectionID)
	}

	if !runner.Started() {
		return inspectionNotStartedResult()
	}

	if err := runner.Cancel(); err != nil {
		return inspectionAlreadyFinishedResult()
	}

	timer := time.NewTimer(cancelWaitTimeout)
	defer timer.Stop()
	select {
	case <-runner.Wait():
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-timer.C:
	}

	data, err := h.buildWaitInspectionData(in.InspectionID, runner)
	if err != nil {
		return nil, nil, err
	}

	if data.Phase == string(inspectionmetadata.TaskPhaseCancelled) {
		return h.templates.ToolResult("cancel_inspection.md.tmpl", inspectionIDData{
			ID: in.InspectionID,
		})
	}
	return h.templates.ToolResult("wait_inspection.md.tmpl", data)
}
