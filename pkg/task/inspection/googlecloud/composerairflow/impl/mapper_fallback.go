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
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerairflow"
)

// airflowOtherLogGrouperTask groups other Airflow logs.
var airflowOtherLogGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	composerairflow.AirflowOtherLogGrouperTaskID,
	composerairflow.AirflowOtherLogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			return ""
		}
	},
)

// airflowOtherLogIngesterTask is the task that ingests other Airflow logs.
var airflowOtherLogIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	composerairflow.AirflowOtherLogIngesterTaskID,
	composerairflow.AirflowOtherLogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		return processAirflowLog
	},
)

type otherLogToTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	environmentName coretask.Input[string]
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *otherLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	return mapOtherLog(ctx, l, m.environmentName.Get(ctx)), struct{}{}, nil
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*otherLogToTimelineMapper)(nil)

// mapOtherLog maps other Airflow logs to timeline changes for the given environment.
func mapOtherLog(ctx context.Context, l *log.Log, environmentName string) *khifilev6.TimelineChangeSet {
	envPath := composerairflow.MustAirflowTimeline(ctx, environmentName)

	composerFieldSet, err := composerairflow.ExtractComposer(l.NodeReader)
	if err != nil {
		return nil
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

	return cs
}

// airflowOtherLogToTimelineMapperTask is the task that maps other Airflow logs to timeline events.
var airflowOtherLogToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	composerairflow.AirflowOtherLogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: composerairflow.AirflowOtherLogIngesterTaskID.Ref(),
		GroupedLogs: composerairflow.AirflowOtherLogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &otherLogToTimelineMapper{
			environmentName: coretask.Use(b, composercluster.InputComposerEnvironmentNameTaskID.Ref()),
		}
	},
)
