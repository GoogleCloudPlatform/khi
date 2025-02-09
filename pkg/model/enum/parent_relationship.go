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

// EnumParentRelationshipLength is the count of ParentRelationship enum elements.
const EnumParentRelationshipLength = int(relationshipUnusedEnd)

// parentRelationshipFrontendMetadata is a type defined for each parent relationship types.
type ParentRelationshipFrontendMetadata struct {
	// Visible is a flag if this relationship is visible as a chip left of timeline name.
	Visible bool
	// EnumKeyName is the name of enum exactly matching with the constant variable defined in this file.
	EnumKeyName string
	// Label is a short name shown on frontend as the chip on the left of timeline name.
	Label string
	// LongName is a descriptive name of the ralationship. This value is used in the document.
	LongName string
	// Hint explains the meaning of this timeline. This is shown as the tooltip on front end.
	Hint                 string
	LabelColor           string
	LabelBackgroundColor string
	SortPriority         int

	// GeneratableEvents contains the list of possible event types put on a timeline with the relationship type. This field is used for document generation.
	GeneratableEvents []GeneratableEventInfo
	// GeneratableRevisions contains the list of possible revision types put on a timeline with the relationship type. This field is used for document generation.
	GeneratableRevisions []GeneratableRevisionInfo
	// GeneratableAliasTimelineInfo contains the list of possible target timelines aliased from the timeline of this relationship. This field is used for document generation.
	GeneratableAliasTimelineInfo []GeneratableAliasTimelineInfo
}

type GeneratableEventInfo struct {
	SourceLogType LogType
	Description   string
}

type GeneratableRevisionInfo struct {
	State         RevisionState
	SourceLogType LogType
	Description   string
}

type GeneratableAliasTimelineInfo struct {
	AliasedTimelineRelationship ParentRelationship
	SourceLogType               LogType
	Description                 string
}

