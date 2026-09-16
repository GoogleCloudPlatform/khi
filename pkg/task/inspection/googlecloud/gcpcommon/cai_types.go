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

package gcpcommon

import (
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// LogTypeCAIResourceSnapshot represents the log type for existing resources discovered from CAI.
var LogTypeCAIResourceSnapshot = style.MustRegisterLogType(
	"Asset Inventory",
	"Asset Inventory Resource Snapshot",
	style.Color{R: 0.2, G: 0.4, B: 0.6, A: 1.0},
	style.ColorWhite,
)

// CAIAssetSnapshot represents a single temporal asset snapshot captured from Cloud Asset Inventory.
type CAIAssetSnapshot struct {
	TemporalAsset *assetpb.TemporalAsset
}

// StartTime returns the start time of the temporal asset's validity window, or zero time if absent.
func (s *CAIAssetSnapshot) StartTime() time.Time {
	if st := s.TemporalAsset.GetWindow().GetStartTime(); st != nil {
		return st.AsTime()
	}
	return time.Time{}
}

// CAITaskIDSet groups the task implementation IDs for a standard CAI inspection pipeline.
type CAITaskIDSet struct {
	Fetcher        taskid.TaskImplementationID[[]*CAIAssetSnapshot]
	RawLog         taskid.TaskImplementationID[[]*log.Log]
	LogGrouper     taskid.TaskImplementationID[inspectiontaskbase.LogGroupMap]
	LogIngester    taskid.TaskImplementationID[struct{}]
	TimelineMapper taskid.TaskImplementationID[struct{}]
}

// NewCAITaskIDSet creates a CAITaskIDSet using a task ID prefix such as "cloud.google.com/cai/gke/".
func NewCAITaskIDSet(prefix string) CAITaskIDSet {
	return CAITaskIDSet{
		Fetcher:        taskid.NewDefaultImplementationID[[]*CAIAssetSnapshot](prefix + "fetcher"),
		RawLog:         taskid.NewDefaultImplementationID[[]*log.Log](prefix + "raw-logs"),
		LogGrouper:     taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](prefix + "grouper"),
		LogIngester:    taskid.NewDefaultImplementationID[struct{}](prefix + "log-ingester"),
		TimelineMapper: taskid.NewDefaultImplementationID[struct{}](prefix + "timeline-mapper"),
	}
}
