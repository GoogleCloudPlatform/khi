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

package googlecloudlogk8sevent_contract

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustEventExporterTimeline returns the timeline path for the GKE Event Exporter.
// It panics if the given cluster timeline is nil or not of GKE type.
func MustEventExporterTimeline(ctx context.Context, clusterTimeline *khifilev6.TimelinePath) *khifilev6.TimelinePath {
	if clusterTimeline == nil || clusterTimeline.Type.GetId() != googlecloudcommon_contract.TimelineTypeGKE.GetId() {
		panic("parent timeline path must be GKE type")
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(clusterTimeline, khifilev6.PathSegment{
		Name: "event-exporter",
		Type: TimelineTypeEventExporter,
	})
}
