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
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AirflowWorkerLogGrouperTask groups Airflow worker logs.
var AirflowWorkerLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudclustercomposer_contract.AirflowWorkerLogGrouperTaskID,
	googlecloudclustercomposer_contract.AirflowWorkerLogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		return ""
	},
)

type workerLogIngester struct{}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *workerLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowWorkerLogFilterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *workerLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog is called for each log entry to customize log metadata (summary, severity, timestamp, etc.).
func (i *workerLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
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

	if messageFS, err := log.GetFieldSet(l, &log.MainMessageFieldSet{}); err == nil {
		cs.SetSummary(messageFS.MainMessage)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngesterV2 = (*workerLogIngester)(nil)

// AirflowWorkerLogIngesterTask is the task that ingests Airflow worker logs.
var AirflowWorkerLogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
	googlecloudclustercomposer_contract.AirflowWorkerLogIngesterTaskID,
	&workerLogIngester{},
)

type workerLogToTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask returns a reference to the ingester task.
func (m *workerLogToTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowWorkerLogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the mapper.
func (m *workerLogToTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *workerLogToTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudclustercomposer_contract.AirflowWorkerLogGrouperTaskID.Ref()
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *workerLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	envPath := googlecloudclustercomposer_contract.MustComposerEnvironmentTimeline(ctx, clusterIdentity.ProjectID, environmentName)

	workerField, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerFieldSet{})
	cs := khifilev6.NewTimelineChangeSet(l)

	if err == nil {
		if workerField.WorkerID != "" {
			workerTimelinePath := googlecloudclustercomposer_contract.MustAirflowWorkerTimeline(ctx, envPath, workerField.WorkerID)
			cs.AddEvent(workerTimelinePath)
		}
	}

	commonField, _ := log.GetFieldSet(l, &log.CommonFieldSet{})
	workerTiField, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerWorkerTaskInstanceFieldSet{})
	if err != nil || workerTiField.TaskInstance == nil {
		return cs, struct{}{}, nil
	}
	ti := workerTiField.TaskInstance
	var detail = ti.TaskId()
	if ti.MapIndex() != "-1" {
		detail += "+" + ti.MapIndex()
	}
	runPath := googlecloudclustercomposer_contract.MustAirflowDAGRunTimeline(ctx, envPath, ti.DagId(), ti.RunId())
	timelinePath := googlecloudclustercomposer_contract.MustAirflowTaskInstanceTimeline(ctx, runPath, detail)

	if ti.Status() == googlecloudclustercomposer_contract.TASKINSTANCE_NONE {
		cs.AddEvent(timelinePath)
	} else {
		verb, state := tiStatusToVerb(ti)
		node, err := structured.FromYAML(ti.ToYaml())
		if err != nil {
			node = structured.NewStandardScalarNode(ti.ToYaml())
		}
		cs.AddRevision(timelinePath, &khifilev6.StagingRevision{
			ChangedTime:  commonField.Timestamp,
			ResourceBody: node,
			Principal:    "airflow-worker",
			VerbType:     verb,
			StateType:    state,
		})
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*workerLogToTimelineMapper)(nil)

// AirflowWorkerLogToTimelineMapperTask is the task that maps Airflow worker logs to timeline events.
var AirflowWorkerLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(
	googlecloudclustercomposer_contract.AirflowWorkerLogToTimelineMapperTaskID,
	&workerLogToTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabelV2(
		"Airflow Worker Logs",
		"Timeline mapping for worker logs under Apache Airflow.",
		1502,
		false,
	),
)
