// Copyright 2025 Google LLC
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

// Package googlecloudlogk8scontrolplane_contract defines the contract for tasks related to GKE control plane component logs.
package googlecloudlogk8scontrolplane_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// K8sControlPlaneLogTaskIDPrefix is the prefix for all task IDs in this package.
const K8sControlPlaneLogTaskIDPrefix = "cloud.google.com/log/k8s-control-plane/"

// InputControlPlaneComponentNameFilterTaskID is the task ID for the form task that inputs the control plane component name filter.
var InputControlPlaneComponentNameFilterTaskID = taskid.NewDefaultImplementationID[*gcpqueryutil.SetFilterParseResult](K8sControlPlaneLogTaskIDPrefix + "input/component-names")

// ListLogEntriesTaskID is the task id for the task that queries controlplane logs from Cloud Logging.
var ListLogEntriesTaskID = taskid.NewDefaultImplementationID[[]*log.Log](K8sControlPlaneLogTaskIDPrefix + "query")

// CommonFieldSetReaderTaskID is the task id to read the common fieldset of controlplane logs for processing the log in the later task.
var CommonFieldSetReaderTaskID = taskid.NewDefaultImplementationID[[]*log.Log](K8sControlPlaneLogTaskIDPrefix + "fieldset-reader-common")

// LogIngesterTaskID is the task ID to finalize the logs to be included in the final output.
var LogIngesterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](K8sControlPlaneLogTaskIDPrefix + "log-ingester")

// SchedulerLogFilterTaskID is the task ID for filtering scheduler logs.
var SchedulerLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](K8sControlPlaneLogTaskIDPrefix + "scheduler-log-filter")

// SchedulerLogFieldSetReaderTaskID is the task ID for reading field sets specific to scheduler logs.
var SchedulerLogFieldSetReaderTaskID = taskid.NewDefaultImplementationID[[]*log.Log](K8sControlPlaneLogTaskIDPrefix + "fieldset-reader-scheduler")

// SchedulerLogGrouperTaskID is the task ID for grouping scheduler logs.
var SchedulerLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](K8sControlPlaneLogTaskIDPrefix + "grouper-scheduler")

// SchedulerLogToTimelineMapperTaskID is the task ID for adding events on history based on scheduler logs.
var SchedulerLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](K8sControlPlaneLogTaskIDPrefix + "timeline-mapper-scheduler")

// ControllerManagerLogFilterTaskID is the task ID for filtering controller manager logs.
var ControllerManagerLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](K8sControlPlaneLogTaskIDPrefix + "controller-manager-log-filter")

// ControllerManagerLogFieldSetReaderTaskID is the task ID for reading field sets specific to controller manager logs.
var ControllerManagerLogFieldSetReaderTaskID = taskid.NewDefaultImplementationID[[]*log.Log](K8sControlPlaneLogTaskIDPrefix + "fieldset-reader-controller-manager")

// ControllerManagerLogGrouperTaskID is the task ID for grouping controller manager logs.
var ControllerManagerLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](K8sControlPlaneLogTaskIDPrefix + "grouper-controller-manager")

// ControllerManagerLogToTimelineMapperTaskID is the task ID for adding events on history based on controller manager logs.
var ControllerManagerLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](K8sControlPlaneLogTaskIDPrefix + "timeline-mapper-controller-manager")

// OtherLogFilterTaskID is the task ID for filtering logs from other control plane components.
var OtherLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](K8sControlPlaneLogTaskIDPrefix + "other-log-filter")

// OtherLogFieldSetReaderTaskID is the task ID for reading field sets specific to other control plane component logs.
var OtherLogFieldSetReaderTaskID = taskid.NewDefaultImplementationID[[]*log.Log](K8sControlPlaneLogTaskIDPrefix + "fieldset-reader-other")

// OtherLogGrouperTaskID is the task ID for grouping logs from other control plane components.
var OtherLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](K8sControlPlaneLogTaskIDPrefix + "grouper-other")

// OtherLogToTimelineMapperTaskID is the task ID for adding events on history based on the other control plane components.
var OtherLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](K8sControlPlaneLogTaskIDPrefix + "timeline-mapper-other")

// TailTaskID is the task ID for the final task in the control plane log processing pipeline.
var TailTaskID = taskid.NewDefaultImplementationID[struct{}](K8sControlPlaneLogTaskIDPrefix + "tail")
