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

// K8sTaskIDPrefix is the prefix for Google Cloud CAI K8s task IDs.
const K8sTaskIDPrefix = "cloud.google.com/cai/k8s/"

// GKETaskIDPrefix is the prefix for Google Cloud CAI GKE task IDs.
const GKETaskIDPrefix = "cloud.google.com/cai/gke/"

// ClusterResourceFetcherTaskID is the task ID for fetching cluster resource snapshots from CAI.
var ClusterResourceFetcherTaskID = taskid.NewDefaultImplementationID[[]*ClusterResourceSnapshot](K8sTaskIDPrefix + "fetcher")

// RawLogTaskID is the task ID for raw logs generated from CAI cluster resource snapshots.
var RawLogTaskID = taskid.NewDefaultImplementationID[[]*log.Log](K8sTaskIDPrefix + "raw-logs")

// LogGrouperTaskID is the task ID for grouping CAI cluster resource snapshot logs.
var LogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](K8sTaskIDPrefix + "grouper")

// LogIngesterTaskID is the task ID for ingesting CAI cluster resource snapshot log metadata.
var LogIngesterTaskID = taskid.NewDefaultImplementationID[struct{}](K8sTaskIDPrefix + "log-ingester")

// LogToTimelineMapperTaskID is the task ID for mapping CAI cluster resource snapshots to timeline revisions.
var LogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](K8sTaskIDPrefix + "timeline-mapper")

// GKEResourceFetcherTaskID is the task ID for fetching GKE cluster and nodepool snapshots from CAI.
var GKEResourceFetcherTaskID = taskid.NewDefaultImplementationID[[]*GKEResourceSnapshot](GKETaskIDPrefix + "fetcher")

// GKERawLogTaskID is the task ID for raw logs generated from CAI GKE resource snapshots.
var GKERawLogTaskID = taskid.NewDefaultImplementationID[[]*log.Log](GKETaskIDPrefix + "raw-logs")

// GKELogGrouperTaskID is the task ID for grouping CAI GKE resource snapshot logs.
var GKELogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](GKETaskIDPrefix + "grouper")

// GKELogIngesterTaskID is the task ID for ingesting CAI GKE resource snapshot log metadata.
var GKELogIngesterTaskID = taskid.NewDefaultImplementationID[struct{}](GKETaskIDPrefix + "log-ingester")

// GKELogToTimelineMapperTaskID is the task ID for mapping CAI GKE resource snapshots to timeline revisions.
var GKELogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](GKETaskIDPrefix + "timeline-mapper")
