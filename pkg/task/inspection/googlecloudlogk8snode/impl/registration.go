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

/*
This package provides inspection tasks for Kubernetes node logs from Google Cloud Logging.
The following is a Mermaid graph of the task dependencies within this package.

```mermaid
graph TD

	subgraph Inputs
	    InputProjectId
	    InputClusterName
	    InputNodeNameFilter
	end

	subgraph Log Fetching
	    ListLogEntries
	end

	subgraph Common Processing
	    LogSerializer
	    CommonFieldSetReader
	end

	subgraph Containerd Pipeline
	    ContainerdLogFilter
	    ContainerdLogGroup
	    ContainerdIDDiscovery
	    ContainerdHistoryModifier
	end

	subgraph Kubelet Pipeline
	    KubeletLogFilter
	    KubeletLogGroup
	    KubeletHistoryModifier
	end

	subgraph Other Pipeline
	    OtherLogFilter
	    OtherLogGroup
	    OtherHistoryModifier
	end

	subgraph Finalization
	    Tail
	end

	%% Input Dependencies
	InputProjectId --> ListLogEntries
	InputClusterName --> ListLogEntries
	InputNodeNameFilter --> ListLogEntries

	%% Common Processing Dependencies
	ListLogEntries --> LogSerializer
	ListLogEntries --> CommonFieldSetReader

	%% Containerd Pipeline Dependencies
	CommonFieldSetReader --> ContainerdLogFilter
	ContainerdLogFilter --> ContainerdLogGroup
	ContainerdLogFilter --> ContainerdIDDiscovery
	ContainerdIDDiscovery --> ContainerdHistoryModifier
	ContainerdLogGroup --> ContainerdHistoryModifier
	LogSerializer --> ContainerdHistoryModifier

	%% Kubelet Pipeline Dependencies
	CommonFieldSetReader --> KubeletLogFilter
	KubeletLogFilter --> KubeletLogGroup
	ContainerdIDDiscovery --> KubeletHistoryModifier
	KubeletLogGroup --> KubeletHistoryModifier
	LogSerializer --> KubeletHistoryModifier

	%% Other Pipeline Dependencies
	CommonFieldSetReader --> OtherLogFilter
	OtherLogFilter --> OtherLogGroup
	OtherLogGroup --> OtherHistoryModifier
	LogSerializer --> OtherHistoryModifier

	%% Finalization
	ContainerdHistoryModifier --> Tail
	KubeletHistoryModifier --> Tail
	OtherHistoryModifier --> Tail

```
*/
package googlecloudlogk8snode_impl

import (
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

// Register registers all googlecloudlogk8snode inspection tasks to the registry.
func Register(registry coreinspection.InspectionTaskRegistry) error {
	return coretask.RegisterTasks(registry,
		ListLogEntriesTask,
		LogSerializerTask,
		CommonFieldSetReaderTask,
		ContainerdLogFilterTask,
		ContainerdLogGroupTask,
		PodSandboxIDDiscoveryTask,
		ContainerdNodeLogHistoryModifierTask,
		KubeletLogFilterTask,
		KubeletLogGroupTask,
		KubeletLogHistoryModifierTask,
		OtherLogFilterTask,
		OtherLogGroupTask,
		OtherLogHistoryModifierTask,
		TailTask,
		ContainerIDDiscoveryTask,
	)
}
