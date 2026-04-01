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
	RelationshipPodBinding            ParentRelationship = 7 // Deprecated and replaced by PodPhase
	RelationshipNetworkEndpointGroup  ParentRelationship = 8
	RelationshipManagedInstanceGroup  ParentRelationship = 9
	RelationshipControlPlaneComponent ParentRelationship = 10
	RelationshipSerialPort            ParentRelationship = 11
	RelationshipAirflowTaskInstance   ParentRelationship = 12
	RelationshipCSMAccessLog          ParentRelationship = 13 // Added since 0.49
	RelationshipPodPhase              ParentRelationship = 14 // Added since 0.50
	relationshipUnusedEnd                                     // Add items above. This field is used for counting items in this enum to test.
)

// EnumParentRelationshipLength is the count of ParentRelationship enum elements.
const EnumParentRelationshipLength = int(relationshipUnusedEnd) + 1

// parentRelationshipFrontendMetadata is a type defined for each parent relationship types.
type ParentRelationshipFrontendMetadata struct {
	// Visible is a flag if this relationship is visible as a chip left of timeline name.
	Visible bool
	// EnumKeyName is the name of enum exactly matching with the constant variable defined in this file.
	EnumKeyName string
	// Label is a short name shown on frontend as the chip on the left of timeline name.
	Label string
	// Hint explains the meaning of this timeline. This is shown as the tooltip on front end.
	Hint                 string
	LabelColor           HDRColor4
	LabelBackgroundColor HDRColor4
	SortPriority         int
}

var ParentRelationships = map[ParentRelationship]ParentRelationshipFrontendMetadata{
	RelationshipChild: {
		Visible:              false,
		EnumKeyName:          "RelationshipChild",
		Label:                "resource",
		LabelColor:           mustHexToHDRColor4("#000000"),
		LabelBackgroundColor: mustHexToHDRColor4("#CCCCCC"),
		SortPriority:         1000,
		Hint:                 "General resource lifecycle and logs",
	},
	RelationshipResourceCondition: {
		Visible:              true,
		EnumKeyName:          "RelationshipResourceCondition",
		Label:                "condition",
		LabelColor:           mustHexToHDRColor4("#FFFFFF"),
		LabelBackgroundColor: mustHexToHDRColor4("#4c29e8"),
		Hint:                 "Resource conditions from .status.conditions",
		SortPriority:         2000,
	},
	RelationshipOperation: {
		Visible:              true,
		EnumKeyName:          "RelationshipOperation",
		Label:                "operation",
		LabelColor:           mustHexToHDRColor4("#FFFFFF"),
		LabelBackgroundColor: mustHexToHDRColor4("#000000"),
		Hint:                 "GCP operations associated with this resource",
		SortPriority:         3000,
	},
	RelationshipEndpointSlice: {
		Visible:              true,
		EnumKeyName:          "RelationshipEndpointSlice",
		Label:                "endpoint", // renamed from "endpointslice" in 0.50.0
		LabelColor:           mustHexToHDRColor4("#FFFFFF"),
		LabelBackgroundColor: mustHexToHDRColor4("#008000"),
		Hint:                 "Pod serving status from EndpointSlice",
		SortPriority:         20000, // later than container
	},
	RelationshipContainer: {
		Visible:              true,
		EnumKeyName:          "RelationshipContainer",
		Label:                "container",
		LabelColor:           mustHexToHDRColor4("#000000"),
		LabelBackgroundColor: mustHexToHDRColor4("#fe9bab"),
		Hint:                 "Container status and logs",
		SortPriority:         5000,
	},
	RelationshipNodeComponent: {
		Visible:              true,
		EnumKeyName:          "RelationshipNodeComponent",
		Label:                "node-component",
		LabelColor:           mustHexToHDRColor4("#FFFFFF"),
		LabelBackgroundColor: mustHexToHDRColor4("#0077CC"),
		Hint:                 "Non-containerized component on the node",
		SortPriority:         6000,
	},
	RelationshipOwnerReference: {
		Visible:              true,
		EnumKeyName:          "RelationshipOwnerReference",
		Label:                "owns",
		LabelColor:           mustHexToHDRColor4("#000000"),
		LabelBackgroundColor: mustHexToHDRColor4("#33DD88"),
		Hint:                 "Child resource from .metadata.ownerReferences",
		SortPriority:         7000,
	},
	RelationshipPodBinding: {
		Visible:              true,
		EnumKeyName:          "RelationshipPodBinding",
		Label:                "binds",
		LabelColor:           mustHexToHDRColor4("#000000"),
		LabelBackgroundColor: mustHexToHDRColor4("#FF8855"),
		Hint:                 "Pod binding subresource associated with this node",
		SortPriority:         8000,
	},
	RelationshipNetworkEndpointGroup: {
		Visible:              true,
		EnumKeyName:          "RelationshipNetworkEndpointGroup",
		Label:                "neg",
		LabelColor:           mustHexToHDRColor4("#FFFFFF"),
		LabelBackgroundColor: mustHexToHDRColor4("#A52A2A"),
		Hint:                 "Associated NEG serving status",
		SortPriority:         20500, // later than endpoint slice
	},
	RelationshipManagedInstanceGroup: {
		Visible:              true,
		EnumKeyName:          "RelationshipManagedInstanceGroup",
		Label:                "mig",
		LabelColor:           mustHexToHDRColor4("#FFFFFF"),
		LabelBackgroundColor: mustHexToHDRColor4("#FF5555"),
		Hint:                 "MIG logs for the parent node pool",
		SortPriority:         10000,
	},
	RelationshipControlPlaneComponent: {
		Visible:              true,
		EnumKeyName:          "RelationshipControlPlaneComponent",
		Label:                "controlplane",
		LabelColor:           mustHexToHDRColor4("#FFFFFF"),
		LabelBackgroundColor: mustHexToHDRColor4("#FF5555"),
		Hint:                 "Control plane component of the cluster",
		SortPriority:         11000,
	},
	RelationshipSerialPort: {
		Visible:              true,
		EnumKeyName:          "RelationshipSerialPort",
		Label:                "serialport",
		LabelColor:           mustHexToHDRColor4("#FFFFFF"),
		LabelBackgroundColor: mustHexToHDRColor4("#333333"),
		Hint:                 "Serial port logs of the node",
		SortPriority:         1500, // in the middle of direct children and status.
	},
	RelationshipAirflowTaskInstance: {
		Visible:              true,
		EnumKeyName:          "RelationshipAirflowTaskInstance",
		Label:                "task",
		LabelColor:           mustHexToHDRColor4("#FFFFFF"),
		LabelBackgroundColor: mustHexToHDRColor4("#377e22"),
		Hint:                 "Airflow Task Instance execution state",
		SortPriority:         1501,
	},
	RelationshipCSMAccessLog: {
		Visible:              true,
		EnumKeyName:          "RelationshipCSMAccessLog",
		Label:                "csm",
		LabelColor:           mustHexToHDRColor4("#FFFFFF"),
		LabelBackgroundColor: mustHexToHDRColor4("#FF8500"),
		Hint:                 "CSM Access logs related to this resource",
		SortPriority:         5001, // just under container logs
	},
	RelationshipPodPhase: {
		Visible:              true,
		EnumKeyName:          "RelationshipPodPhase",
		Label:                "pod",
		LabelColor:           mustHexToHDRColor4("#FFFFFF"),
		LabelBackgroundColor: mustHexToHDRColor4("#FF8855"),
		Hint:                 "Pod status on the node from .status.phase",
		SortPriority:         8000, // just under container logs
	},
}
