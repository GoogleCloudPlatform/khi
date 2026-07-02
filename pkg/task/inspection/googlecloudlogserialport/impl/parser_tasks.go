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

package googlecloudlogserialport_impl

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogserialport_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogserialport/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReadTask is the task to run GCESerialPortLogFieldSetReader on logs to parse serial port logs.
var FieldSetReadTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudlogserialport_contract.FieldSetReadTaskID,
	googlecloudlogserialport_contract.LogQueryTaskID.Ref(),
	[]log.FieldSetReader{
		&googlecloudlogserialport_contract.GCESerialPortLogFieldSetReader{},
		&googlecloudcommon_contract.GCPDefaultSeverityFieldSetReader{},
	},
)

// LogFilterTask removes logs with empty message.
// This message is mostly just contained escape sequences and stripped by ANSIEscapeSequenceStripper.
var LogFilterTask = inspectiontaskbase.NewLogFilterTask(
	googlecloudlogserialport_contract.LogFilterTaskID,
	googlecloudlogserialport_contract.FieldSetReadTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		return log.MustGetFieldSet(l, &googlecloudlogserialport_contract.GCESerialPortLogFieldSet{}).Message != ""
	},
)

// serialPortLogIngester implements the LogIngester interface.
type serialPortLogIngester struct{}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *serialPortLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogserialport_contract.LogFilterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *serialPortLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog parses raw log entry and manually populates the LogChangeSet.
func (i *serialPortLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(googlecloudlogserialport_contract.LogTypeSerialPort)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severityFS.Severity)
	}

	if serialFS, err := log.GetFieldSet(l, &googlecloudlogserialport_contract.GCESerialPortLogFieldSet{}); err == nil {
		cs.SetSummary(serialFS.Message)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngester = (*serialPortLogIngester)(nil)

// LogIngesterTask is the log ingester task.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	googlecloudlogserialport_contract.LogIngesterTaskID,
	&serialPortLogIngester{},
)

// LogGrouperTask is the grouper task for GCE serial port logs.
// It groups logs by the node name and port name.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogserialport_contract.LogGrouperTaskID,
	googlecloudlogserialport_contract.LogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		serialFS := log.MustGetFieldSet(l, &googlecloudlogserialport_contract.GCESerialPortLogFieldSet{})
		return fmt.Sprintf("%s#%s", serialFS.NodeName, serialFS.Port)
	},
)

// serialportLogToTimelineMapper maps logs to hierarchical node serial port timelines.
type serialportLogToTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask implements the LogToTimelineMapper interface.
func (s *serialportLogToTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogserialport_contract.LogIngesterTaskID.Ref()
}

// GroupedLogTask implements the LogToTimelineMapper interface.
func (s *serialportLogToTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogserialport_contract.LogGrouperTaskID.Ref()
}

// Dependencies implements the LogToTimelineMapper interface.
func (s *serialportLogToTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(),
	}
}

// ProcessLogByGroup processes each log inside the group and stages the event on the timeline.
func (s *serialportLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	serialportFieldSet, err := log.GetFieldSet(l, &googlecloudlogserialport_contract.GCESerialPortLogFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}

	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref())

	targetPath := googlecloudlogserialport_contract.MustSerialPortTimeline(
		ctx,
		clusterIdentity.ClusterName,
		serialportFieldSet.NodeName,
		serialportFieldSet.Port,
	)

	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(targetPath)

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*serialportLogToTimelineMapper)(nil)

// LogToTimelineMapperTask is the timeline mapper task.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	googlecloudlogserialport_contract.LogToTimelineMapperTaskID,
	&serialportLogToTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabel(
		"GCE Node Serial Port Logs",
		`Gather serial port logs from GCE instances to troubleshoot VM bootstrapping and OS initialization issues.`,
		10000,
		false,
	),
)
