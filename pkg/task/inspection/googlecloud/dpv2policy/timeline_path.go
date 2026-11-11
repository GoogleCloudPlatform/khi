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

package dpv2policy

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

const (
	// PodOutboundTimelineName is the name of the network policy timeline of a Pod for the connections whose source is the Pod.
	PodOutboundTimelineName = "outbound"
	// PodInboundTimelineName is the name of the network policy timeline of a Pod for the connections whose destination is the Pod.
	PodInboundTimelineName = "inbound"
	// NodeTimelineName is the name of the network policy timeline of a Node for the connections between endpoints that are not Pods.
	NodeTimelineName = "policy-action"
)

// MustNetworkPolicyTimeline returns the timeline path for network policy decisions nested under a Pod or Node resource timeline.
// Constructs a path like: <apiVersion>/<kind>/<namespace>/<name>/<timelineName>
func MustNetworkPolicyTimeline(ctx context.Context, resourceTimeline *khifilev6.TimelinePath, timelineName string) *khifilev6.TimelinePath {
	if resourceTimeline == nil || resourceTimeline.Type.GetId() != inspectioncore.TimelineTypeResource.GetId() {
		panic("parent timeline path must be Resource type")
	}

	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)
	return builder.TimelineAccumulator.GetPath(resourceTimeline, khifilev6.PathSegment{
		Name: timelineName,
		Type: TimelineTypeNetworkPolicy,
	})
}
