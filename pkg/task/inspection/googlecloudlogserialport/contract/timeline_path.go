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

package googlecloudlogserialport_contract

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustSerialPortTimeline returns the hierarchical timeline path for a serial port of a GCE node.
func MustSerialPortTimeline(ctx context.Context, clusterName, nodeName, port string) *khifilev6.TimelinePath {
	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "node")
	nodeTimeline := commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kindTimeline, nodeName)

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(nodeTimeline, khifilev6.PathSegment{
		Name: port,
		Type: TimelineTypeSerialPort,
	})
}
