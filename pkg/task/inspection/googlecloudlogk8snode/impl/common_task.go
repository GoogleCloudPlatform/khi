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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// LogIngesterTask serializes logs to history for timeline mappers to associate event or revisions in later tasks.
// No node logs are discarded, thus this LogIngesterTask simply receives logs from the ListLogEntriesTask.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(googlecloudlogk8snode_contract.LogIngesterTaskID, googlecloudlogk8snode_contract.ListLogEntriesTaskID.Ref())

var CommonFieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogk8snode_contract.CommonFieldsetReaderTaskID, googlecloudlogk8snode_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSetReader{
		StructuredLogParser: logutil.NewMultiTextLogParser(
			logutil.NewKLogTextParser(true),
			logutil.NewLogfmtTextParser(),
			&logutil.FallbackRawTextLogParser{},
		),
	},
})

var TailTask = inspectiontaskbase.NewInspectionTask(googlecloudlogk8snode_contract.TailTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudlogk8snode_contract.ContainerdLogLogToTimelineMapperTaskID.Ref(),
		googlecloudlogk8snode_contract.KubeletLogLogToTimelineMapperTaskID.Ref(),
		googlecloudlogk8snode_contract.OtherLogLogToTimelineMapperTaskID.Ref(),

		googlecloudlogk8snode_contract.ContainerIDDiscoveryTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	},
	inspectioncore_contract.FeatureTaskLabel(
		"Kubernetes Node Logs",
		"Gather node components(e.g docker/container) logs. Log volume can be huge when the cluster has many nodes.",
		enum.LogTypeControlPlaneComponent,
		3000,
		false,
		googlecloudinspectiontypegroup_contract.GCPK8sClusterInspectionTypes...,
	),
)

// newParserTypeFilterTask creates a new filter task that filters only for specific parserType.
func newParserTypeFilterTask(taskid taskid.TaskImplementationID[[]*log.Log], logSource taskid.TaskReference[[]*log.Log], parserType googlecloudlogk8snode_contract.K8sNodeParserType) coretask.Task[[]*log.Log] {
	return inspectiontaskbase.NewLogFilterTask(
		taskid,
		logSource,
		func(ctx context.Context, l *log.Log) bool {
			componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
			return componentFieldSet.ParserType() == parserType
		},
	)
}

// newNodeAndComponentNameGrouperTask creates a new grouper task with grouping by node name and component name.
func newNodeAndComponentNameGrouperTask(taskid taskid.TaskImplementationID[inspectiontaskbase.LogGroupMap], logSource taskid.TaskReference[[]*log.Log]) coretask.Task[inspectiontaskbase.LogGroupMap] {
	return inspectiontaskbase.NewLogGrouperTask(taskid, logSource, func(ctx context.Context, l *log.Log) string {
		componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
		return fmt.Sprintf("%s-%s", componentFieldSet.NodeName, componentFieldSet.Component)
	})
}
