// Copyright 2024 Google LLC
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

package resourceinfo

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
)

// NodeResourceIDType is enums to represent the type of ids.
type NodeResourceIDType int

const (
	NodeResourceIDTypeUnknown    NodeResourceIDType = 0
	NodeResourceIDTypeContainer  NodeResourceIDType = 1
	NodeResourceIDTypePodSandbox NodeResourceIDType = 2
)

// Cluster stores resource information(node name, Pod IP,Host IP...etc) used from another parser.
// This struct must modify the own fields in thread safe.
type Cluster struct {
	IPs *resourcelease.ResourceLeaseHistory[*resourcelease.K8sResourceLeaseHolder]
	// records lease history of NEG id to ServiceNetworkEndpointGroup
	NEGs *resourcelease.ResourceLeaseHistory[*resourcelease.K8sResourceLeaseHolder]
}

func NewClusterResourceInfo() *Cluster {
	ips := resourcelease.NewResourceLeaseHistory[*resourcelease.K8sResourceLeaseHolder]()
	return &Cluster{
		IPs:  ips,
		NEGs: resourcelease.NewResourceLeaseHistory[*resourcelease.K8sResourceLeaseHolder](),
	}
}
