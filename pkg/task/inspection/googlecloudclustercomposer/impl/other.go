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
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
)

var AirflowOtherLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudclustercomposer_contract.AirflowOtherLogGrouperTaskID,
	googlecloudclustercomposer_contract.AirflowOtherLogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		return ""
	},
)

var AirflowOtherLogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	googlecloudclustercomposer_contract.AirflowOtherLogIngesterTaskID,
	googlecloudclustercomposer_contract.AirflowOtherLogFilterTaskID.Ref(),
)

var AirflowOtherLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](
	googlecloudclustercomposer_contract.AirflowOtherLogToTimelineMapperTaskID,
	&airflowOtherLogToTimelineMapperSetting{},
)

type airflowOtherLogToTimelineMapperSetting struct{}

func (c *airflowOtherLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (c *airflowOtherLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudclustercomposer_contract.AirflowOtherLogGrouperTaskID.Ref()
}

func (c *airflowOtherLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowOtherLogIngesterTaskID.Ref()
}

func (c *airflowOtherLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	composerFieldSet, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerFieldSet{})
	if err != nil {
		return struct{}{}, nil
	}

	mainMessage, err := log.GetFieldSet(l, &log.MainMessageFieldSet{})
	if err == nil {
		cs.SetLogSummary(mainMessage.MainMessage)
	}

	commonField, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err == nil {
		if commonField.Severity == enum.SeverityError || commonField.Severity == enum.SeverityWarning {
			cs.SetLogSeverity(commonField.Severity)
		}
	}

	componentName := composerFieldSet.Component
	if componentName == "" {
		componentName = "unknown-component"
	}

	mappedToTimeline := false
	if composerFieldSet.WorkerID != "" {
		cs.AddEvent(resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "AirflowWorker", "cluster-scope", composerFieldSet.WorkerID, composerFieldSet.Component))
		mappedToTimeline = true
	}

	if composerFieldSet.SchedulerID != "" {
		cs.AddEvent(resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "AirflowScheduler", "cluster-scope", composerFieldSet.SchedulerID, composerFieldSet.Component))
		mappedToTimeline = true
	}

	if composerFieldSet.DagProcessorManagerID != "" {
		cs.AddEvent(resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "AirflowDagProcessorManager", "cluster-scope", composerFieldSet.DagProcessorManagerID, composerFieldSet.Component))
		mappedToTimeline = true
	}

	if composerFieldSet.TriggererID != "" {
		cs.AddEvent(resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "AirflowTriggerer", "cluster-scope", composerFieldSet.TriggererID, composerFieldSet.Component))
		mappedToTimeline = true
	}

	if composerFieldSet.WebserverID != "" {
		cs.AddEvent(resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "AirflowWebserver", "cluster-scope", composerFieldSet.WebserverID, composerFieldSet.Component))
		mappedToTimeline = true
	}

	if !mappedToTimeline {
		if composerFieldSet.Subservice != "" {
			componentName = composerFieldSet.Subservice
		}
		rp := resourcepath.NameLayerGeneralItem("Apache Airflow", "Airflow components", "cluster-scope", componentName)
		cs.AddEvent(rp)
	}

	return struct{}{}, nil
}
