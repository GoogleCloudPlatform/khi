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

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AirflowSchedulerLogGrouperTask groups Airflow scheduler logs.
var AirflowSchedulerLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudclustercomposer_contract.AirflowSchedulerLogGrouperTaskID,
	googlecloudclustercomposer_contract.AirflowSchedulerLogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		return ""
	},
)

type schedulerLogIngester struct{}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *schedulerLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowSchedulerLogFilterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *schedulerLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog is called for each log entry to customize log metadata (summary, severity, timestamp, etc.).
func (i *schedulerLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(googlecloudclustercomposer_contract.LogTypeComposerEnvironment)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severityFS.Severity)
	}

	if cs.Severity == nil {
		cs.SetSeverity(inspectioncore_contract.SeverityUnknown)
	}

	if messageFS, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPMainMessageFieldSet{}); err == nil {
		cs.SetSummary(messageFS.MainMessage)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngesterV2 = (*schedulerLogIngester)(nil)

// AirflowSchedulerLogIngesterTask is the task that ingests Airflow scheduler logs.
var AirflowSchedulerLogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
	googlecloudclustercomposer_contract.AirflowSchedulerLogIngesterTaskID,
	&schedulerLogIngester{},
)

type schedulerLogToTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask returns a reference to the ingester task.
func (m *schedulerLogToTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowSchedulerLogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the mapper.
func (m *schedulerLogToTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *schedulerLogToTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudclustercomposer_contract.AirflowSchedulerLogGrouperTaskID.Ref()
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *schedulerLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	envPath := googlecloudclustercomposer_contract.MustComposerEnvironmentTimeline(ctx, clusterIdentity.ProjectID, environmentName)

	schedulerField, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerFieldSet{})
	cs := khifilev6.NewTimelineChangeSet(l)

	if err == nil {
		if schedulerField.SchedulerID != "" {
			schedulerTimelinePath := googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowScheduler, schedulerField.SchedulerID)
			cs.AddEvent(schedulerTimelinePath)
		}
	}

	commonField, _ := log.GetFieldSet(l, &log.CommonFieldSet{})
	tiField, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerTaskInstanceFieldSet{})
	if err != nil || tiField.TaskInstance == nil {
		return cs, struct{}{}, nil // Not an Airflow TaskInstance log
	}
	ti := tiField.TaskInstance
	var detail = ti.TaskId()
	if ti.MapIndex() != "-1" {
		detail += "+" + ti.MapIndex()
	}
	runPath := googlecloudclustercomposer_contract.MustAirflowDAGRunTimeline(ctx, envPath, ti.DagId(), ti.RunId())
	timelinePath := googlecloudclustercomposer_contract.MustAirflowTaskInstanceTimeline(ctx, runPath, detail)
	verb, state := tiStatusToVerb(ti)

	node, err := structured.FromYAML(ti.ToYaml())
	if err != nil {
		node = structured.NewStandardScalarNode(ti.ToYaml())
	}

	cs.AddRevision(timelinePath, &khifilev6.StagingRevision{
		ChangedTime:  commonField.Timestamp,
		ResourceBody: node,
		Principal:    "airflow-scheduler",
		VerbType:     verb,
		StateType:    state,
	})

	cs.AddEvent(timelinePath)

	// If the ti status is zombie, record it on worker
	if ti.Status() == googlecloudclustercomposer_contract.TASKINSTANCE_ZOMBIE && ti.Host() != "" {
		workerTimelinePath := googlecloudclustercomposer_contract.MustAirflowWorkerTimeline(ctx, envPath, ti.Host())
		cs.AddEvent(workerTimelinePath)
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*schedulerLogToTimelineMapper)(nil)

// AirflowSchedulerLogToTimelineMapperTask is the task that maps Airflow scheduler logs to timeline events.
var AirflowSchedulerLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(
	googlecloudclustercomposer_contract.AirflowSchedulerLogToTimelineMapperTaskID,
	&schedulerLogToTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabelV2(
		"Airflow Scheduler Logs",
		"Gather Apache Airflow scheduler logs to visualize DAG scheduling decisions and latency on resource timelines.",
		1501,
		false,
	),
)
