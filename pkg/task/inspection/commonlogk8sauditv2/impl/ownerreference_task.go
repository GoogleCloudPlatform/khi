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

package commonlogk8sauditv2_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

// ResourceOwnerReferenceModifierTask is the task to modify resource owner reference.
var ResourceOwnerReferenceModifierTask = commonlogk8sauditv2_contract.NewManifestHistoryModifier[struct{}](&resourceOwnerReferenceModifierTaskSetting{
	nonNamespacedOwnerTypes: map[string]struct{}{
		"core/v1#node": {},
	},
})

type resourceOwnerReferenceModifierTaskSetting struct {
	// nonNamespacedOwnerTypes is the set of owner types that are not namespaced.
	nonNamespacedOwnerTypes map[string]struct{}
}

// Process implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (r *resourceOwnerReferenceModifierTaskSetting) Process(ctx context.Context, passIndex int, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	if event.EventTargetBodyReader == nil {
		return struct{}{}, nil
	}
	ownerReferencesReaders, err := event.EventTargetBodyReader.GetReader("metadata.ownerReferences")
	if err != nil {
		return struct{}{}, nil
	}
	k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
	for _, ownerReferenceReader := range ownerReferencesReaders.Children() {
		kind, err := ownerReferenceReader.ReadString("kind")
		if err != nil {
			continue
		}
		kind = strings.ToLower(kind)
		apiVersion, err := ownerReferenceReader.ReadString("apiVersion")
		if err != nil {
			continue
		}
		name, err := ownerReferenceReader.ReadString("name")
		if err != nil {
			continue
		}
		if !strings.Contains(apiVersion, "/") {
			apiVersion = "core/" + apiVersion
		}
		namespace := k8sFieldSet.K8sOperation.Namespace
		if _, ok := r.nonNamespacedOwnerTypes[fmt.Sprintf("%s#%s", apiVersion, kind)]; ok {
			namespace = "cluster-scope"
		}
		path := resourcepath.ResourcePath{
			Path:               event.EventTargetResource.ResourcePathString(),
			ParentRelationship: enum.RelationshipChild,
		}
		ownerResource := resourcepath.NameLayerGeneralItem(apiVersion, kind, namespace, name)
		ownerSubresource := resourcepath.OwnerSubresource(ownerResource, k8sFieldSet.K8sOperation.Name, k8sFieldSet.K8sOperation.GetSingularKindName())
		cs.AddResourceAlias(path, ownerSubresource)
	}
	return struct{}{}, nil
}

// TaskID implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (r *resourceOwnerReferenceModifierTaskSetting) TaskID() taskid.TaskImplementationID[struct{}] {
	return commonlogk8sauditv2_contract.ResourceOwnerReferenceModifierTaskID
}

// ResourcePairs implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (r *resourceOwnerReferenceModifierTaskSetting) ResourcePairs(ctx context.Context, groupedLogs commonlogk8sauditv2_contract.ResourceManifestLogGroupMap) ([]commonlogk8sauditv2_contract.ResourcePair, error) {
	result := make([]commonlogk8sauditv2_contract.ResourcePair, 0, len(groupedLogs))
	for _, group := range groupedLogs {
		if group.Resource.Type() == commonlogk8sauditv2_contract.Namespace {
			continue
		}
		result = append(result, commonlogk8sauditv2_contract.ResourcePair{
			TargetGroup: group.Resource,
		})
	}
	return result, nil
}

// Dependencies implements commonlogk8sauditv2_contract.ResourceBasedHistoryModifer.
func (r *resourceOwnerReferenceModifierTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// PassCount implements commonlogk8sauditv2_contract.ResourceBasedHistoryModifer.
func (r *resourceOwnerReferenceModifierTaskSetting) PassCount() int {
	return 1
}

// GroupedLogTask implements commonlogk8sauditv2_contract.ResourceBasedHistoryModifer.
func (r *resourceOwnerReferenceModifierTaskSetting) GroupedLogTask() taskid.TaskReference[commonlogk8sauditv2_contract.ResourceManifestLogGroupMap] {
	return commonlogk8sauditv2_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogSerializerTask implements commonlogk8sauditv2_contract.ResourceBasedHistoryModifer.
func (r *resourceOwnerReferenceModifierTaskSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8sauditv2_contract.K8sAuditLogSerializerTaskID.Ref()
}

var _ commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting[struct{}] = (*resourceOwnerReferenceModifierTaskSetting)(nil)
