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

// Package dpv2policy defines the contract of the tasks that visualize GKE Dataplane V2 network policy logs (logs/policy-action).
package dpv2policy

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

// TaskIDPrefix is the prefix for all task IDs in this package.
const TaskIDPrefix = "cloud.google.com/log/dpv2-policy-action/"

// ClusterIdentityTaskID is the task ID for aliasing the cluster identity.
var ClusterIdentityTaskID = taskid.NewDefaultImplementationID[k8scommon.GoogleCloudClusterIdentity](TaskIDPrefix + "cluster-identity")

// ListLogEntriesTaskID is the task ID for the task that queries network policy logs from Cloud Logging.
var ListLogEntriesTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "list-log-entries")

// LogIngesterTaskID is the task ID to finalize the network policy logs to be included in the final output.
var LogIngesterTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "log-ingester")

// LogGrouperTaskID is the task ID to group network policy logs by their reporting node for parallel processing.
var LogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "grouper")

// LogToTimelineMapperTaskID is the task ID for associating network policy logs with Pod, network policy, and Node timelines.
var LogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "timeline-mapper")
