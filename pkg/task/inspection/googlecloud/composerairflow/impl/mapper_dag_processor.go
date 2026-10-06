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
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerairflow"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// airflowDagProcessorManagerLogGrouperTask groups Airflow DAG processor manager logs.
var airflowDagProcessorManagerLogGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	composerairflow.AirflowDagProcessorManagerLogGrouperTaskID,
	composerairflow.AirflowDagProcessorManagerLogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			fs, err := composerairflow.ExtractComposer(l.NodeReader)
			if err != nil {
				return ""
			}
			if fs.SchedulerID != "" {
				return fs.SchedulerID
			}
			return fs.DagProcessorManagerID
		}
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

// ProcessLogByGroup is called for each log entry in a group to customize log metadata.
// It parses tabular log entries and maintains sequence state within the group.
func (i *dagProcessorManagerLogIngester) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData *DagProcessorState) (*khifilev6.LogChangeSet, *DagProcessorState, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, prevGroupData, err
	}
	cs.SetLogType(composerairflow.LogTypeManagedAirflowEnvironment)
	cs.SetTimestamp(l.Timestamp)

	// Default severity is Unknown and summary is empty
	cs.SetSeverity(inspectioncore.SeverityUnknown)
	cs.SetSummary("")

	rawLog, err := gcpcommon.ExtractGCPMainMessage(l.NodeReader)
	if err != nil || rawLog == "" {
		return cs, prevGroupData, nil
	}

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
		cs.SetSeverity(inspectioncore.SeverityError)
	}

	summaryText := fmt.Sprintf("File Path: %s PID: %s #DAGs: %s #Errors: %s", res.Values[dagProcessorManagerColumnFilePath], res.Values[dagProcessorManagerColumnPID], res.Values[dagProcessorManagerColumnNumDags], res.Values[dagProcessorManagerColumnNumErrors])
	cs.SetSummary(summaryText)

	return cs, prevGroupData, nil
}

var _ inspectiontaskbase.GroupedLogIngester[*DagProcessorState] = (*dagProcessorManagerLogIngester)(nil)

// airflowDagProcessorManagerLogIngesterTask is the task that ingests Airflow DAG processor manager logs.
var airflowDagProcessorManagerLogIngesterTask = inspectiontaskbase.DefineGroupedLogIngesterTask(
	composerairflow.AirflowDagProcessorManagerLogIngesterTaskID,
	composerairflow.AirflowDagProcessorManagerLogGrouperTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.GroupedLogIngester[*DagProcessorState] {
		return &dagProcessorManagerLogIngester{}
	},
)

type dagProcessorManagerTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*DagProcessorState]
	environmentName coretask.Input[string]
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *dagProcessorManagerTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData *DagProcessorState) (*khifilev6.TimelineChangeSet, *DagProcessorState, error) {
	cs, state := mapDagProcessorManagerLog(ctx, l, prevGroupData, m.environmentName.Get(ctx))
	return cs, state, nil
}

var _ inspectiontaskbase.TimelineMapper[*DagProcessorState] = (*dagProcessorManagerTimelineMapper)(nil)

// mapDagProcessorManagerLog maps an Airflow DAG processor manager log to timeline changes for the given environment.
func mapDagProcessorManagerLog(ctx context.Context, l *log.Log, prevGroupData *DagProcessorState, environmentName string) (*khifilev6.TimelineChangeSet, *DagProcessorState) {
	envPath := composerairflow.MustAirflowTimeline(ctx, environmentName)

	rawLog, err := gcpcommon.ExtractGCPMainMessage(l.NodeReader)
	if err != nil || rawLog == "" {
		return nil, prevGroupData
	}
	dpmField, err := composerairflow.ExtractComposer(l.NodeReader)
	cs := khifilev6.NewTimelineChangeSet(l)
	parserID := "unknown-parser"
	if err == nil {
		if dpmField.SchedulerID != "" {
			cs.AddEvent(composerairflow.MustAirflowComponentTimeline(ctx, envPath, dpmField.SchedulerID))
			parserID = dpmField.SchedulerID
		} else if dpmField.DagProcessorManagerID != "" {
			cs.AddEvent(composerairflow.MustAirflowComponentTimeline(ctx, envPath, dpmField.DagProcessorManagerID))
			parserID = dpmField.DagProcessorManagerID
		}
	}

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
		return cs, prevGroupData
	}

	if res.Type != logutil.TabulateLineTypeBody {
		return cs, prevGroupData
	}

	condition := composerairflow.RevisionStateComposerDagProcessorNoError
	if res.Values[dagProcessorManagerColumnNumErrors] != "" && res.Values[dagProcessorManagerColumnNumErrors] != "0" {
		condition = composerairflow.RevisionStateComposerDagProcessorHasErrors
	}

	timelinePath := composerairflow.MustAirflowDAGProcessorManagerInstanceTimeline(ctx, envPath, res.Values[dagProcessorManagerColumnFilePath], parserID)

	cs.AddRevision(timelinePath, &khifilev6.StagingRevision{
		ChangedTime: l.Timestamp,
		Principal:   "dag-processor-manager",
		VerbType:    composerairflow.VerbComposerTaskInstanceStats,
		StateType:   condition,
	})

	return cs, prevGroupData
}

// airflowDagProcessorManagerLogToTimelineMapperTask is the task that maps Airflow DAG processor manager logs to timeline events.
var airflowDagProcessorManagerLogToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	composerairflow.AirflowDagProcessorManagerLogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: composerairflow.AirflowDagProcessorManagerLogIngesterTaskID.Ref(),
		GroupedLogs: composerairflow.AirflowDagProcessorManagerLogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[*DagProcessorState] {
		return &dagProcessorManagerTimelineMapper{
			environmentName: coretask.Use(b, composercluster.InputComposerEnvironmentNameTaskID.Ref()),
		}
	},
)
