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

package googlecloudlogcomputeapiaudit_contract

import (
	"context"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

// MustNodeTimelinePath returns the hierarchical TimelinePath for a Kubernetes Node resource under V6 format.
func MustNodeTimelinePath(ctx context.Context, clusterName string, nodeName string) *khifilev6.TimelinePath {
	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "node")
	namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "cluster-scope")
	return commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, nodeName)
}
