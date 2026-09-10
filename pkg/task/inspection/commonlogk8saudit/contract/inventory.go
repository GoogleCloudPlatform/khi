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

package commonlogk8saudit_contract

import (
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
)

// TagNodeNameDiscovery is the tag for discovery tasks producing node names.
var TagNodeNameDiscovery = coretask.NewTag[[]string]("khi.google.com/inspection/commonlogk8saudit/nodename")

var NodeNameInventoryTaskID = taskid.NewDefaultImplementationID[[]string](TaskIDPrefix + "node-name-inventory")

type UIDToResourceIdentity = map[string]*ResourceIdentity

// TagResourceUIDDiscovery is the tag for discovery tasks producing resource UID maps.
var TagResourceUIDDiscovery = coretask.NewTag[UIDToResourceIdentity]("khi.google.com/inspection/commonlogk8saudit/resourceuid")

var ResourceUIDInventoryTaskID = taskid.NewDefaultImplementationID[UIDToResourceIdentity](TaskIDPrefix + "resource-uid-inventory")

type ContainerIDToContainerIdentity = map[string]*ContainerIdentity

// TagContainerIDDiscovery is the tag for discovery tasks producing container ID maps.
var TagContainerIDDiscovery = coretask.NewTag[ContainerIDToContainerIdentity]("khi.google.com/inspection/commonlogk8saudit/containerid")

var ContainerIDInventoryTaskID = taskid.NewDefaultImplementationID[ContainerIDToContainerIdentity](TaskIDPrefix + "container-id-inventory")

type IPLeaseHistory = *resourcelease.ResourceLeaseHistory[*ResourceIdentity]

// TagIPLeaseHistoryDiscovery is the tag for discovery tasks producing IP lease histories.
var TagIPLeaseHistoryDiscovery = coretask.NewTag[IPLeaseHistory]("khi.google.com/inspection/commonlogk8saudit/iplease")

var IPLeaseHistoryInventoryTaskID = taskid.NewDefaultImplementationID[IPLeaseHistory](TaskIDPrefix + "ip-lease-history-inventory")
