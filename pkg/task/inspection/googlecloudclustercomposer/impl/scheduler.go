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
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var AirflowSchedulerLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudclustercomposer_contract.AirflowSchedulerLogGrouperTaskID,
	googlecloudclustercomposer_contract.ComposerSchedulerFieldSetReadTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		return ""
	},
)

var AirflowSchedulerLogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	googlecloudclustercomposer_contract.AirflowSchedulerLogIngesterTaskID,
	googlecloudclustercomposer_contract.ComposerSchedulerFieldSetReadTaskID.Ref(),
)

var AirflowSchedulerLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](
	googlecloudclustercomposer_contract.AirflowSchedulerLogToTimelineMapperTaskID,
	&airflowSchedulerLogToTimelineMapperSetting{
		targetLogType: enum.LogTypeComposerEnvironment,
	},
	inspectioncore_contract.FeatureTaskLabel(
		"Airflow Scheduler",
		"Airflow Scheduler logs contain information related to the scheduling of TaskInstances, making it an ideal source for understanding the lifecycle of TaskInstances.",
		enum.LogTypeComposerEnvironment,
		100000,
		true,
		googlecloudinspectiontypegroup_contract.CloudComposerInspectionTypes...,
	),
)

type airflowSchedulerLogToTimelineMapperSetting struct {
	targetLogType enum.LogType
}

func (c *airflowSchedulerLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (c *airflowSchedulerLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudclustercomposer_contract.AirflowSchedulerLogGrouperTaskID.Ref()
}

func (c *airflowSchedulerLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowSchedulerLogIngesterTaskID.Ref()
}

func (c *airflowSchedulerLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	schedulerField, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerSchedulerFieldSet{})
	if err == nil && schedulerField.SchedulerID != "" {
		scheduler := googlecloudclustercomposer_contract.NewAirflowScheduler(schedulerField.SchedulerID, "airflow-scheduler")
		cs.AddEvent(scheduler.ResourcePath())
	}

	mainMessage, err := log.GetFieldSet(l, &log.MainMessageFieldSet{})
	if err == nil {
		cs.SetLogSummary(mainMessage.MainMessage)
	}

	commonField, _ := log.GetFieldSet(l, &log.CommonFieldSet{})
	tiField, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerTaskInstanceFieldSet{})
	if err != nil {
		return struct{}{}, nil // Not an Airflow TaskInstance log
	}
	ti := tiField.TaskInstance

	resourcePath := ti.ResourcePath()
	verb, state := tiStatusToVerb(ti)
	cs.AddRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       verb,
		State:      state,
		Requestor:  "airflow-scheduler",
		ChangeTime: commonField.Timestamp,
		Partial:    false,
		Body:       ti.ToYaml(),
	})

	cs.AddEvent(resourcePath)

	// if the ti status is zombie, record it on worker
	if ti.Status() == googlecloudclustercomposer_contract.TASKINSTANCE_ZOMBIE && ti.Host() != "" {
		host := googlecloudclustercomposer_contract.NewAirflowWorker(ti.Host())
		cs.AddEvent(host.ResourcePath())
	}

	return struct{}{}, nil
}