var ParentRelationships = map[ParentRelationship]ParentRelationshipFrontendMetadata{
	RelationshipChild: {
		Visible:              false,
		EnumKeyName:          "RelationshipChild",
		Label:                "resource",
		LongName:             "The default resource timeline",
		LabelColor:           "#000000",
		LabelBackgroundColor: "#CCCCCC",
		SortPriority:         1000,
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateExisting,
				SourceLogType: LogTypeAudit,
				Description:   "This state indicates the resource exits at the time",
			},
			{
				State:         RevisionStateDeleted,
				SourceLogType: LogTypeAudit,
				Description:   "This state indicates the resource is deleted at the time.",
			},
			{
				State:         RevisionStateDeleting,
				SourceLogType: LogTypeAudit,
				Description:   "This state indicates the resource is being deleted with grace period at the time.",
			},
		},
		GeneratableEvents: []GeneratableEventInfo{
			{
				SourceLogType: LogTypeAudit,
				Description:   "An event that related to a resource but not changing the resource. This is often an error log for an operation to the resource.",
			},
			{
				SourceLogType: LogTypeEvent,
				Description:   "An event that related to a resource",
			},
		},
	},
	RelationshipResourceCondition: {
		Visible:              true,
		EnumKeyName:          "RelationshipResourceCondition",
		Label:                "condition",
		LongName:             "Status condition field timeline",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#4c29e8",
		Hint:                 "Resource condition written on .status.conditions",
		SortPriority:         2000,
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateConditionTrue,
				SourceLogType: LogTypeAudit,
			},
			{
				State:         RevisionStateConditionFalse,
				SourceLogType: LogTypeAudit,
			},
			{
				State:         RevisionStateConditionUnknown,
				SourceLogType: LogTypeAudit,
			},
		},
	},
	RelationshipOperation: {
		Visible:              true,
		EnumKeyName:          "RelationshipOperation",
		Label:                "operation",
		LongName:             "Operation timeline",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#000000",
		Hint:                 "GCP operations associated with this resource",
		SortPriority:         3000,
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateOperationStarted,
				SourceLogType: LogTypeComputeApi,
			},
			{
				State:         RevisionStateOperationFinished,
				SourceLogType: LogTypeComputeApi,
			},
		},
	},
	RelationshipEndpointSlice: {
		Visible:              true,
		EnumKeyName:          "RelationshipEndpointSlice",
		Label:                "endpointslice",
		LongName:             "Endpoint serving state timeline",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#008000",
		Hint:                 "Pod serving status obtained from endpoint slice",
		SortPriority:         20000, // later than container
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateEndpointReady,
				SourceLogType: LogTypeAudit,
			},
			{
				State:         RevisionStateEndpointUnready,
				SourceLogType: LogTypeAudit,
			},
			{
				State:         RevisionStateEndpointTerminating,
				SourceLogType: LogTypeAudit,
			},
		},
	},
	RelationshipContainer: {
		Visible:              true,
		EnumKeyName:          "RelationshipContainer",
		Label:                "container",
		LongName:             "Container timeline",
		LabelColor:           "#000000",
		LabelBackgroundColor: "#fe9bab",
		Hint:                 "Statuses/logs of a container",
		SortPriority:         5000,
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateContainerWaiting,
				SourceLogType: LogTypeContainer,
			},
			{
				State:         RevisionStateContainerRunningNonReady,
				SourceLogType: LogTypeContainer,
			},
			{
				State:         RevisionStateContainerRunningReady,
				SourceLogType: LogTypeContainer,
			},
			{
				State:         RevisionStateContainerTerminatedWithSuccess,
				SourceLogType: LogTypeContainer,
			},
			{
				State:         RevisionStateContainerTerminatedWithError,
				SourceLogType: LogTypeContainer,
			},
		},
		GeneratableEvents: []GeneratableEventInfo{
			{
				SourceLogType: LogTypeContainer,
			},
			{
				SourceLogType: LogTypeNode,
			},
		},
	},
	RelationshipNodeComponent: {
		Visible:              true,
		EnumKeyName:          "RelationshipNodeComponent",
		Label:                "node-component",
		LongName:             "Node component timeline",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#0077CC",
		Hint:                 "Non container resource running on a node",
		SortPriority:         6000,
		GeneratableEvents: []GeneratableEventInfo{
			{
				SourceLogType: LogTypeNode,
			},
		},
	},
	RelationshipOwnerReference: {
		Visible:              true,
		EnumKeyName:          "RelationshipOwnerReference",
		Label:                "owns",
		LongName:             "Owning children timeline",
		LabelColor:           "#000000",
		LabelBackgroundColor: "#33DD88",
		Hint:                 "A k8s resource related to this resource from .metadata.ownerReference field",
		SortPriority:         7000,
		GeneratableAliasTimelineInfo: []GeneratableAliasTimelineInfo{
			{
				AliasedTimelineRelationship: RelationshipChild,
				SourceLogType:               LogTypeAudit,
				Description:                 "This timeline shows the events and revisions of the owning resources.",
			},
		},
	},
	RelationshipPodBinding: {
		Visible:              true,
		EnumKeyName:          "RelationshipPodBinding",
		Label:                "binds",
		LongName:             "Pod binding timeline",
		LabelColor:           "#000000",
		LabelBackgroundColor: "#FF8855",
		Hint:                 "Pod binding subresource associated with this node",
		SortPriority:         8000,
		GeneratableAliasTimelineInfo: []GeneratableAliasTimelineInfo{
			{
				AliasedTimelineRelationship: RelationshipChild,
				SourceLogType:               LogTypeAudit,
				Description:                 "This timeline shows the binding subresources associated on a node",
			},
		},
	},
	RelationshipNetworkEndpointGroup: {
		Visible:              true,
		EnumKeyName:          "RelationshipNetworkEndpointGroup",
		Label:                "neg",
		LongName:             "NEG timeline",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#A52A2A",
		Hint:                 "Pod serving status obtained from the associated NEG status",
		SortPriority:         20500, // later than endpoint slice
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateConditionTrue,
				SourceLogType: LogTypeNetworkAPI,
			},
			{
				State:         RevisionStateConditionFalse,
				SourceLogType: LogTypeNetworkAPI,
			},
		},
	},
	RelationshipManagedInstanceGroup: {
		Visible:              true,
		EnumKeyName:          "RelationshipManagedInstanceGroup",
		Label:                "mig",
		LongName:             "Managed instance group timeline",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#FF5555",
		Hint:                 "MIG logs associated to the parent node pool",
		SortPriority:         10000,
		GeneratableEvents: []GeneratableEventInfo{
			{
				SourceLogType: LogTypeAutoscaler,
			},
		},
	},
	RelationshipControlPlaneComponent: {
		Visible:              true,
		EnumKeyName:          "RelationshipControlPlaneComponent",
		Label:                "controlplane",
		LongName:             "Control plane component timeline",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#FF5555",
		Hint:                 "control plane component of the cluster",
		SortPriority:         11000,
		GeneratableEvents: []GeneratableEventInfo{
			{
				SourceLogType: LogTypeControlPlaneComponent,
			},
		},
	},
	RelationshipSerialPort: {
		Visible:              true,
		EnumKeyName:          "RelationshipSerialPort",
		Label:                "serialport",
		LongName:             "Serialport log timeline",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#333333",
		Hint:                 "Serial port logs of the node",
		SortPriority:         1500, // in the middle of direct children and status.
		GeneratableEvents: []GeneratableEventInfo{
			{
				SourceLogType: LogTypeSerialPort,
			},
		},
	},
}
