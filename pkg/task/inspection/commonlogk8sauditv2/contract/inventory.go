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

package commonlogk8sauditv2_contract

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
)

var NodeNameInventoryTaskID = taskid.NewDefaultImplementationID[[]string](TaskIDPrefix + "node-name-inventory")

// NodeNameInventoryBuilder is the inventory tasks builder for gathering node names
var NodeNameInventoryBuilder = inspectiontaskbase.NewInventoryTaskBuilder[[]string](NodeNameInventoryTaskID)

type UIDToResourceIdentity = map[string]*ResourceIdentity

var ResourceUIDInventoryTaskID = taskid.NewDefaultImplementationID[UIDToResourceIdentity](TaskIDPrefix + "resource-uid-inventory")

// ResourceUIDInventoryBuilder is the inventory tasks builder for gathering resource uids
var ResourceUIDInventoryBuilder = inspectiontaskbase.NewInventoryTaskBuilder[UIDToResourceIdentity](ResourceUIDInventoryTaskID)

type ContainerIDToContainerIdentity = map[string]*ContainerIdentity

var ContainerIDInventoryTaskID = taskid.NewDefaultImplementationID[ContainerIDToContainerIdentity](TaskIDPrefix + "container-id-inventory")

// ContainerIDInventoryBuilder is the inventory tasks builder for gathering the relationship between container id and container identity
var ContainerIDInventoryBuilder = inspectiontaskbase.NewInventoryTaskBuilder[ContainerIDToContainerIdentity](ContainerIDInventoryTaskID)

type IPLeaseHistory = *resourcelease.ResourceLeaseHistory[*ResourceIdentity]

var IPLeaseHistoryInventoryTaskID = taskid.NewDefaultImplementationID[IPLeaseHistory](TaskIDPrefix + "ip-lease-history-inventory")

var IPLeaseHistoryInventoryBuilder = inspectiontaskbase.NewInventoryTaskBuilder[IPLeaseHistory](IPLeaseHistoryInventoryTaskID)
