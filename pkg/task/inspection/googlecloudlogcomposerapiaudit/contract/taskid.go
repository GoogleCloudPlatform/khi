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

package googlecloudlogcomposerapiaudit_contract

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// ComposerAPIAuditLogTaskIDPrefix is the task ID prefix for Composer API audit log tasks.
var ComposerAPIAuditLogTaskIDPrefix = "cloud.google.com/log/composer-api/"

// ClusterIdentityTaskID is the task ID for aliasing the cluster/project identity.
var ClusterIdentityTaskID = taskid.NewDefaultImplementationID[googlecloudk8scommon_contract.GoogleCloudClusterIdentity](ComposerAPIAuditLogTaskIDPrefix + "cluster-identity")

// ListLogEntriesTaskID is the task ID for the task that queries Composer API audit logs from Cloud Logging.
var ListLogEntriesTaskID = taskid.NewDefaultImplementationID[[]*log.Log](ComposerAPIAuditLogTaskIDPrefix + "query")

// FieldSetReaderTaskID is the task ID to read field sets for processing Composer audit logs in later tasks.
var FieldSetReaderTaskID = taskid.NewDefaultImplementationID[[]*log.Log](ComposerAPIAuditLogTaskIDPrefix + "fieldset-reader")

// LogIngesterTaskID is the task ID to finalize Composer audit logs to be included in the final output.
var LogIngesterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](ComposerAPIAuditLogTaskIDPrefix + "log-ingester")

// LogGrouperTaskID is the task ID to group Composer audit logs by environment name.
var LogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](ComposerAPIAuditLogTaskIDPrefix + "grouper")

// LogToTimelineMapperTaskID is the task ID for associating events/revisions with Composer audit logs.
var LogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.TimelineMapperResult](ComposerAPIAuditLogTaskIDPrefix + "timeline-mapper")
