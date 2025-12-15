// Copyright 2024 Google LLC
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

package googlecloudlogk8scontainer_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogk8scontainer_contract.FieldSetReaderTaskID, googlecloudlogk8scontainer_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSetReader{},
})

var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(googlecloudlogk8scontainer_contract.LogIngesterTaskID, googlecloudlogk8scontainer_contract.ListLogEntriesTaskID.Ref())

var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlogk8scontainer_contract.LogGrouperTaskID, googlecloudlogk8scontainer_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		// container log parser is stateless and it doesn't require grouping to work, but grouping them by its associated instance resource name for better performance to process them in parallel.
		containerFields, err := log.GetFieldSet(l, &googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{})
		if err != nil {
			return "unknown"
		}
		return containerFields.ResourcePath().Path
	})

var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](googlecloudlogk8scontainer_contract.LogToTimelineMapperTaskID, &containerLogLogToTimelineMapperSetting{},
	inspectioncore_contract.FeatureTaskLabel(`Kubernetes container logs`,
		`Gather stdout/stderr logs of containers on the cluster to visualize them on the timeline under an associated Pod. Log volume can be huge when the cluster has many Pods.`,
		enum.LogTypeContainer,
		4000,
		false,
		googlecloudinspectiontypegroup_contract.GCPK8sClusterInspectionTypes...),
)

type containerLogLogToTimelineMapperSetting struct {
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (c *containerLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (c *containerLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8scontainer_contract.LogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (c *containerLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8scontainer_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (c *containerLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	containerFields, err := log.GetFieldSet(l, &googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{})
	if err != nil {
		return struct{}{}, nil
	}

	cs.AddEvent(containerFields.ResourcePath())
	cs.SetLogSummary(containerFields.Message)
	return struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*containerLogLogToTimelineMapperSetting)(nil)
