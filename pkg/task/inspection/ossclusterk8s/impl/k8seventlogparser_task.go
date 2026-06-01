// Copyright 2026 Google LLC
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

package ossclusterk8s_impl

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	ossclusterk8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/ossclusterk8s/contract"
)

// OSSK8sEventFieldSetReadTask reads event field sets in parallel.
var OSSK8sEventFieldSetReadTask = inspectiontaskbase.NewFieldSetReadTask(
	ossclusterk8s_contract.OSSK8sEventFieldSetReadTaskID,
	ossclusterk8s_contract.EventAuditLogFilterTaskID.Ref(),
	[]log.FieldSetReader{
		&ossclusterk8s_contract.OSSK8sEventFieldSetReader{},
		&ossclusterk8s_contract.OSSK8sAuditLogCommonFieldSetReader{},
	},
)

// OSSK8sEventLogIngester handles event log metadata ingestion.
type OSSK8sEventLogIngester struct{}

// RawLogTask returns the task reference providing raw event logs.
func (i *OSSK8sEventLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return ossclusterk8s_contract.OSSK8sEventFieldSetReadTaskID.Ref()
}

// Dependencies returns additional dependencies of the ingester.
func (i *OSSK8sEventLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog populates metadata into the LogChangeSet.
func (i *OSSK8sEventLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(commonlogk8saudit_contract.LogTypeEvent)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	eventFS, err := log.GetFieldSet(l, &ossclusterk8s_contract.OSSK8sEventFieldSet{})
	if err != nil {
		return nil, fmt.Errorf("failed to get OSS k8s event fieldset: %w", err)
	}
	cs.SetSummary(fmt.Sprintf("【%s】%s", eventFS.Reason, eventFS.Message))
	cs.SetSeverity(inspectioncore_contract.SeverityUnknown)

	return cs, nil
}

var _ inspectiontaskbase.LogIngesterV2 = (*OSSK8sEventLogIngester)(nil)

// OSSK8sEventLogIngesterTask is the V2 log ingester task.
var OSSK8sEventLogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
	ossclusterk8s_contract.OSSK8sEventLogIngesterTaskID,
	&OSSK8sEventLogIngester{},
)

// OSSK8sEventLogGrouperTask groups event logs by their resource path.
var OSSK8sEventLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	ossclusterk8s_contract.OSSK8sEventLogGrouperTaskID,
	ossclusterk8s_contract.OSSK8sEventFieldSetReadTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		event, err := log.GetFieldSet(l, &ossclusterk8s_contract.OSSK8sEventFieldSet{})
		if err != nil {
			return "unknown"
		}
		return event.ResourceIdentity().String()
	},
)

// OSSK8sEventTimelineMapper maps grouped events to timeline paths.
type OSSK8sEventTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask returns the prerequisite log ingester task.
func (m *OSSK8sEventTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return ossclusterk8s_contract.OSSK8sEventLogIngesterTaskID.Ref()
}

// Dependencies returns additional mapper dependencies.
func (m *OSSK8sEventTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask returns the task providing grouped logs.
func (m *OSSK8sEventTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return ossclusterk8s_contract.OSSK8sEventLogGrouperTaskID.Ref()
}

// ProcessLogByGroup maps a single event log to its resource timeline.
func (m *OSSK8sEventTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	event, err := log.GetFieldSet(l, &ossclusterk8s_contract.OSSK8sEventFieldSet{})
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to get OSS k8s event fieldset: %w", err)
	}

	targetPath := commonlogk8saudit_contract.MustResourceTimeline(ctx, "cluster", event.ResourceIdentity())

	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(targetPath)

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*OSSK8sEventTimelineMapper)(nil)

// OSSK8sEventLogToTimelineMapperTask is the V2 log to timeline mapper task.
var OSSK8sEventLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(
	ossclusterk8s_contract.OSSK8sEventLogToTimelineMapperTaskID,
	&OSSK8sEventTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabelV2(
		"OSS Kubernetes Event logs from JSONL audit log",
		"The event log parser for OSS kubernetes from the audit log",
		2000,
		true,
	),
)
