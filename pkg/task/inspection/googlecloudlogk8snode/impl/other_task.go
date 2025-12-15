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

package googlecloudlogk8snode_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
)

var OtherLogFilterTask = newParserTypeFilterTask(googlecloudlogk8snode_contract.OtherLogFilterTaskID, googlecloudlogk8snode_contract.CommonFieldsetReaderTaskID.Ref(), googlecloudlogk8snode_contract.Other)

var OtherLogGroupTask = newNodeAndComponentNameGrouperTask(googlecloudlogk8snode_contract.OtherLogGroupTaskID, googlecloudlogk8snode_contract.OtherLogFilterTaskID.Ref())

var OtherLogLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](googlecloudlogk8snode_contract.OtherLogLogToTimelineMapperTaskID, &otherNodeLogLogToTimelineMapperSetting{
	StartingMessagesByComponent: map[string]string{
		"dockerd":             "Starting up",
		"configure.sh":        "Start to install kubernetes files",
		"configure-helper.sh": "Start to configure instance for kubernetes",
	},
	TerminatingMessagesByComponent: map[string]string{
		"dockerd":             "Daemon shutdown complete",
		"configure.sh":        "Done for installing kubernetes files",
		"configure-helper.sh": "Done for the configuration for kubernetes",
	},
})

type otherNodeLogLogToTimelineMapperSetting struct {
	StartingMessagesByComponent    map[string]string
	TerminatingMessagesByComponent map[string]string
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (o *otherNodeLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *otherNodeLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8snode_contract.OtherLogGroupTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *otherNodeLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8snode_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (o *otherNodeLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})

	var startingMessage string
	var terminatingMessage string
	if msg, found := o.StartingMessagesByComponent[componentFieldSet.Component]; found {
		startingMessage = msg
	}
	if msg, found := o.TerminatingMessagesByComponent[componentFieldSet.Component]; found {
		terminatingMessage = msg
	}
	checkStartingAndTerminationLog(cs, l, startingMessage, terminatingMessage)

	cs.AddEvent(componentFieldSet.ResourcePath())

	severity, err := componentFieldSet.Message.Severity()
	if err == nil {
		cs.SetLogSeverity(severity)
	}

	summary, err := parseDefaultSummary(componentFieldSet.Message)
	if summary == "" || err != nil {
		summary, _ = componentFieldSet.Message.MainMessage()
	}
	cs.SetLogSummary(summary)
	return struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*otherNodeLogLogToTimelineMapperSetting)(nil)
