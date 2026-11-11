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

package k8saudit_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
)

// resourceOwnerReferenceTimelineMapperTask is the task to map logs into resource owner reference aliases.
var resourceOwnerReferenceTimelineMapperTask = k8saudit.DefineManifestLogToTimelineMapper[struct{}](
	k8saudit.ResourceOwnerReferenceTimelineMapperTaskID,
	func(_ *coretask.Binder) k8saudit.ManifestLogToTimelineMapper[struct{}] {
		return &resourceOwnerReferenceTimelineMapperTaskSetting{
			nonNamespacedOwnerTypes: map[string]struct{}{
				"core/v1#node": {},
			},
			podOwnedTypesByName: map[string]struct{}{
				// The Cilium agent creates a CiliumEndpoint with the same namespace and name as its Pod and sets the Pod as the owner.
				"cilium.io#ciliumendpoint": {},
			},
		}
	},
)

// resourceOwnerReferenceTimelineMapperTaskSetting maps resource owner references to timeline aliases under the model.
type resourceOwnerReferenceTimelineMapperTaskSetting struct {
	k8saudit.ManifestStatelessMapperBase

	// nonNamespacedOwnerTypes is the set of owner types that are not namespaced.
	nonNamespacedOwnerTypes map[string]struct{}
	// podOwnedTypesByName is the set of resource types ("<API group>#<kind>") whose resources are owned by the Pod with the same namespace and name.
	// It links these resources to their Pods even when audit logs do not record their manifests, which is common for custom resources.
	podOwnedTypesByName map[string]struct{}
}

// ResolveRelatedGroupSets implements k8saudit.ManifestLogToTimelineMapper.
func (r *resourceOwnerReferenceTimelineMapperTaskSetting) ResolveRelatedGroupSets(ctx context.Context, groupedLogs k8saudit.ResourceManifestLogGroupMap) ([]k8saudit.RelatedGroupSet, error) {
	result := []k8saudit.RelatedGroupSet{}
	for _, group := range groupedLogs {
		if group.Resource.Type() == k8saudit.Namespace {
			continue
		}
		result = append(result, k8saudit.RelatedGroupSet{
			Roles: map[string]*k8saudit.ResourceManifestLogGroup{
				"target": group,
			},
		})
	}
	return result, nil
}

var (
	pathOwnerReferences = structured.CompileFieldPath("metadata.ownerReferences")
	pathOwnerKind       = structured.CompileFieldPath("kind")
	pathOwnerAPIVersion = structured.CompileFieldPath("apiVersion")
	pathOwnerName       = structured.CompileFieldPath("name")
)

// ProcessLog implements k8saudit.ManifestLogToTimelineMapper.
func (r *resourceOwnerReferenceTimelineMapperTaskSetting) ProcessLog(ctx context.Context, event k8saudit.MultiGroupLogEvent, state struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs := khifilev6.NewTimelineChangeSet(event.Log)

	k8sFieldSet, err := k8saudit.ExtractK8sAuditLog(event.Log.NodeReader, nil)
	if err != nil {
		return cs, struct{}{}, err
	}
	if k8sFieldSet.IsDryRun {
		return cs, struct{}{}, nil
	}

	aliasPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity)

	if r.isOwnedByPodWithSameName(event.ResourceIdentity) {
		podPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, &k8saudit.ResourceIdentity{
			APIVersion: "core/v1",
			Kind:       "pod",
			Namespace:  event.ResourceIdentity.Namespace,
			Name:       event.ResourceIdentity.Name,
		})
		// This path equals the path derived from the Pod in the ownerReferences of the manifest, so both aliases point to the same timeline.
		cs.AddAlias(k8saudit.MustOwnedResourceTimeline(ctx, podPath, ownedResourceTimelineName(event.ResourceIdentity)), aliasPath)
	}

	bodyReader, ok := event.GetLastBodyReader("target")
	if !ok || bodyReader == nil {
		return cs, struct{}{}, nil
	}
	ownerReferencesReaders, err := bodyReader.GetReader(pathOwnerReferences)
	if err != nil {
		return cs, struct{}{}, nil
	}

	for _, ownerReferenceReader := range ownerReferencesReaders.Children() {
		kind, err := ownerReferenceReader.ReadString(pathOwnerKind)
		if err != nil {
			continue
		}
		kind = strings.ToLower(kind)
		apiVersion, err := ownerReferenceReader.ReadString(pathOwnerAPIVersion)
		if err != nil {
			continue
		}
		name, err := ownerReferenceReader.ReadString(pathOwnerName)
		if err != nil {
			continue
		}
		if !strings.Contains(apiVersion, "/") {
			apiVersion = "core/" + apiVersion
		}
		namespace := k8sFieldSet.Namespace
		if _, ok := r.nonNamespacedOwnerTypes[fmt.Sprintf("%s#%s", apiVersion, kind)]; ok {
			namespace = k8saudit.ClusterScopeNamespace
		}

		ownerIdentity := &k8saudit.ResourceIdentity{
			APIVersion: apiVersion,
			Kind:       kind,
			Namespace:  namespace,
			Name:       name,
		}
		ownerPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, ownerIdentity)

		targetPath := k8saudit.MustOwnedResourceTimeline(ctx, ownerPath, ownedResourceTimelineName(event.ResourceIdentity))

		cs.AddAlias(targetPath, aliasPath)
	}
	return cs, struct{}{}, nil
}

// isOwnedByPodWithSameName reports whether the resource is of a type owned by the Pod with the same namespace and name.
func (r *resourceOwnerReferenceTimelineMapperTaskSetting) isOwnedByPodWithSameName(identity *k8saudit.ResourceIdentity) bool {
	if identity.Type() != k8saudit.Resource || identity.Namespace == "" || identity.Namespace == k8saudit.ClusterScopeNamespace {
		return false
	}
	group, _, _ := strings.Cut(identity.APIVersion, "/")
	_, found := r.podOwnedTypesByName[fmt.Sprintf("%s#%s", group, identity.Kind)]
	return found
}

// ownedResourceTimelineName returns the name of the timeline showing an owned resource under the timeline of its owner.
func ownedResourceTimelineName(identity *k8saudit.ResourceIdentity) string {
	return fmt.Sprintf("%s[kind:%s]", identity.Name, identity.Kind)
}

// Explicit interface compliance assertion.
var _ k8saudit.ManifestLogToTimelineMapper[struct{}] = (*resourceOwnerReferenceTimelineMapperTaskSetting)(nil)
