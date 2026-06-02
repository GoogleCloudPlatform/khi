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

// Package googlecloudlognetworkapiaudit_contract defines the timeline path builders.
package googlecloudlognetworkapiaudit_contract

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustNEGTimeline returns the hierarchical timeline path for GKE NEGs under the cluster.
// The NEG is an API resource with apiVersion "networking.gke.io/v1beta1", kind "servicenetworkendpointgroup".
func MustNEGTimeline(ctx context.Context, clusterName string, namespace string, negName string) *khifilev6.TimelinePath {
	if namespace == "" {
		namespace = "unknown"
	}
	if negName == "" {
		negName = "unknown"
	}
	clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "networking.gke.io/v1beta1")
	kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiPath, "servicenetworkendpointgroup")
	nsPath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, namespace)
	return commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, nsPath, negName)
}

// MustNEGOperationTimeline returns the timeline path for the GCE operation under a NEG.
func MustNEGOperationTimeline(ctx context.Context, negPath *khifilev6.TimelinePath, methodName string, operationID string) *khifilev6.TimelinePath {
	if negPath == nil {
		panic("negPath must not be nil")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	shortMethodName := "unknown"
	if methodName != "" {
		methodNameSplitted := strings.Split(methodName, ".")
		shortMethodName = methodNameSplitted[len(methodNameSplitted)-1]
	}
	return builder.TimelineAccumulator.GetPath(negPath, khifilev6.PathSegment{
		Name: fmt.Sprintf("%s-%s", shortMethodName, operationID),
		Type: googlecloudcommon_contract.TimelineTypeOperation,
	})
}

// MustNEGUnderResourceTimeline returns the timeline path for a NEG subresource nested under a parent timeline.
func MustNEGUnderResourceTimeline(ctx context.Context, parentPath *khifilev6.TimelinePath, negName string) *khifilev6.TimelinePath {
	if parentPath == nil {
		panic("parentPath must not be nil")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(parentPath, khifilev6.PathSegment{
		Name: negName,
		Type: TimelineTypeNetworkEndpointGroup,
	})
}

// MustGCPResourceTimeline returns the timeline path for a generic GCP resource.
func MustGCPResourceTimeline(ctx context.Context, projectID string, resourceType string, resourceName string) *khifilev6.TimelinePath {
	projectPath := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, projectID)
	resourceTypePath := googlecloudcommon_contract.MustGCPResourceTypeTimeline(ctx, projectPath, resourceType)
	return googlecloudcommon_contract.MustGCPResourceTimeline(ctx, resourceTypePath, resourceName)
}
