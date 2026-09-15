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

package commonlogcsmcp_contract

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// MustCSMCPPodLogTimeline returns the timeline path for a CSM CP Pod Log nested under a Pod.
func MustCSMCPPodLogTimeline(ctx context.Context, podTimeline *khifilev6.TimelinePath) *khifilev6.TimelinePath {
	if podTimeline == nil || podTimeline.Type.GetId() != inspectioncore.TimelineTypeResource.GetId() {
		panic("parent timeline path must be Resource type")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)
	return builder.TimelineAccumulator.GetPath(podTimeline, khifilev6.PathSegment{
		Name: "CSM CP",
		Type: TimelineTypeCSMCPPodLog,
	})
}

// MustCSMCPConnectionTimeline returns the timeline path for a CSM CP Connection nested under a Pod.
func MustCSMCPConnectionTimeline(ctx context.Context, podTimeline *khifilev6.TimelinePath, connectionID string) *khifilev6.TimelinePath {
	if podTimeline == nil || podTimeline.Type.GetId() != inspectioncore.TimelineTypeResource.GetId() {
		panic("parent timeline path must be Resource type")
	}
	if connectionID == "" {
		connectionID = "unknown"
		slog.WarnContext(ctx, "connectionID is empty, using unknown instead")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)
	return builder.TimelineAccumulator.GetPath(podTimeline, khifilev6.PathSegment{
		Name: fmt.Sprintf("connection-%s", connectionID),
		Type: TimelineTypeCSMCPConnection,
	})
}
