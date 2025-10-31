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

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogserialport_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogserialport/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReadTask is the task to run GCESerialPortLogFieldSetReader on logs to parse serial port logs.
var FieldSetReadTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogserialport_contract.FieldSetReadTaskID, googlecloudlogserialport_contract.LogQueryTaskID.Ref(), []log.FieldSetReader{
	&googlecloudlogserialport_contract.GCESerialPortLogFieldSetReader{},
})

// LogFilterTask removes logs with empty message.
// This message is mostly just contained escape sequences and stripped by ANSIEscapeSequenceStripper.
var LogFilterTask = inspectiontaskbase.NewLogFilterTask(googlecloudlogserialport_contract.LogFilterTaskID, googlecloudlogserialport_contract.FieldSetReadTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		return log.MustGetFieldSet(l, &googlecloudlogserialport_contract.GCESerialPortLogFieldSet{}).Message != ""
	},
)

// LogSerializerTask is the log serializer task for GCE serial port logs.
// It includes all logs gathered from log list.
var LogSerializerTask = inspectiontaskbase.NewLogSerializerTask(
	googlecloudlogserialport_contract.LogSerializerTaskID,
	googlecloudlogserialport_contract.LogFilterTaskID.Ref(),
)

// LogGrouperTask is the grouper task for GCE serial port logs.
// It groups logs by the node name and port name
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogserialport_contract.LogGrouperTaskID,
	googlecloudlogserialport_contract.LogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		return log.MustGetFieldSet(l, &googlecloudlogserialport_contract.GCESerialPortLogFieldSet{}).GetResourcePath().Path
	},
)

// HistoryModifierTask is the task to modify history to generate events on serial port logs.
var HistoryModifierTask = inspectiontaskbase.NewHistoryModifierTask[struct{}](
	googlecloudlogserialport_contract.HistoryModifierTaskID,
	&serialportHistoryModifier{},
	inspectioncore_contract.FeatureTaskLabel(
		"GCE Node Serialport log",
		`Serialport logs from GCE instances. This helps detailed investigation on VM bootstrapping issue on GCE instance.`,
		enum.LogTypeSerialPort,
		10000, false, googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes...,
	),
)

type serialportHistoryModifier struct {
}

// GroupedLogTask implements inspectiontaskbase.HistoryModifer.
func (s *serialportHistoryModifier) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogserialport_contract.LogGrouperTaskID.Ref()
}

// LogSerializerTask implements inspectiontaskbase.HistoryModifer.
func (s *serialportHistoryModifier) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogserialport_contract.LogSerializerTaskID.Ref()
}

func (s *serialportHistoryModifier) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ModifyChangeSetFromLog implements inspectiontaskbase.HistoryModifer.
func (s *serialportHistoryModifier) ModifyChangeSetFromLog(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	serialportFieldSet := log.MustGetFieldSet(l, &googlecloudlogserialport_contract.GCESerialPortLogFieldSet{})
	cs.AddEvent(serialportFieldSet.GetResourcePath())
	cs.SetLogSummary(serialportFieldSet.Message)
	return struct{}{}, nil
}

var _ inspectiontaskbase.HistoryModifer[struct{}] = (*serialportHistoryModifier)(nil)
