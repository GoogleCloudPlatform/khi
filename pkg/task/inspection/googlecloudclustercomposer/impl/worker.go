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
)

var AirflowWorkerLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudclustercomposer_contract.AirflowWorkerLogGrouperTaskID,
	googlecloudclustercomposer_contract.AirflowWorkerLogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		return ""
	},
)

var AirflowWorkerLogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	googlecloudclustercomposer_contract.AirflowWorkerLogIngesterTaskID,
	googlecloudclustercomposer_contract.AirflowWorkerLogFilterTaskID.Ref(),
)

var AirflowWorkerLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](
	googlecloudclustercomposer_contract.AirflowWorkerLogToTimelineMapperTaskID,
	&airflowWorkerLogToTimelineMapperSetting{
		targetLogType: enum.LogTypeComposerEnvironment,
	},
)

type airflowWorkerLogToTimelineMapperSetting struct {
	targetLogType enum.LogType
}

func (c *airflowWorkerLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (c *airflowWorkerLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudclustercomposer_contract.AirflowWorkerLogGrouperTaskID.Ref()
}

func (c *airflowWorkerLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowWorkerLogIngesterTaskID.Ref()
}

func (c *airflowWorkerLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	workerField, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerFieldSet{})
	if err == nil {
		if workerField.WorkerID != "" {
			worker := googlecloudclustercomposer_contract.NewAirflowWorker(workerField.WorkerID)
			cs.AddEvent(worker.ResourcePath())
		}
	}

	mainMessage, err := log.GetFieldSet(l, &log.MainMessageFieldSet{})
	if err == nil {
		cs.SetLogSummary(mainMessage.MainMessage)
	}

	workerTiField, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerWorkerTaskInstanceFieldSet{})
	if err != nil {
		return struct{}{}, nil
	}
	ti := workerTiField.TaskInstance

	commonField, _ := log.GetFieldSet(l, &log.CommonFieldSet{})

	r := ti.ResourcePath()
	verb, state := tiStatusToVerb(ti)
	cs.AddRevision(r, &history.StagingResourceRevision{
		Verb:       verb,
		State:      state,
		Requestor:  "airflow-worker",
		ChangeTime: commonField.Timestamp,
		Partial:    false,
		Body:       ti.ToYaml(),
	})

	return struct{}{}, nil
}
