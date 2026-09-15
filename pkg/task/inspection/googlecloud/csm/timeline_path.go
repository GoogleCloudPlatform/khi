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

package csm

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// getK8sNamespacedResourceTimeline constructs a standard namespaced resource timeline path using common K8s contract helpers.
func getK8sNamespacedResourceTimeline(ctx context.Context, clusterName string, apiVersion string, kind string, namespace string, name string) *khifilev6.TimelinePath {
	clusterTimeline := k8saudit.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionTimeline := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterTimeline, apiVersion)
	kindTimeline := k8saudit.MustK8sKindTimeline(ctx, apiVersionTimeline, kind)
	namespaceTimeline := k8saudit.MustK8sNamespaceTimeline(ctx, kindTimeline, namespace)
	return k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, name)
}

// MustCSMServerAccessTimeline returns the timeline path for CSM Server Access under a Pod.
func MustCSMServerAccessTimeline(ctx context.Context, clusterName string, podNamespace string, podName string, containerName string) *khifilev6.TimelinePath {
	podPath := getK8sNamespacedResourceTimeline(ctx, clusterName, "core/v1", "pod", podNamespace, podName)
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	suffix := "server"
	if containerName != "" {
		suffix = "server:" + containerName
	}
	return builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{
		Name: suffix,
		Type: TimelineTypeCSMTrafficLog,
	})
}

// MustCSMClientAccessTimeline returns the timeline path for CSM Client Access under a Pod.
func MustCSMClientAccessTimeline(ctx context.Context, clusterName string, podNamespace string, podName string) *khifilev6.TimelinePath {
	podPath := getK8sNamespacedResourceTimeline(ctx, clusterName, "core/v1", "pod", podNamespace, podName)
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	return builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{
		Name: "client",
		Type: TimelineTypeCSMTrafficLog,
	})
}

// MustCSMServiceServerAccessTimeline returns the timeline path for CSM Service Server Access.
func MustCSMServiceServerAccessTimeline(ctx context.Context, clusterName string, serviceNamespace string, serviceName string) *khifilev6.TimelinePath {
	servicePath := getK8sNamespacedResourceTimeline(ctx, clusterName, "core/v1", "service", serviceNamespace, serviceName)
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	return builder.TimelineAccumulator.GetPath(servicePath, khifilev6.PathSegment{
		Name: "server",
		Type: TimelineTypeCSMTrafficLog,
	})
}

// MustCSMServiceClientAccessTimeline returns the timeline path for CSM Service Client Access.
func MustCSMServiceClientAccessTimeline(ctx context.Context, clusterName string, serviceNamespace string, serviceName string) *khifilev6.TimelinePath {
	servicePath := getK8sNamespacedResourceTimeline(ctx, clusterName, "core/v1", "service", serviceNamespace, serviceName)
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	return builder.TimelineAccumulator.GetPath(servicePath, khifilev6.PathSegment{
		Name: "client",
		Type: TimelineTypeCSMTrafficLog,
	})
}
