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

package enum

type ParentRelationship int

const (
	RelationshipChild                 ParentRelationship = 0
	RelationshipResourceCondition     ParentRelationship = 1
	RelationshipOperation             ParentRelationship = 2
	RelationshipEndpointSlice         ParentRelationship = 3
	RelationshipContainer             ParentRelationship = 4
	RelationshipNodeComponent         ParentRelationship = 5
	RelationshipOwnerReference        ParentRelationship = 6
	RelationshipPodBinding            ParentRelationship = 7
	RelationshipNetworkEndpointGroup  ParentRelationship = 8
	RelationshipManagedInstanceGroup  ParentRelationship = 9
	RelationshipControlPlaneComponent ParentRelationship = 10
	RelationshipSerialPort            ParentRelationship = 11
	relationshipUnusedEnd                                // Add items above. This field is used for counting items in this enum to test.
)

// parentRelationshipFrontendMetadata is a type defined for each parent relationship types.
type ParentRelationshipFrontendMetadata struct {
	Visible              bool
	EnumKeyName          string
	Label                string
	Hint                 string
	LabelColor           string
	LabelBackgroundColor string
	SortPriority         int
}

var ParentRelationships = map[ParentRelationship]ParentRelationshipFrontendMetadata{
	RelationshipChild: {
		Visible:              false,
		EnumKeyName:          "RelationshipChild",
		Label:                "subresource",
		LabelColor:           "#000000",
		LabelBackgroundColor: "#CCCCCC",
		SortPriority:         1000,
	},
	RelationshipResourceCondition: {
		Visible:              true,
		EnumKeyName:          "RelationshipResourceCondition",
		Label:                "condition",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#4c29e8",
		Hint:                 "Resource condition written on .status.conditions",
		SortPriority:         2000,
	},
	RelationshipOperation: {
		Visible:              true,
		EnumKeyName:          "RelationshipOperation",
		Label:                "operation",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#000000",
		Hint:                 "GCP operations associated with this resource",
		SortPriority:         3000,
	},
	RelationshipEndpointSlice: {
		Visible:              true,
		EnumKeyName:          "RelationshipEndpointSlice",
		Label:                "endpointslice",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#008000",
		Hint:                 "Pod serving status obtained from endpoint slice",
		SortPriority:         20000, // later than container
	},
	RelationshipContainer: {
		Visible:              true,
		EnumKeyName:          "RelationshipContainer",
		Label:                "container",
		LabelColor:           "#000000",
		LabelBackgroundColor: "#fe9bab",
		Hint:                 "Containers statuses/logs in Pods",
		SortPriority:         5000,
	},
	RelationshipNodeComponent: {
		Visible:              true,
		EnumKeyName:          "RelationshipNodeComponent",
		Label:                "node-component",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#0077CC",
		Hint:                 "Non container resource running on a node",
		SortPriority:         6000,
	},
	RelationshipOwnerReference: {
		Visible:              true,
		EnumKeyName:          "RelationshipOwnerReference",
		Label:                "owns",
		LabelColor:           "#000000",
		LabelBackgroundColor: "#33DD88",
		Hint:                 "A k8s resource related to this resource from .metadata.ownerReference field",
		SortPriority:         7000,
	},
	RelationshipPodBinding: {
		Visible:              true,
		EnumKeyName:          "RelationshipPodBinding",
		Label:                "binds",
		LabelColor:           "#000000",
		LabelBackgroundColor: "#FF8855",
		Hint:                 "Pod binding subresource associated with this node",
		SortPriority:         8000,
	},
	RelationshipNetworkEndpointGroup: {
		Visible:              true,
		EnumKeyName:          "RelationshipNetworkEndpointGroup",
		Label:                "neg",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#A52A2A",
		Hint:                 "Pod serving status obtained from the associated NEG status",
		SortPriority:         20500, // later than endpoint slice
	},
	RelationshipManagedInstanceGroup: {
		Visible:              true,
		EnumKeyName:          "RelationshipManagedInstanceGroup",
		Label:                "mig",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#FF5555",
		Hint:                 "MIG logs associated to the parent node pool",
		SortPriority:         10000,
	},
	RelationshipControlPlaneComponent: {
		Visible:              true,
		EnumKeyName:          "RelationshipControlPlaneComponent",
		Label:                "controlplane",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#FF5555",
		Hint:                 "control plane component of the cluster",
		SortPriority:         11000,
	},
	RelationshipSerialPort: {
		Visible:              true,
		EnumKeyName:          "RelationshipSerialPort",
		Label:                "serialport",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#333333",
		Hint:                 "Serial port logs of the node",
		SortPriority:         1500, // in the middle of direct children and status.
	},
}
