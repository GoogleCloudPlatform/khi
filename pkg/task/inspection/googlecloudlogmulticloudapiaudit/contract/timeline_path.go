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

package googlecloudlogmulticloudapiaudit_contract

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustProjectTimeline returns the hierarchical timeline path for a Google Cloud Project.
func MustProjectTimeline(ctx context.Context, projectID string) *khifilev6.TimelinePath {
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "projects/" + projectID,
		Type: TimelineTypeProject,
	})
}

// MustMultiCloudClusterTimeline returns the hierarchical timeline path for a MultiCloud Cluster.
func MustMultiCloudClusterTimeline(ctx context.Context, parent *khifilev6.TimelinePath, clusterName string) *khifilev6.TimelinePath {
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(parent, khifilev6.PathSegment{
		Name: clusterName,
		Type: TimelineTypeMultiCloudCluster,
	})
}

// MustMultiCloudNodepoolTimeline returns the hierarchical timeline path for a MultiCloud NodePool.
func MustMultiCloudNodepoolTimeline(ctx context.Context, parent *khifilev6.TimelinePath, nodepoolName string) *khifilev6.TimelinePath {
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(parent, khifilev6.PathSegment{
		Name: nodepoolName,
		Type: TimelineTypeMultiCloudNodepool,
	})
}

// MustOperationTimeline returns the hierarchical timeline path for an operation on a resource.
func MustOperationTimeline(ctx context.Context, parent *khifilev6.TimelinePath, shortMethodName string, operationID string) *khifilev6.TimelinePath {
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(parent, khifilev6.PathSegment{
		Name: shortMethodName + "-" + operationID,
		Type: inspectioncore_contract.TimelineTypeSubresource,
	})
}
