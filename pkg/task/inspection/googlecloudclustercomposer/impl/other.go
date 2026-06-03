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

package googlecloudclustercomposer_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AirflowOtherLogGrouperTask groups other Airflow logs.
var AirflowOtherLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudclustercomposer_contract.AirflowOtherLogGrouperTaskID,
	googlecloudclustercomposer_contract.AirflowOtherLogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		return ""
	},
)

type otherLogIngester struct{}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *otherLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowOtherLogFilterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *otherLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog is called for each log entry to customize log metadata (summary, severity, timestamp, etc.).
func (i *otherLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(googlecloudclustercomposer_contract.LogTypeComposerEnvironment)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		if severityFS.Severity != nil && severityFS.Severity.Id != nil {
			if severityFS.Severity.GetId() == inspectioncore_contract.SeverityError.GetId() ||
				severityFS.Severity.GetId() == inspectioncore_contract.SeverityWarning.GetId() {
				cs.SetSeverity(severityFS.Severity)
			}
		}
	}

	// Ensure severity is at least SeverityUnknown if not set.
	if cs.Severity == nil {
		cs.SetSeverity(inspectioncore_contract.SeverityUnknown)
	}

	if messageFS, err := log.GetFieldSet(l, &log.MainMessageFieldSet{}); err == nil {
		cs.SetSummary(messageFS.MainMessage)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngesterV2 = (*otherLogIngester)(nil)

// AirflowOtherLogIngesterTask is the task that ingests other Airflow logs.
var AirflowOtherLogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
	googlecloudclustercomposer_contract.AirflowOtherLogIngesterTaskID,
	&otherLogIngester{},
)

type otherLogToTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask returns a reference to the ingester task.
func (m *otherLogToTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowOtherLogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the mapper.
func (m *otherLogToTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *otherLogToTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudclustercomposer_contract.AirflowOtherLogGrouperTaskID.Ref()
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *otherLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	envPath := googlecloudclustercomposer_contract.MustComposerEnvironmentTimeline(ctx, clusterIdentity.ProjectID, environmentName)

	composerFieldSet, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerFieldSet{})
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
		cs.AddEvent(googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowWorker, composerFieldSet.WorkerID))
		mappedToTimeline = true
	}

	if composerFieldSet.SchedulerID != "" {
		cs.AddEvent(googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowScheduler, composerFieldSet.SchedulerID))
		mappedToTimeline = true
	}

	if composerFieldSet.DagProcessorManagerID != "" {
		cs.AddEvent(googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowDagProcessorManager, composerFieldSet.DagProcessorManagerID))
		mappedToTimeline = true
	}

	if composerFieldSet.TriggererID != "" {
		cs.AddEvent(googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowTriggerer, composerFieldSet.TriggererID))
		mappedToTimeline = true
	}

	if composerFieldSet.WebserverID != "" {
		cs.AddEvent(googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowWebserver, composerFieldSet.WebserverID))
		mappedToTimeline = true
	}

	if !mappedToTimeline {
		if composerFieldSet.Subservice != "" {
			componentName = composerFieldSet.Subservice
		}
		cs.AddEvent(googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowComponent, componentName))
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*otherLogToTimelineMapper)(nil)

// AirflowOtherLogToTimelineMapperTask is the task that maps other Airflow logs to timeline events.
var AirflowOtherLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(
	googlecloudclustercomposer_contract.AirflowOtherLogToTimelineMapperTaskID,
	&otherLogToTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabelV2(
		"Airflow Other Components Logs",
		"Timeline mapping for other components' logs under Apache Airflow.",
		1505,
		false,
	),
)
