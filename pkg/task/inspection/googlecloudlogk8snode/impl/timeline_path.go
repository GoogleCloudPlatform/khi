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

package googlecloudlogk8snode_impl

import (
	"context"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

// MustK8sNodeTimeline returns the timeline path for the Kubernetes Node resource layer.
func MustK8sNodeTimeline(ctx context.Context, clusterName string, nodeName string) *khifilev6.TimelinePath {
	clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
	kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionPath, "node")
	return commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kindPath, nodeName)
}

// MustK8sPodTimeline returns the timeline path for a Kubernetes Pod resource layer.
func MustK8sPodTimeline(ctx context.Context, clusterName string, namespace string, podName string) *khifilev6.TimelinePath {
	clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
	kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionPath, "pod")
	namespacePath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, namespace)
	return commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespacePath, podName)
}
