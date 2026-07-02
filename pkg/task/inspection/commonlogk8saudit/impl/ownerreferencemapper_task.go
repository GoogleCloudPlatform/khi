// Copyright 2026 Google LLC
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

package commonlogk8saudit_impl

import (
	"context"
	"fmt"
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

// ResourceOwnerReferenceTimelineMapperTask is the V2 task to map logs into resource owner reference aliases.
var ResourceOwnerReferenceTimelineMapperTask = commonlogk8saudit_contract.NewManifestLogToTimelineMapper[struct{}](&resourceOwnerReferenceTimelineMapperTaskSetting{
	nonNamespacedOwnerTypes: map[string]struct{}{
		"core/v1#node": {},
	},
})

// resourceOwnerReferenceTimelineMapperTaskSetting maps resource owner references to timeline aliases under the V2 model.
type resourceOwnerReferenceTimelineMapperTaskSetting struct {
	commonlogk8saudit_contract.ManifestStatelessMapperBase

	// nonNamespacedOwnerTypes is the set of owner types that are not namespaced.
	nonNamespacedOwnerTypes map[string]struct{}
}

// Dependencies implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (r *resourceOwnerReferenceTimelineMapperTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (r *resourceOwnerReferenceTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[commonlogk8saudit_contract.ResourceManifestLogGroupMap] {
	return commonlogk8saudit_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogIngesterTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (r *resourceOwnerReferenceTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8saudit_contract.K8sAuditLogIngesterTaskID.Ref()
}

// TaskID implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (r *resourceOwnerReferenceTimelineMapperTaskSetting) TaskID() taskid.TaskImplementationID[inspectiontaskbase.TimelineMapperResult] {
	return commonlogk8saudit_contract.ResourceOwnerReferenceTimelineMapperTaskID
}

// ResolveRelatedGroupSets implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (r *resourceOwnerReferenceTimelineMapperTaskSetting) ResolveRelatedGroupSets(ctx context.Context, groupedLogs commonlogk8saudit_contract.ResourceManifestLogGroupMap) ([]commonlogk8saudit_contract.RelatedGroupSet, error) {
	result := []commonlogk8saudit_contract.RelatedGroupSet{}
	for _, group := range groupedLogs {
		if group.Resource.Type() == commonlogk8saudit_contract.Namespace {
			continue
		}
		result = append(result, commonlogk8saudit_contract.RelatedGroupSet{
			Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
				"target": group,
			},
		})
	}
	return result, nil
}

// ProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (r *resourceOwnerReferenceTimelineMapperTaskSetting) ProcessLog(ctx context.Context, event commonlogk8saudit_contract.MultiGroupLogEvent, state struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs := khifilev6.NewTimelineChangeSet(event.Log)

	bodyReader, ok := event.GetLastBodyReader("target")
	if !ok || bodyReader == nil {
		return cs, struct{}{}, nil
	}
	ownerReferencesReaders, err := bodyReader.GetReader("metadata.ownerReferences")
	if err != nil {
		return cs, struct{}{}, nil
	}
	k8sFieldSet, err := log.GetFieldSet(event.Log, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})
	if err != nil {
		return cs, struct{}{}, err
	}

	aliasPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity)

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
		namespace := k8sFieldSet.Namespace
		if _, ok := r.nonNamespacedOwnerTypes[fmt.Sprintf("%s#%s", apiVersion, kind)]; ok {
			namespace = "cluster-scope"
		}

		ownerIdentity := &commonlogk8saudit_contract.ResourceIdentity{
			APIVersion: apiVersion,
			Kind:       kind,
			Namespace:  namespace,
			Name:       name,
		}
		ownerPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, ownerIdentity)

		ownLabel := fmt.Sprintf("%s[kind:%s]", event.ResourceIdentity.Name, event.ResourceIdentity.Kind)
		targetPath := commonlogk8saudit_contract.MustOwnedResourceTimeline(ctx, ownerPath, ownLabel)

		cs.AddAlias(targetPath, aliasPath)
	}
	return cs, struct{}{}, nil
}

// Explicit interface compliance assertion.
var _ commonlogk8saudit_contract.ManifestLogToTimelineMapper[struct{}] = (*resourceOwnerReferenceTimelineMapperTaskSetting)(nil)
