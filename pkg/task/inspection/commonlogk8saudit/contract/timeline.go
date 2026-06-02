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

package commonlogk8saudit_contract

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustK8sClusterTimeline returns the timeline path for the Kubernetes Cluster layer.
func MustK8sClusterTimeline(ctx context.Context, clusterName string) *khifilev6.TimelinePath {
	if clusterName == "" {
		clusterName = "unknown"
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: clusterName,
		Type: inspectioncore_contract.TimelineTypeK8sCluster,
	})
}

// MustK8sAPIVersionTimeline returns the timeline path for the API version layer under a K8sCluster.
func MustK8sAPIVersionTimeline(ctx context.Context, clusterTimeline *khifilev6.TimelinePath, apiVersion string) *khifilev6.TimelinePath {
	if clusterTimeline == nil || clusterTimeline.Type.GetId() != inspectioncore_contract.TimelineTypeK8sCluster.GetId() {
		panic("parent timeline path must be K8sCluster type")
	}
	if !strings.Contains(apiVersion, "/") {
		panic(fmt.Sprintf("invalid APIVersion: %s. Missing core/?", apiVersion))
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(clusterTimeline, khifilev6.PathSegment{
		Name: apiVersion,
		Type: inspectioncore_contract.TimelineTypeAPIVersion,
	})
}

// MustK8sKindTimeline returns the timeline path for the resource kind layer.
func MustK8sKindTimeline(ctx context.Context, apiVersionTimeline *khifilev6.TimelinePath, kind string) *khifilev6.TimelinePath {
	if apiVersionTimeline == nil || apiVersionTimeline.Type.GetId() != inspectioncore_contract.TimelineTypeAPIVersion.GetId() {
		panic("parent timeline path must be APIVersion type")
	}
	if kind != strings.ToLower(kind) {
		panic(fmt.Sprintf("invalid Kind: %s. Kind must be all lowercase", kind))
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(apiVersionTimeline, khifilev6.PathSegment{
		Name: kind,
		Type: inspectioncore_contract.TimelineTypeKind,
	})
}

// MustK8sNamespaceTimeline returns the timeline path for the namespace layer.
func MustK8sNamespaceTimeline(ctx context.Context, kindTimeline *khifilev6.TimelinePath, namespace string) *khifilev6.TimelinePath {
	if kindTimeline == nil || kindTimeline.Type.GetId() != inspectioncore_contract.TimelineTypeKind.GetId() {
		panic("parent timeline path must be Kind type")
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(kindTimeline, khifilev6.PathSegment{
		Name: namespace,
		Type: inspectioncore_contract.TimelineTypeNamespace,
	})
}

// MustK8sNamespacedResourceTimeline returns the timeline path for the namespaced resource layer.
func MustK8sNamespacedResourceTimeline(ctx context.Context, namespaceTimeline *khifilev6.TimelinePath, resourceName string) *khifilev6.TimelinePath {
	if namespaceTimeline == nil || namespaceTimeline.Type.GetId() != inspectioncore_contract.TimelineTypeNamespace.GetId() {
		panic("parent timeline path must be Namespace type")
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(namespaceTimeline, khifilev6.PathSegment{
		Name: resourceName,
		Type: inspectioncore_contract.TimelineTypeResource,
	})
}

// MustK8sClusterScopeResourceTimeline returns the timeline path for the cluster-scoped resource layer.
func MustK8sClusterScopeResourceTimeline(ctx context.Context, kindTimeline *khifilev6.TimelinePath, resourceName string) *khifilev6.TimelinePath {
	if kindTimeline == nil || kindTimeline.Type.GetId() != inspectioncore_contract.TimelineTypeKind.GetId() {
		panic("parent timeline path must be Kind type")
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	clusterScopeNamespaceTimeline := MustK8sNamespaceTimeline(ctx, kindTimeline, "cluster-scope")
	return builder.TimelineAccumulator.GetPath(clusterScopeNamespaceTimeline, khifilev6.PathSegment{
		Name: resourceName,
		Type: inspectioncore_contract.TimelineTypeResource,
	})
}

// MustK8sSubresourceTimeline returns the timeline path for the subresource layer.
func MustK8sSubresourceTimeline(ctx context.Context, resourceTimeline *khifilev6.TimelinePath, subresourceName string) *khifilev6.TimelinePath {
	if resourceTimeline == nil || resourceTimeline.Type.GetId() != inspectioncore_contract.TimelineTypeResource.GetId() {
		panic("parent timeline path must be Resource type")
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(resourceTimeline, khifilev6.PathSegment{
		Name: subresourceName,
		Type: inspectioncore_contract.TimelineTypeSubresource,
	})
}

// MustOwnedResourceTimeline returns the timeline path for the owned resource by another resource.
func MustOwnedResourceTimeline(ctx context.Context, resourceTimeline *khifilev6.TimelinePath, ownedResourceName string) *khifilev6.TimelinePath {
	if resourceTimeline == nil || resourceTimeline.Type.GetId() != inspectioncore_contract.TimelineTypeResource.GetId() {
		panic("parent timeline path must be Resource type")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(resourceTimeline, khifilev6.PathSegment{
		Name: ownedResourceName,
		Type: TimelineTypeOwnerReference,
	})
}

// MustK8sContainerTimeline returns the hierarchical timeline path for a Kubernetes Container log nested under a Pod.
func MustK8sContainerTimeline(ctx context.Context, podTimeline *khifilev6.TimelinePath, containerName string) *khifilev6.TimelinePath {
	if podTimeline == nil || podTimeline.Type.GetId() != inspectioncore_contract.TimelineTypeResource.GetId() {
		panic("parent timeline path must be Resource type")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(podTimeline, khifilev6.PathSegment{
		Name: containerName,
		Type: TimelineTypeContainer,
	})
}

// MustResourceTimeline returns the timeline path for a given ResourceIdentity.
func MustResourceTimeline(ctx context.Context, clusterName string, res *ResourceIdentity) *khifilev6.TimelinePath {
	if clusterName == "" {
		clusterName = "unknown"
	}
	if res == nil {
		panic("resource identity must not be nil")
	}
	clusterPath := MustK8sClusterTimeline(ctx, clusterName)
	apiVersionPath := MustK8sAPIVersionTimeline(ctx, clusterPath, res.APIVersion)
	kindPath := MustK8sKindTimeline(ctx, apiVersionPath, res.Kind)
	var resourcePath *khifilev6.TimelinePath
	if res.Namespace != "" {
		namespacePath := MustK8sNamespaceTimeline(ctx, kindPath, res.Namespace)
		resourcePath = MustK8sNamespacedResourceTimeline(ctx, namespacePath, res.Name)
	} else {
		resourcePath = MustK8sClusterScopeResourceTimeline(ctx, kindPath, res.Name)
	}
	if res.SubresourceName != "" {
		return MustK8sSubresourceTimeline(ctx, resourcePath, res.SubresourceName)
	}
	return resourcePath
}
