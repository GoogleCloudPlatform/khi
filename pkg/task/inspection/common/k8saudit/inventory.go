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

package k8saudit

import (
	"time"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
)

// TagNodeNameDiscovery is the tag for discovery tasks producing node names.
var TagNodeNameDiscovery = coretask.NewTag[[]string]("khi.google.com/inspection/commonlogk8saudit/nodename")

// NodeNameInventoryTaskID is the task ID for the inventory task aggregating discovered node names.
var NodeNameInventoryTaskID = taskid.NewDefaultImplementationID[[]string](TaskIDPrefix + "node-name-inventory")

// TagResourceUIDDiscovery is the tag for discovery tasks producing resource UID maps.
var TagResourceUIDDiscovery = coretask.NewTag[UIDToResourceIdentity]("khi.google.com/inspection/commonlogk8saudit/resourceuid")

// ResourceUIDInventoryTaskID is the task ID for the inventory task aggregating resource UID mappings.
var ResourceUIDInventoryTaskID = taskid.NewDefaultImplementationID[UIDToResourceIdentity](TaskIDPrefix + "resource-uid-inventory")

// TagContainerIDDiscovery is the tag for discovery tasks producing container ID maps.
var TagContainerIDDiscovery = coretask.NewTag[ContainerIDToContainerIdentity]("khi.google.com/inspection/commonlogk8saudit/containerid")

// ContainerIDInventoryTaskID is the task ID for the inventory task aggregating container ID mappings.
var ContainerIDInventoryTaskID = taskid.NewDefaultImplementationID[ContainerIDToContainerIdentity](TaskIDPrefix + "container-id-inventory")

// TagIPLeaseHistoryDiscovery is the tag for discovery tasks producing IP lease histories.
var TagIPLeaseHistoryDiscovery = coretask.NewTag[IPLeaseHistory]("khi.google.com/inspection/commonlogk8saudit/iplease")

// IPLeaseHistoryInventoryTaskID is the task ID for the inventory task aggregating IP lease histories.
var IPLeaseHistoryInventoryTaskID = taskid.NewDefaultImplementationID[IPLeaseHistory](TaskIDPrefix + "ip-lease-history-inventory")

// TagTimelineCreationTimeDiscovery is the tag for discovery tasks producing creation timestamps per timeline path.
var TagTimelineCreationTimeDiscovery = coretask.NewTag[TimelineCreationTimes]("khi.google.com/inspection/commonlogk8saudit/timelinecreationtime")

// TimelineCreationTimeInventoryTaskID is the task ID for the inventory task aggregating creation timestamps per timeline path.
var TimelineCreationTimeInventoryTaskID = taskid.NewDefaultImplementationID[TimelineCreationTimes](TaskIDPrefix + "timeline-creation-time-inventory")

// UIDToResourceIdentity maps Kubernetes resource UIDs to their ResourceIdentity.
type UIDToResourceIdentity = map[string]*ResourceIdentity

// ContainerIDToContainerIdentity maps container IDs to their ContainerIdentity.
type ContainerIDToContainerIdentity = map[string]*ContainerIdentity

// IPLeaseHistory tracks IP address leases associated with Kubernetes resources over time.
type IPLeaseHistory = *resourcelease.ResourceLeaseHistory[*ResourceIdentity]

// TimelineCreationTimes maps a timeline path to its observed creation timestamps in chronological order.
type TimelineCreationTimes = map[*khifilev6.TimelinePath][]time.Time
