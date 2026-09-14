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

package googlecloudcaik8s_contract

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// TaskIDPrefix is the prefix for Google Cloud CAI K8s task IDs.
const TaskIDPrefix = "cloud.google.com/cai/k8s/"

// ClusterResourceFetcherTaskID is the task ID for fetching cluster resource snapshots from CAI.
var ClusterResourceFetcherTaskID = taskid.NewDefaultImplementationID[[]*ClusterResourceSnapshot](TaskIDPrefix + "fetcher")

// RawLogTaskID is the task ID for raw logs generated from CAI cluster resource snapshots.
var RawLogTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "raw-logs")

// LogGrouperTaskID is the task ID for grouping CAI cluster resource snapshot logs.
var LogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "grouper")

// LogIngesterTaskID is the task ID for ingesting CAI cluster resource snapshot log metadata.
var LogIngesterTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "log-ingester")

// LogToTimelineMapperTaskID is the task ID for mapping CAI cluster resource snapshots to timeline revisions.
var LogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "timeline-mapper")
