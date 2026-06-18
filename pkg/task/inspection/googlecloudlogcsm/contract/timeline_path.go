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

package googlecloudlogcsm_contract

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// getK8sNamespacedResourceTimeline constructs a standard namespaced resource timeline path using common K8s contract helpers.
func getK8sNamespacedResourceTimeline(ctx context.Context, clusterName string, apiVersion string, kind string, namespace string, name string) *khifilev6.TimelinePath {
	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, apiVersion)
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, kind)
	namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, namespace)
	return commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, name)
}

// MustCSMServerAccessTimeline returns the timeline path for CSM Server Access under a Pod.
func MustCSMServerAccessTimeline(ctx context.Context, clusterName string, podNamespace string, podName string, containerName string) *khifilev6.TimelinePath {
	podPath := getK8sNamespacedResourceTimeline(ctx, clusterName, "core/v1", "pod", podNamespace, podName)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)

	suffix := "server"
	if containerName != "" {
		suffix = "server:" + containerName
	}
	return builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{
		Name: suffix,
		Type: TimelineTypeCSMAccessLog,
	})
}

// MustCSMClientAccessTimeline returns the timeline path for CSM Client Access under a Pod.
func MustCSMClientAccessTimeline(ctx context.Context, clusterName string, podNamespace string, podName string) *khifilev6.TimelinePath {
	podPath := getK8sNamespacedResourceTimeline(ctx, clusterName, "core/v1", "pod", podNamespace, podName)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)

	return builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{
		Name: "client",
		Type: TimelineTypeCSMAccessLog,
	})
}

// MustCSMServiceServerAccessTimeline returns the timeline path for CSM Service Server Access.
func MustCSMServiceServerAccessTimeline(ctx context.Context, clusterName string, serviceNamespace string, serviceName string) *khifilev6.TimelinePath {
	servicePath := getK8sNamespacedResourceTimeline(ctx, clusterName, "core/v1", "service", serviceNamespace, serviceName)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)

	return builder.TimelineAccumulator.GetPath(servicePath, khifilev6.PathSegment{
		Name: "server",
		Type: TimelineTypeCSMAccessLog,
	})
}

// MustCSMServiceClientAccessTimeline returns the timeline path for CSM Service Client Access.
func MustCSMServiceClientAccessTimeline(ctx context.Context, clusterName string, serviceNamespace string, serviceName string) *khifilev6.TimelinePath {
	servicePath := getK8sNamespacedResourceTimeline(ctx, clusterName, "core/v1", "service", serviceNamespace, serviceName)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)

	return builder.TimelineAccumulator.GetPath(servicePath, khifilev6.PathSegment{
		Name: "client",
		Type: TimelineTypeCSMAccessLog,
	})
}
