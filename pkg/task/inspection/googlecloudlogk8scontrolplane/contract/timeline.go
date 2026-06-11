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

package googlecloudlogk8scontrolplane_contract

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustControlPlaneComponentTimeline returns the timeline path for a Kubernetes control plane component under a GKE cluster.
func MustControlPlaneComponentTimeline(ctx context.Context, gkeTimeline *khifilev6.TimelinePath, componentName string) *khifilev6.TimelinePath {
	if gkeTimeline == nil || gkeTimeline.Type.GetId() != googlecloudcommon_contract.TimelineTypeGKE.GetId() {
		panic("parent timeline path must be GKE type")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	controlPlanesTimeline := builder.TimelineAccumulator.GetPath(gkeTimeline, khifilev6.PathSegment{
		Name: "controlplanes",
		Type: googlecloudcommon_contract.TimelineTypeGKEControlPlanes,
	})
	return builder.TimelineAccumulator.GetPath(controlPlanesTimeline, khifilev6.PathSegment{
		Name: componentName,
		Type: TimelineTypeControlPlaneComponent,
	})
}

// MustControllerManagerControlPlaneTimeline returns the timeline path for a Kubernetes controller manager control plane component.
func MustControllerManagerControlPlaneTimeline(ctx context.Context, gkeTimeline *khifilev6.TimelinePath, controllerName string) *khifilev6.TimelinePath {
	if controllerName == "" {
		return MustControlPlaneComponentTimeline(ctx, gkeTimeline, "controller-manager")
	}
	return MustControlPlaneComponentTimeline(ctx, gkeTimeline, fmt.Sprintf("%s(controller-manager)", controllerName))
}
