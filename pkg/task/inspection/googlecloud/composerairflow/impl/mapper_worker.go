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

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerairflow"
)

// airflowWorkerLogGrouperTask groups Airflow worker logs.
var airflowWorkerLogGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	composerairflow.AirflowWorkerLogGrouperTaskID,
	composerairflow.AirflowWorkerLogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			return ""
		}
	},
)

// airflowWorkerLogIngesterTask is the task that ingests Airflow worker logs.
var airflowWorkerLogIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	composerairflow.AirflowWorkerLogIngesterTaskID,
	composerairflow.AirflowWorkerLogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		return processAirflowLog
	},
)

type workerLogToTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	environmentName coretask.Input[string]
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *workerLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	return mapWorkerLog(ctx, l, m.environmentName.Get(ctx)), struct{}{}, nil
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*workerLogToTimelineMapper)(nil)

// mapWorkerLog maps an Airflow worker log to timeline changes for the given environment.
func mapWorkerLog(ctx context.Context, l *log.Log, environmentName string) *khifilev6.TimelineChangeSet {
	envPath := composerairflow.MustAirflowTimeline(ctx, environmentName)

	workerField, err := composerairflow.ExtractComposer(l.NodeReader)
	cs := khifilev6.NewTimelineChangeSet(l)

	if err == nil {
		if workerField.WorkerID != "" {
			workerTimelinePath := composerairflow.MustAirflowComponentTimeline(ctx, envPath, workerField.WorkerID)
			cs.AddEvent(workerTimelinePath)
		}
	}

	workerTiField, err := composerairflow.ExtractComposerWorkerTaskInstance(l.NodeReader)
	if err != nil || workerTiField.TaskInstance == nil {
		return cs
	}
	ti := workerTiField.TaskInstance
	var detail = ti.TaskId()
	if ti.MapIndex() != "-1" {
		detail += "+" + ti.MapIndex()
	}
	runPath := composerairflow.MustAirflowDAGRunTimeline(ctx, envPath, ti.DagId(), ti.RunId())
	timelinePath := composerairflow.MustAirflowTaskInstanceTimeline(ctx, runPath, detail)

	if ti.Status() == composerairflow.TASKINSTANCE_NONE {
		cs.AddEvent(timelinePath)
	} else {
		verb, state := tiStatusToVerb(ti)
		node, err := structured.FromYAML(ti.ToYaml())
		if err != nil {
			node = structured.NewStandardScalarNode(ti.ToYaml())
		}
		cs.AddRevision(timelinePath, &khifilev6.StagingRevision{
			ChangedTime:  l.Timestamp,
			ResourceBody: node,
			Principal:    "airflow-worker",
			VerbType:     verb,
			StateType:    state,
		})
	}

	return cs
}

// airflowWorkerLogToTimelineMapperTask is the task that maps Airflow worker logs to timeline events.
var airflowWorkerLogToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	composerairflow.AirflowWorkerLogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: composerairflow.AirflowWorkerLogIngesterTaskID.Ref(),
		GroupedLogs: composerairflow.AirflowWorkerLogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &workerLogToTimelineMapper{
			environmentName: coretask.Use(b, composercluster.InputComposerEnvironmentNameTaskID.Ref()),
		}
	},
)
