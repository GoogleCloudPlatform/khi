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

package computeapiaudit

import (
	"context"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
)

// MustNodeTimelinePath returns the hierarchical TimelinePath for a Kubernetes Node resource under V6 format.
func MustNodeTimelinePath(ctx context.Context, clusterName string, nodeName string) *khifilev6.TimelinePath {
	clusterTimeline := k8saudit.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionTimeline := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := k8saudit.MustK8sKindTimeline(ctx, apiVersionTimeline, "node")
	namespaceTimeline := k8saudit.MustK8sNamespaceTimeline(ctx, kindTimeline, "cluster-scope")
	return k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, nodeName)
}
