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

package googlecloudloggkeautoscaler_contract

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustAutoscalerTimeline returns the timeline path for GKE cluster autoscaler under the Kubernetes Cluster timeline.
func MustAutoscalerTimeline(ctx context.Context, clusterTimeline *khifilev6.TimelinePath) *khifilev6.TimelinePath {
	if clusterTimeline == nil || clusterTimeline.Type.GetId() != inspectioncore_contract.TimelineTypeK8sCluster.GetId() {
		panic("parent timeline path must be K8sCluster type")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(clusterTimeline, khifilev6.PathSegment{
		Name: "cluster-autoscaler",
		Type: TimelineTypeAutoscaler,
	})
}

// MustMigTimeline returns the timeline path for a Managed Instance Group (MIG) under a nodepool timeline.
func MustMigTimeline(ctx context.Context, nodepoolTimeline *khifilev6.TimelinePath, migName string) *khifilev6.TimelinePath {
	if nodepoolTimeline == nil || nodepoolTimeline.Type.GetId() != googlecloudcommon_contract.TimelineTypeGKENodePool.GetId() {
		panic("parent timeline path must be GKENodePool type")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(nodepoolTimeline, khifilev6.PathSegment{
		Name: migName,
		Type: TimelineTypeManagedInstanceGroup,
	})
}
