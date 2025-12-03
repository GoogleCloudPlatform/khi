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

// Package googlecloudlogk8snode_contract defines the contract for the googlecloudlogk8snode task.
package googlecloudlogk8snode_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

const (
	// TaskIDPrefix is the prefix for all task IDs in this package.
	TaskIDPrefix = "cloud.google.com/log/k8s-node/"
)

// ListLogEntriesTaskID is the task id for the task that queries k8s node logs from Cloud Logging.
var ListLogEntriesTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "query")

// LogSerializerTaskID is the task ID to finalize the logs to be included in the final output.
var LogSerializerTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "log-serializer")

// CommonFieldsetReaderTaskID is the ID for a task to read the fieldset used by all parsers in node log parsers later.
var CommonFieldsetReaderTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "common-fieldset-reader")

// ContainerdLogFilterTaskID is the ID for a task to filter only the logs for containerd.
var ContainerdLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "containerd-log-filter")

// ContainerdLogGroupTaskID is the ID for a task to group containerd related logs based on instance names.
var ContainerdLogGroupTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "containerd-log-group")

// PodSandboxIDDiscoveryTaskID is the ID for a task to extract pod sandbox IDs for the other parsers to corelate a log to Pods.
var PodSandboxIDDiscoveryTaskID = taskid.NewDefaultImplementationID[patternfinder.PatternFinder[*PodSandboxIDInfo]](TaskIDPrefix + "containerd-id-discovery")

// ContainerdLogHistoryModifierTaskID is the ID for a task to add events or revisions based on containerd logs.
var ContainerdLogHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "containerd-log-history-modifier")

// KubeletLogFilterTaskID is the ID for a task to filter only the logs for kubelet.
var KubeletLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "kubelet-log-filter")

// KubeletLogGroupTaskID is the ID for a task to group kubelet related logs based on instance names.
var KubeletLogGroupTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "kubelet-log-group")

// KubeletLogHistoryModifierTaskID is the ID for a task to add events or revisions based on kubelet logs.
var KubeletLogHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "kubelet-log-history-modifier")

// OtherLogFilterTaskID is the task ID for filtering other logs.
var OtherLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "other-log-filter")

// OtherLogGroupTaskID is the ID for a task to group other related logs based on instance names and component name.
var OtherLogGroupTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "other-log-group")

// OtherLogHistoryModifierTaskID is the task ID for a task to add events or revisions based on other logs.
var OtherLogHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "other-log-history-modifier")

// TailTaskID is a nop task just to require all child parsers.
var TailTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "tail")

var ContainerIDDiscoveryTaskID = taskid.NewDefaultImplementationID[commonlogk8sauditv2_contract.ContainerIDToContainerIdentity](TaskIDPrefix + "container-id-discovery")
