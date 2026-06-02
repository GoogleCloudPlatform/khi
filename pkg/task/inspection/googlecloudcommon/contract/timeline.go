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

package googlecloudcommon_contract

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustGKEClusterTimeline returns the timeline path for a GKE Cluster.
func MustGKEClusterTimeline(ctx context.Context, clusterName string) *khifilev6.TimelinePath {
	if clusterName == "" {
		clusterName = "unknown"
		slog.WarnContext(ctx, "clusterName is empty, using unknown instead")
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: clusterName,
		Type: TimelineTypeGKE,
	})
}

// MustGKENodePoolTimeline returns the timeline path for a GKE NodePool under a GKE Cluster.
func MustGKENodePoolTimeline(ctx context.Context, gkeClusterTimeline *khifilev6.TimelinePath, nodePoolName string) *khifilev6.TimelinePath {
	if gkeClusterTimeline == nil || gkeClusterTimeline.Type.GetId() != TimelineTypeGKE.GetId() {
		panic("parent timeline path must be GKE type")
	}
	if nodePoolName == "" {
		nodePoolName = "unknown"
		slog.WarnContext(ctx, "nodePoolName is empty, using unknown instead")
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(gkeClusterTimeline, khifilev6.PathSegment{
		Name: nodePoolName,
		Type: TimelineTypeGKENodePool,
	})
}

// MustGCPOperationTimeline returns the timeline path for a GCP long running operation.
// The operation timeline is nested under its associated GKE or GCP resource timeline path.
func MustGCPOperationTimeline(ctx context.Context, parentTimeline *khifilev6.TimelinePath, shortMethodName string, operationID string) *khifilev6.TimelinePath {
	if parentTimeline == nil {
		panic("parent timeline path must not be nil")
	}
	if shortMethodName == "" {
		shortMethodName = "unknown"
		slog.WarnContext(ctx, "shortMethodName is empty, using unknown instead")
	}
	if operationID == "" {
		operationID = "unknown"
		slog.WarnContext(ctx, "operationID is empty, using unknown instead")
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(parentTimeline, khifilev6.PathSegment{
		Name: fmt.Sprintf("%s-%s", shortMethodName, operationID),
		Type: TimelineTypeOperation,
	})
}

// MustGCPProjectTimeline returns the timeline path for a Google Cloud Project root timeline.
func MustGCPProjectTimeline(ctx context.Context, projectID string) *khifilev6.TimelinePath {
	if projectID == "" {
		projectID = "unknown"
		slog.WarnContext(ctx, "projectID is empty, using unknown instead")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: projectID,
		Type: TimelineTypeGCPProject,
	})
}

// MustGCPResourceTypeTimeline returns the timeline path for a GCP resource type layer under a GCP Project.
func MustGCPResourceTypeTimeline(ctx context.Context, projectPath *khifilev6.TimelinePath, resourceType string) *khifilev6.TimelinePath {
	if projectPath == nil || projectPath.Type.GetId() != TimelineTypeGCPProject.GetId() {
		panic("parent timeline path must be GCPProject type")
	}
	if resourceType == "" {
		resourceType = "unknown"
		slog.WarnContext(ctx, "resourceType is empty, using unknown instead")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(projectPath, khifilev6.PathSegment{
		Name: resourceType,
		Type: TimelineTypeGCPResourceType,
	})
}

// MustGCPResourceTimeline returns the timeline path for a GCP resource under a resource type.
func MustGCPResourceTimeline(ctx context.Context, resourceTypePath *khifilev6.TimelinePath, resourceName string) *khifilev6.TimelinePath {
	if resourceTypePath == nil || resourceTypePath.Type.GetId() != TimelineTypeGCPResourceType.GetId() {
		panic("parent timeline path must be GCPResourceType type")
	}
	if resourceName == "" {
		resourceName = "unknown"
		slog.WarnContext(ctx, "resourceName is empty, using unknown instead")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(resourceTypePath, khifilev6.PathSegment{
		Name: resourceName,
		Type: TimelineTypeGCPResource,
	})
}
