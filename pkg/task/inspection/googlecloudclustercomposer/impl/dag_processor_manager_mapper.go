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
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
)

var AirflowDagProcessorManagerLogSorterTask = inspectiontaskbase.NewLogSorterByTimeTask(
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogSorterTaskID,
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogFilterTaskID.Ref(),
)

var AirflowDagProcessorManagerLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogGrouperTaskID,
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogSorterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		fs, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerFieldSet{})
		if err != nil {
			return ""
		}
		if fs.SchedulerID != "" {
			return fs.SchedulerID
		}
		return fs.DagProcessorManagerID
	},
)

var AirflowDagProcessorManagerLogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogIngesterTaskID,
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogFilterTaskID.Ref(),
)

var AirflowDagProcessorManagerLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[*DagProcessorState](
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogToTimelineMapperTaskID,
	&airflowDagProcessorManagerLogToTimelineMapperSetting{
		targetLogType: enum.LogTypeComposerEnvironment,
		dagFilePath:   "/home/airflow/gcs/dags", // It seems dagFilePath was passed historically, fixing it to default.
	},
)

const (
	dagProcessorManagerColumnFilePath    = "File Path"
	dagProcessorManagerColumnPID         = "PID"
	dagProcessorManagerColumnRuntime     = "Runtime"
	dagProcessorManagerColumnNumDags     = "# DAGs"
	dagProcessorManagerColumnNumErrors   = "# Errors"
	dagProcessorManagerColumnLastRuntime = "Last Runtime"
	dagProcessorManagerColumnLastRun     = "Last Run"
)

// DagProcessorState retains the parsing state using TabulateReader.
type DagProcessorState struct {
	Reader *logutil.TabulateReader
}

type airflowDagProcessorManagerLogToTimelineMapperSetting struct {
	targetLogType enum.LogType
	dagFilePath   string
}

func (c *airflowDagProcessorManagerLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogGrouperTaskID.Ref()
}

func (c *airflowDagProcessorManagerLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogIngesterTaskID.Ref()
}

func (c *airflowDagProcessorManagerLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return nil
}

func (c *airflowDagProcessorManagerLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData *DagProcessorState) (*DagProcessorState, error) {
	commonField, _ := log.GetFieldSet(l, &log.CommonFieldSet{})
	mainMessage, err := log.GetFieldSet(l, &log.MainMessageFieldSet{})
	if err != nil {
		return prevGroupData, nil
	}
	dpmField, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerFieldSet{})
	parserID := "unknown-parser"
	if err == nil {
		if dpmField.SchedulerID != "" {
			cs.AddEvent(resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "AirflowScheduler", "cluster-scope", dpmField.SchedulerID, "airflow-dag-processor-manager"))
			parserID = dpmField.SchedulerID
		} else if dpmField.DagProcessorManagerID != "" {
			cs.AddEvent(resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "AirflowDagProcessorManager", "cluster-scope", dpmField.DagProcessorManagerID, "airflow-dag-processor-manager"))
			parserID = dpmField.DagProcessorManagerID
		}
	}

	rawLog := mainMessage.MainMessage
	rawLog = strings.TrimPrefix(rawLog, "DAG_PROCESSOR_MANAGER_LOG:")
	rawLog = strings.TrimSpace(rawLog)

	if prevGroupData == nil {
		prevGroupData = &DagProcessorState{
			Reader: logutil.NewTabulateReader(),
		}
	}
	if strings.Contains(rawLog, "==========") {
		prevGroupData.Reader.Reset()
	}
	res, err := prevGroupData.Reader.ParseLine(rawLog)
	if err != nil {
		cs.SetLogSummary(rawLog)
		// Log format exception or end of table reached
		return prevGroupData, nil
	}

	if res.Type != logutil.TabulateLineTypeBody {
		cs.SetLogSummary(rawLog)
		return prevGroupData, nil
	}

	condition := enum.RevisionStateConditionTrue
	if res.Values[dagProcessorManagerColumnNumErrors] != "0" {
		cs.SetLogSeverity(enum.SeverityError)
		condition = enum.RevisionStateConditionFalse
	}

	cs.AddRevision(resourcepath.NameLayerGeneralItem("Apache Airflow", "Dag File Processor Stats", parserID, res.Values[dagProcessorManagerColumnFilePath]), &history.StagingResourceRevision{
		Verb:       enum.RevisionVerbComposerTaskInstanceStats,
		State:      condition,
		Requestor:  "dag-processor-manager",
		ChangeTime: commonField.Timestamp,
	})
	cs.SetLogSummary(fmt.Sprintf("File Path: %s PID: %s #DAGs: %s #Errors: %s", res.Values[dagProcessorManagerColumnFilePath], res.Values[dagProcessorManagerColumnPID], res.Values[dagProcessorManagerColumnNumDags], res.Values[dagProcessorManagerColumnNumErrors]))

	return prevGroupData, nil
}
