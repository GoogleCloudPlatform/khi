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

// airflowSchedulerLogGrouperTask groups Airflow scheduler logs.
var airflowSchedulerLogGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	composerairflow.AirflowSchedulerLogGrouperTaskID,
	composerairflow.AirflowSchedulerLogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			return ""
		}
	},
)

// airflowSchedulerLogIngesterTask is the task that ingests Airflow scheduler logs.
var airflowSchedulerLogIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	composerairflow.AirflowSchedulerLogIngesterTaskID,
	composerairflow.AirflowSchedulerLogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		return processAirflowLog
	},
)

type schedulerLogToTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	environmentName coretask.Input[string]
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *schedulerLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	return mapSchedulerLog(ctx, l, m.environmentName.Get(ctx)), struct{}{}, nil
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*schedulerLogToTimelineMapper)(nil)

// mapSchedulerLog maps an Airflow scheduler log to timeline changes for the given environment.
func mapSchedulerLog(ctx context.Context, l *log.Log, environmentName string) *khifilev6.TimelineChangeSet {
	envPath := composerairflow.MustAirflowTimeline(ctx, environmentName)

	schedulerField, err := composerairflow.ExtractComposer(l.NodeReader)
	cs := khifilev6.NewTimelineChangeSet(l)

	if err == nil {
		if schedulerField.SchedulerID != "" {
			schedulerTimelinePath := composerairflow.MustAirflowComponentTimeline(ctx, envPath, schedulerField.SchedulerID)
			cs.AddEvent(schedulerTimelinePath)
		}
	}

	tiField, err := composerairflow.ExtractComposerTaskInstance(l.NodeReader)
	if err != nil || tiField.TaskInstance == nil {
		return cs // Not an Airflow TaskInstance log
	}
	ti := tiField.TaskInstance
	var detail = ti.TaskId()
	if ti.MapIndex() != "-1" {
		detail += "+" + ti.MapIndex()
	}
	runPath := composerairflow.MustAirflowDAGRunTimeline(ctx, envPath, ti.DagId(), ti.RunId())
	timelinePath := composerairflow.MustAirflowTaskInstanceTimeline(ctx, runPath, detail)
	verb, state := tiStatusToVerb(ti)

	node, err := structured.FromYAML(ti.ToYaml())
	if err != nil {
		node = structured.NewStandardScalarNode(ti.ToYaml())
	}

	cs.AddRevision(timelinePath, &khifilev6.StagingRevision{
		ChangedTime:  l.Timestamp,
		ResourceBody: node,
		Principal:    "airflow-scheduler",
		VerbType:     verb,
		StateType:    state,
	})

	cs.AddEvent(timelinePath)

	// If the ti status is zombie, record it on worker
	if ti.Status() == composerairflow.TASKINSTANCE_ZOMBIE && ti.Host() != "" {
		workerTimelinePath := composerairflow.MustAirflowComponentTimeline(ctx, envPath, ti.Host())
		cs.AddEvent(workerTimelinePath)
	}

	return cs
}

// airflowSchedulerLogToTimelineMapperTask is the task that maps Airflow scheduler logs to timeline events.
var airflowSchedulerLogToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	composerairflow.AirflowSchedulerLogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: composerairflow.AirflowSchedulerLogIngesterTaskID.Ref(),
		GroupedLogs: composerairflow.AirflowSchedulerLogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &schedulerLogToTimelineMapper{
			environmentName: coretask.Use(b, composercluster.InputComposerEnvironmentNameTaskID.Ref()),
		}
	},
)
