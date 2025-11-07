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

package googlecloudlogk8sevent_impl

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogk8sevent_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8sevent/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogk8sevent_contract.FieldSetReaderTaskID, googlecloudlogk8sevent_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudlogk8sevent_contract.GCPKubernetesEventFieldSetReader{},
})

var LogSerializerTask = inspectiontaskbase.NewLogSerializerTask(googlecloudlogk8sevent_contract.LogSerializerTaskID, googlecloudlogk8sevent_contract.ListLogEntriesTaskID.Ref())

var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlogk8sevent_contract.LogGrouperTaskID, googlecloudlogk8sevent_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		event, err := log.GetFieldSet(l, &googlecloudlogk8sevent_contract.KubernetesEventFieldSet{})
		if err != nil {
			return "unknown"
		}
		return event.ResourcePath().Path
	},
)

var HistoryModifierTask = inspectiontaskbase.NewHistoryModifierTask[struct{}](googlecloudlogk8sevent_contract.HistoryModifierTaskID, &KubernetesEventHistoryModifierSetting{}, inspectioncore_contract.FeatureTaskLabel(
	"Kubernetes Event Logs",
	"Gather kubernetes event logs and visualize these on the associated resource timeline.",
	enum.LogTypeEvent,
	2000,
	true,
	googlecloudinspectiontypegroup_contract.GCPK8sClusterInspectionTypes...,
))

type KubernetesEventHistoryModifierSetting struct {
}

// Dependencies implements inspectiontaskbase.HistoryModifer.
func (k *KubernetesEventHistoryModifierSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.HistoryModifer.
func (k *KubernetesEventHistoryModifierSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8sevent_contract.LogGrouperTaskID.Ref()
}

// LogSerializerTask implements inspectiontaskbase.HistoryModifer.
func (k *KubernetesEventHistoryModifierSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8sevent_contract.LogSerializerTaskID.Ref()
}

// ModifyChangeSetFromLog implements inspectiontaskbase.HistoryModifer.
func (k *KubernetesEventHistoryModifierSetting) ModifyChangeSetFromLog(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	event, err := log.GetFieldSet(l, &googlecloudlogk8sevent_contract.KubernetesEventFieldSet{})
	if err != nil {
		return struct{}{}, fmt.Errorf("failed to get kubernetes event fieldset: %w", err)
	}

	cs.AddEvent(event.ResourcePath())
	cs.SetLogSummary(fmt.Sprintf("【%s】%s", event.Reason, event.Message))
	return struct{}{}, nil
}

var _ inspectiontaskbase.HistoryModifer[struct{}] = (*KubernetesEventHistoryModifierSetting)(nil)
