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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
)

var OtherLogFilterTask = newParserTypeFilterTask(googlecloudlogk8snode_contract.OtherLogFilterTaskID, googlecloudlogk8snode_contract.CommonFieldsetReaderTaskID.Ref(), googlecloudlogk8snode_contract.Other)

var OtherLogGroupTask = newNodeAndComponentNameGrouperTask(googlecloudlogk8snode_contract.OtherLogGroupTaskID, googlecloudlogk8snode_contract.OtherLogFilterTaskID.Ref())

var OtherLogHistoryModifierTask = inspectiontaskbase.NewHistoryModifierTask[struct{}](googlecloudlogk8snode_contract.OtherLogHistoryModifierTaskID, &otherNodeLogHistoryModifierSetting{
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

type otherNodeLogHistoryModifierSetting struct {
	StartingMessagesByComponent    map[string]string
	TerminatingMessagesByComponent map[string]string
}

// Dependencies implements inspectiontaskbase.HistoryModifer.
func (o *otherNodeLogHistoryModifierSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.HistoryModifer.
func (o *otherNodeLogHistoryModifierSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8snode_contract.OtherLogGroupTaskID.Ref()
}

// LogSerializerTask implements inspectiontaskbase.HistoryModifer.
func (o *otherNodeLogHistoryModifierSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8snode_contract.LogSerializerTaskID.Ref()
}

// ModifyChangeSetFromLog implements inspectiontaskbase.HistoryModifer.
func (o *otherNodeLogHistoryModifierSetting) ModifyChangeSetFromLog(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
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

	severity := logutil.ExractKLogSeverity(componentFieldSet.Message)
	cs.SetLogSeverity(severity)
	summary, err := parseDefaultSummary(componentFieldSet.Message)
	if summary == "" || err != nil {
		summary = componentFieldSet.Message
	}
	cs.SetLogSummary(summary)
	return struct{}{}, nil
}

var _ inspectiontaskbase.HistoryModifer[struct{}] = (*otherNodeLogHistoryModifierSetting)(nil)
