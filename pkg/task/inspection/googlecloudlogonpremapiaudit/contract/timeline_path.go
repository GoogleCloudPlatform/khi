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

package googlecloudlogonpremapiaudit_contract

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustOnPremProjectTimeline returns the hierarchical timeline path for an On-Prem Project.
func MustOnPremProjectTimeline(ctx context.Context, project string) *khifilev6.TimelinePath {
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: project,
		Type: TimelineTypeOnPremProject,
	})
}

// MustOnPremClusterTimeline returns the hierarchical timeline path for an On-Prem Cluster under a Project.
func MustOnPremClusterTimeline(ctx context.Context, projectPath *khifilev6.TimelinePath, clusterName string) *khifilev6.TimelinePath {
	if projectPath == nil || projectPath.Type.GetId() != TimelineTypeOnPremProject.GetId() {
		panic("parent timeline path must be On-Prem Project type")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(projectPath, khifilev6.PathSegment{
		Name: clusterName,
		Type: TimelineTypeOnPremCluster,
	})
}

// MustOnPremNodePoolTimeline returns the hierarchical timeline path for an On-Prem NodePool under a Cluster.
func MustOnPremNodePoolTimeline(ctx context.Context, clusterPath *khifilev6.TimelinePath, nodepoolName string) *khifilev6.TimelinePath {
	if clusterPath == nil || clusterPath.Type.GetId() != TimelineTypeOnPremCluster.GetId() {
		panic("parent timeline path must be On-Prem Cluster type")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{
		Name: nodepoolName,
		Type: TimelineTypeOnPremNodePool,
	})
}
