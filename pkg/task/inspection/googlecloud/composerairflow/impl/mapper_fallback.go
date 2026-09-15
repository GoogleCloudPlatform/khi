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

package composerairflow_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerairflow"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

// AirflowOtherLogGrouperTask groups other Airflow logs.
var AirflowOtherLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	composerairflow.AirflowOtherLogGrouperTaskID,
	composerairflow.AirflowOtherLogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		return ""
	},
)

type otherLogIngester struct{}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *otherLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return composerairflow.AirflowOtherLogFilterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *otherLogIngester) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{}
}

// ProcessLog is called for each log entry to customize log metadata (summary, severity, timestamp, etc.).
func (i *otherLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(composerairflow.LogTypeManagedAirflowEnvironment)
	cs.SetTimestamp(l.Timestamp)

	if severity, err := gcpcommon.ExtractGCPSeverity(l.NodeReader); err == nil {
		cs.SetSeverity(severity)
	}

	if message, err := gcpcommon.ExtractGCPMainMessage(l.NodeReader); err == nil {
		cs.SetSummary(message)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngester = (*otherLogIngester)(nil)

// AirflowOtherLogIngesterTask is the task that ingests other Airflow logs.
var AirflowOtherLogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	composerairflow.AirflowOtherLogIngesterTaskID,
	&otherLogIngester{},
)

type otherLogToTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask returns a reference to the ingester task.
func (m *otherLogToTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return composerairflow.AirflowOtherLogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the mapper.
func (m *otherLogToTimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		composercluster.InputComposerEnvironmentNameTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *otherLogToTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return composerairflow.AirflowOtherLogGrouperTaskID.Ref()
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *otherLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	environmentName := coretask.GetTaskResult(ctx, composercluster.InputComposerEnvironmentNameTaskID.Ref())
	envPath := composerairflow.MustAirflowTimeline(ctx, environmentName)

	composerFieldSet, err := composerairflow.ExtractComposer(l.NodeReader)
	if err != nil {
		return nil, struct{}{}, nil
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	componentName := composerFieldSet.Component
	if componentName == "" {
		componentName = "unknown-component"
	}

	mappedToTimeline := false
	if composerFieldSet.WorkerID != "" {
		cs.AddEvent(composerairflow.MustAirflowComponentTimeline(ctx, envPath, composerFieldSet.WorkerID))
		mappedToTimeline = true
	}

	if composerFieldSet.SchedulerID != "" {
		cs.AddEvent(composerairflow.MustAirflowComponentTimeline(ctx, envPath, composerFieldSet.SchedulerID))
		mappedToTimeline = true
	}

	if composerFieldSet.DagProcessorManagerID != "" {
		cs.AddEvent(composerairflow.MustAirflowComponentTimeline(ctx, envPath, composerFieldSet.DagProcessorManagerID))
		mappedToTimeline = true
	}

	if composerFieldSet.TriggererID != "" {
		cs.AddEvent(composerairflow.MustAirflowComponentTimeline(ctx, envPath, composerFieldSet.TriggererID))
		mappedToTimeline = true
	}

	if composerFieldSet.WebserverID != "" {
		cs.AddEvent(composerairflow.MustAirflowComponentTimeline(ctx, envPath, composerFieldSet.WebserverID))
		mappedToTimeline = true
	}

	if !mappedToTimeline {
		if composerFieldSet.Subservice != "" {
			componentName = composerFieldSet.Subservice
		}
		cs.AddEvent(composerairflow.MustAirflowComponentTimeline(ctx, envPath, componentName))
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*otherLogToTimelineMapper)(nil)

// AirflowOtherLogToTimelineMapperTask is the task that maps other Airflow logs to timeline events.
var AirflowOtherLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	composerairflow.AirflowOtherLogToTimelineMapperTaskID,
	&otherLogToTimelineMapper{},
)
