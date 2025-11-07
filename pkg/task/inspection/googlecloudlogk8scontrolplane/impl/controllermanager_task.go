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

package googlecloudlogk8scontrolplane_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
)

func kindToKLogFieldPair(apiVersion string, kind string, klogField string, isNamespaced bool) *googlecloudlogk8scontrolplane_contract.KindToKLogFieldPairData {
	return &googlecloudlogk8scontrolplane_contract.KindToKLogFieldPairData{
		APIVersion:   apiVersion,
		KindName:     kind,
		KLogField:    klogField,
		IsNamespaced: isNamespaced,
	}
}

var ControllerManagerFilterTask = inspectiontaskbase.NewLogFilterTask(
	googlecloudlogk8scontrolplane_contract.ControllerManagerLogFilterTaskID,
	googlecloudlogk8scontrolplane_contract.CommonFieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		componentFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{})
		if err != nil {
			return false
		}
		return componentFieldSet.ComponentParserType() == googlecloudlogk8scontrolplane_contract.ComponentParserTypeControllerManager
	},
)

var ControllerManagerLogFieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogk8scontrolplane_contract.ControllerManagerLogFieldSetReaderTaskID,
	googlecloudlogk8scontrolplane_contract.ControllerManagerLogFilterTaskID.Ref(),
	[]log.FieldSetReader{
		&googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSetReader{},
		&googlecloudlogk8scontrolplane_contract.K8sControllerManagerComponentFieldSetReader{
			WellKnownSourceLocationToControllerMap: map[string]string{
				"namespace_controller.go":      "namespace-controller",
				"resource_quota_controller.go": "resourcequota-controller",
				"requestheader_controller.go":  "requestheader-controller",
				"pv_protection_controller.go":  "persistentvolume-protection-controller",
			},
			WellKnownKindToKLogFieldPairs: []*googlecloudlogk8scontrolplane_contract.KindToKLogFieldPairData{
				kindToKLogFieldPair("apps/v1", "deployment", "deployment", true),
				kindToKLogFieldPair("apps/v1", "replicaset", "replicaSet", true),
				kindToKLogFieldPair("apps/v1", "statefulset", "statefulSet", true),
				kindToKLogFieldPair("apps/v1", "daemonset", "daemonSet", true),
				kindToKLogFieldPair("batch/v1", "cronjob", "cronjob", true),
				kindToKLogFieldPair("batch/v1", "job", "job", true),
				kindToKLogFieldPair("policy/v1", "poddisruptionbudget", "podDisruptionBudget", true),
				kindToKLogFieldPair("certificates.k8s.io/v1", "certificatesigningrequest", "csr", false),
				kindToKLogFieldPair("core/v1", "persistentvolumeclaim", "PVC", true),
				kindToKLogFieldPair("core/v1", "persistentvolume", "volumeName", false),
				kindToKLogFieldPair("core/v1", "service", "service", true),
				kindToKLogFieldPair("core/v1", "node", "node", false),
				kindToKLogFieldPair("core/v1", "pod", "pod", true),
				kindToKLogFieldPair("core/v1", "namespace", "namespace", false),
			},
		},
	},
)

var ControllerManagerGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogk8scontrolplane_contract.ControllerManagerLogGrouperTaskID,
	googlecloudlogk8scontrolplane_contract.ControllerManagerLogFieldSetReaderTaskID.Ref(),
	func(ctx context.Context, log *log.Log) string {
		return "" // No grouping needed
	},
)

var ControllerManagerHistoryModifierTask = inspectiontaskbase.NewHistoryModifierTask[struct{}](googlecloudlogk8scontrolplane_contract.ControllerManagerHistoryModifierTaskID, &controllerManagerHistoryModifierTaskSetting{})

type controllerManagerHistoryModifierTaskSetting struct {
}

// Dependencies implements inspectiontaskbase.HistoryModifer.
func (o *controllerManagerHistoryModifierTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.HistoryModifer.
func (o *controllerManagerHistoryModifierTaskSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8scontrolplane_contract.ControllerManagerLogGrouperTaskID.Ref()
}

// LogSerializerTask implements inspectiontaskbase.HistoryModifer.
func (o *controllerManagerHistoryModifierTaskSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8scontrolplane_contract.LogSerializerTaskID.Ref()
}

// ModifyChangeSetFromLog implements inspectiontaskbase.HistoryModifer.
func (o *controllerManagerHistoryModifierTaskSetting) ModifyChangeSetFromLog(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	componentFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{})
	if err != nil {
		return struct{}{}, err
	}
	commonMainMessage, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet{})
	if err != nil {
		return struct{}{}, err
	}
	controllerManagerFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControllerManagerComponentFieldSet{})
	if err != nil {
		return struct{}{}, err
	}

	cs.SetLogSummary(commonMainMessage.Message)
	cs.AddEvent(controllerManagerFieldSet.ControlPlaneResourcePath(componentFieldSet.ClusterName))
	for _, resourcePath := range controllerManagerFieldSet.AssociatedResources {
		cs.AddEvent(resourcePath)
	}
	return struct{}{}, nil
}
