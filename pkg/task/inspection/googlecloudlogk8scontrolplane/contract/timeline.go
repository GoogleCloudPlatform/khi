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

// MustControlPlaneComponentTimeline returns the timeline path for a Kubernetes control plane component.
func MustControlPlaneComponentTimeline(ctx context.Context, clusterName string, componentName string) *khifilev6.TimelinePath {
	clusterTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, clusterName)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(clusterTimeline, khifilev6.PathSegment{
		Name: componentName,
		Type: TimelineTypeControlPlaneComponent,
	})
}

// MustControllerManagerControlPlaneTimeline returns the timeline path for a Kubernetes controller manager control plane component.
func MustControllerManagerControlPlaneTimeline(ctx context.Context, clusterName string, controllerName string) *khifilev6.TimelinePath {
	if controllerName == "" {
		return MustControlPlaneComponentTimeline(ctx, clusterName, "controller-manager")
	}
	return MustControlPlaneComponentTimeline(ctx, clusterName, fmt.Sprintf("%s(controller-manager)", controllerName))
}
