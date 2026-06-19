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
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AirflowDagProcessorManagerLogSorterTask sorts Airflow DAG processor manager logs.
var AirflowDagProcessorManagerLogSorterTask = inspectiontaskbase.NewLogSorterByTimeTask(
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogSorterTaskID,
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogFilterTaskID.Ref(),
)

// AirflowDagProcessorManagerLogGrouperTask groups Airflow DAG processor manager logs.
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

type dagProcessorManagerLogIngester struct {
	inspectiontaskbase.SinglePassGroupedIngesterBase[*DagProcessorState]
}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *dagProcessorManagerLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogSorterTaskID.Ref()
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (i *dagProcessorManagerLogIngester) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogGrouperTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *dagProcessorManagerLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLogByGroup is called for each log entry in a group to customize log metadata.
// It parses tabular log entries and maintains sequence state within the group.
func (i *dagProcessorManagerLogIngester) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData *DagProcessorState) (*khifilev6.LogChangeSet, *DagProcessorState, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, prevGroupData, err
	}
	cs.SetLogType(googlecloudclustercomposer_contract.LogTypeComposerEnvironment)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	// Default severity is Unknown and summary is empty
	cs.SetSeverity(inspectioncore_contract.SeverityUnknown)
	cs.SetSummary("")

	mainMessage, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPMainMessageFieldSet{})
	if err != nil {
		return cs, prevGroupData, nil
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
		cs.SetSummary(rawLog)
		return cs, prevGroupData, nil
	}

	if res.Type != logutil.TabulateLineTypeBody {
		cs.SetSummary(rawLog)
		return cs, prevGroupData, nil
	}

	if res.Values[dagProcessorManagerColumnNumErrors] != "" && res.Values[dagProcessorManagerColumnNumErrors] != "0" {
		cs.SetSeverity(inspectioncore_contract.SeverityError)
	}

	summaryText := fmt.Sprintf("File Path: %s PID: %s #DAGs: %s #Errors: %s", res.Values[dagProcessorManagerColumnFilePath], res.Values[dagProcessorManagerColumnPID], res.Values[dagProcessorManagerColumnNumDags], res.Values[dagProcessorManagerColumnNumErrors])
	cs.SetSummary(summaryText)

	return cs, prevGroupData, nil
}

var _ inspectiontaskbase.GroupedLogIngesterV2[*DagProcessorState] = (*dagProcessorManagerLogIngester)(nil)

// AirflowDagProcessorManagerLogIngesterTask is the task that ingests Airflow DAG processor manager logs.
var AirflowDagProcessorManagerLogIngesterTask = inspectiontaskbase.NewGroupedLogIngesterTaskV2(
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogIngesterTaskID,
	&dagProcessorManagerLogIngester{},
)

type dagProcessorManagerTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*DagProcessorState]
	targetLogType *pb.LogType
	dagFilePath   string
}

// LogIngesterTask returns a reference to the ingester task.
func (m *dagProcessorManagerTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the mapper.
func (m *dagProcessorManagerTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *dagProcessorManagerTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogGrouperTaskID.Ref()
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *dagProcessorManagerTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData *DagProcessorState) (*khifilev6.TimelineChangeSet, *DagProcessorState, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	envPath := googlecloudclustercomposer_contract.MustComposerEnvironmentTimeline(ctx, clusterIdentity.ProjectID, environmentName)

	commonField, _ := log.GetFieldSet(l, &log.CommonFieldSet{})
	mainMessage, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPMainMessageFieldSet{})
	if err != nil {
		return nil, prevGroupData, nil
	}
	dpmField, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerFieldSet{})
	cs := khifilev6.NewTimelineChangeSet(l)
	parserID := "unknown-parser"
	if err == nil {
		if dpmField.SchedulerID != "" {
			cs.AddEvent(googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowScheduler, dpmField.SchedulerID))
			parserID = dpmField.SchedulerID
		} else if dpmField.DagProcessorManagerID != "" {
			cs.AddEvent(googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowDagProcessorManager, dpmField.DagProcessorManagerID))
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
		return cs, prevGroupData, nil
	}

	if res.Type != logutil.TabulateLineTypeBody {
		return cs, prevGroupData, nil
	}

	condition := commonlogk8saudit_contract.RevisionStateConditionTrue
	if res.Values[dagProcessorManagerColumnNumErrors] != "" && res.Values[dagProcessorManagerColumnNumErrors] != "0" {
		condition = commonlogk8saudit_contract.RevisionStateConditionFalse
	}

	timelinePath := googlecloudclustercomposer_contract.MustAirflowDAGProcessorManagerInstanceTimeline(ctx, envPath, res.Values[dagProcessorManagerColumnFilePath], parserID)

	cs.AddRevision(timelinePath, &khifilev6.StagingRevision{
		ChangedTime: commonField.Timestamp,
		Principal:   "dag-processor-manager",
		VerbType:    googlecloudclustercomposer_contract.VerbComposerTaskInstanceStats,
		StateType:   condition,
	})

	return cs, prevGroupData, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[*DagProcessorState] = (*dagProcessorManagerTimelineMapper)(nil)

// AirflowDagProcessorManagerLogToTimelineMapperTask is the task that maps Airflow DAG processor manager logs to timeline events.
var AirflowDagProcessorManagerLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogToTimelineMapperTaskID,
	&dagProcessorManagerTimelineMapper{
		targetLogType: googlecloudclustercomposer_contract.LogTypeComposerEnvironment,
		dagFilePath:   "/home/airflow/gcs/dags",
	},
)
