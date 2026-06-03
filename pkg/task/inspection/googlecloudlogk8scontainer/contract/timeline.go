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

package googlecloudlogk8scontainer_contract

import (
	"context"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

// MustK8sContainerTimeline returns the hierarchical timeline path for a Kubernetes Container log under the specified cluster, namespace, pod, and container name.
func MustK8sContainerTimeline(ctx context.Context, clusterName string, namespace string, podName string, containerName string) *khifilev6.TimelinePath {
	clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
	kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionPath, "pod")
	namespacePath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, namespace)
	podPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespacePath, podName)

	return commonlogk8saudit_contract.MustK8sContainerTimeline(ctx, podPath, containerName)
}
