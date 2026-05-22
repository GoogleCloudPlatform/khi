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

package googlecloudlogk8snode_contract

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustNodeComponentTimeline returns the timeline path for a node component (e.g., containerd, kubelet).
func MustNodeComponentTimeline(ctx context.Context, nodeTimeline *khifilev6.TimelinePath, componentName string) *khifilev6.TimelinePath {
	if nodeTimeline == nil || nodeTimeline.Type.GetId() != inspectioncore_contract.TimelineTypeResource.GetId() {
		panic("parent timeline path must be Resource type")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(nodeTimeline, khifilev6.PathSegment{
		Name: componentName,
		Type: TimelineTypeNodeComponent,
	})
}
