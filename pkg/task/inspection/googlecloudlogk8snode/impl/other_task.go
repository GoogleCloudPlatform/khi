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
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
)

// OtherLogFilterTask filters only the components logs that do not match kubelet or containerd.
var OtherLogFilterTask = newParserTypeFilterTask(googlecloudlogk8snode_contract.OtherLogFilterTaskID, googlecloudlogk8snode_contract.CommonFieldsetReaderTaskID.Ref(), googlecloudlogk8snode_contract.Other)

// OtherLogGroupTask groups other logs by node and component.
var OtherLogGroupTask = newNodeAndComponentNameGrouperTask(googlecloudlogk8snode_contract.OtherLogGroupTaskID, googlecloudlogk8snode_contract.OtherLogFilterTaskID.Ref())

type otherNodeLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.StatelessMapperBase
	StartingMessagesByComponent    map[string]string
	TerminatingMessagesByComponent map[string]string
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapperV2.
func (o *otherNodeLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
	}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapperV2.
func (o *otherNodeLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8snode_contract.OtherLogGroupTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapperV2.
func (o *otherNodeLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8snode_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapperV2.
func (o *otherNodeLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())
	componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})

	cs := khifilev6.NewTimelineChangeSet(l)

	nodeTimelinePath := MustK8sNodeTimeline(ctx, clusterName, componentFieldSet.NodeName)
	componentTimelinePath := googlecloudlogk8snode_contract.MustNodeComponentTimeline(ctx, nodeTimelinePath, componentFieldSet.Component)

	var startingMessage string
	var terminatingMessage string
	if msg, found := o.StartingMessagesByComponent[componentFieldSet.Component]; found {
		startingMessage = msg
	}
	if msg, found := o.TerminatingMessagesByComponent[componentFieldSet.Component]; found {
		terminatingMessage = msg
	}
	checkStartingAndTerminationLog(ctx, cs, l, startingMessage, terminatingMessage, componentTimelinePath)

	cs.AddEvent(componentTimelinePath)

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*otherNodeLogLogToTimelineMapperSetting)(nil)

// OtherLogLogToTimelineMapperTask registers the mapper for other node component logs.
var OtherLogLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(googlecloudlogk8snode_contract.OtherLogLogToTimelineMapperTaskID, &otherNodeLogLogToTimelineMapperSetting{
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
