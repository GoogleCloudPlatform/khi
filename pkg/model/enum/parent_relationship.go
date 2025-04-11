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
	RelationshipAirflowTaskInstance   ParentRelationship = 12
	relationshipUnusedEnd                                // Add items above. This field is used for counting items in this enum to test.
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
	LabelColor           string
	LabelBackgroundColor string
	SortPriority         int

	// LongName is a descriptive name of the ralationship. This value is used in the document.
	LongName string
	// Description is a description of this timeline ralationship. This value is used in the document.
	Description string
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
		LabelColor:           "#000000",
		LabelBackgroundColor: "#CCCCCC",
		SortPriority:         1000,
		LongName:             "The default resource timeline",
		Description:          "A default timeline recording the history of Kubernetes resources",
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateInferred,
				SourceLogType: LogTypeAudit,
				Description:   "This state indicates the resource exists at the time, but this existence is inferred from the other logs later. The detailed resource information is not available.",
			},
			{
				State:         RevisionStateExisting,
				SourceLogType: LogTypeAudit,
				Description:   "This state indicates the resource exists at the time",
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
			{
				State:         RevisionStateProvisioning,
				SourceLogType: LogTypeGkeAudit,
				Description:   "This state indicates the resource is being provisioned. Currently this state is only used for cluster/nodepool status only.",
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
			{
				SourceLogType: LogTypeNode,
				Description:   "An event that related to a node resource",
			},
			{
				SourceLogType: LogTypeComputeApi,
				Description:   "An event that related to a compute resource",
			},
			{
				SourceLogType: LogTypeControlPlaneComponent,
				Description:   "A log related to the timeline resource related to control plane component",
			},
			{
				SourceLogType: LogTypeAutoscaler,
				Description:   "A log related to the Pod which triggered or prevented autoscaler",
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
		Description:          "A timeline showing the state changes on `.status.conditions` of the parent resource",
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateConditionTrue,
				SourceLogType: LogTypeAudit,
				Description:   "The condition state is `True`. **This doesn't always mean a good status** (For example, `NetworkUnreachabel` condition on a Node means a bad condition when it is `True`)",
			},
			{
				State:         RevisionStateConditionFalse,
				SourceLogType: LogTypeAudit,
				Description:   "The condition state is `False`. **This doesn't always mean a bad status** (For example, `NetworkUnreachabel` condition on a Node means a good condition when it is `False`)",
			},
			{
				State:         RevisionStateConditionUnknown,
				SourceLogType: LogTypeAudit,
				Description:   "The condition state is `Unknown`",
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
		Description:          "A timeline showing long running operation status related to the parent resource",
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateOperationStarted,
				SourceLogType: LogTypeComputeApi,
				Description:   "A long running operation is running",
			},
			{
				State:         RevisionStateOperationFinished,
				SourceLogType: LogTypeComputeApi,
				Description:   "An operation is finished at the time of left edge of this operation.",
			},
			{
				State:         RevisionStateOperationStarted,
				SourceLogType: LogTypeGkeAudit,
				Description:   "A long running operation is running",
			},
			{
				State:         RevisionStateOperationFinished,
				SourceLogType: LogTypeGkeAudit,
				Description:   "An operation is finished at the time of left edge of this operation.",
			},
			{
				State:         RevisionStateOperationStarted,
				SourceLogType: LogTypeNetworkAPI,
				Description:   "A long running operation is running",
			},
			{
				State:         RevisionStateOperationFinished,
				SourceLogType: LogTypeNetworkAPI,
				Description:   "An operation is finished at the time of left edge of this operation.",
			},
			{
				State:         RevisionStateOperationStarted,
				SourceLogType: LogTypeMulticloudAPI,
				Description:   "A long running operation is running",
			},
			{
				State:         RevisionStateOperationFinished,
				SourceLogType: LogTypeMulticloudAPI,
				Description:   "An operation is finished at the time of left edge of this operation.",
			},
			{
				State:         RevisionStateOperationStarted,
				SourceLogType: LogTypeOnPremAPI,
				Description:   "A long running operation is running",
			},
			{
				State:         RevisionStateOperationFinished,
				SourceLogType: LogTypeOnPremAPI,
				Description:   "An operation is finished at the time of left edge of this operation.",
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
		Description:          "A timeline indicates the status of endpoint related to the parent resource(Pod or Service)",
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateEndpointReady,
				SourceLogType: LogTypeAudit,
				Description:   "An endpoint associated with the parent resource is ready",
			},
			{
				State:         RevisionStateEndpointUnready,
				SourceLogType: LogTypeAudit,
				Description:   "An endpoint associated with the parent resource is not ready. Traffic shouldn't be routed during this time.",
			},
			{
				State:         RevisionStateEndpointTerminating,
				SourceLogType: LogTypeAudit,
				Description:   "An endpoint associated with the parent resource is being terminated. New traffic shouldn't be routed to this endpoint during this time, but the endpoint can still have pending requests.",
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
		Description:          "A timline of a container included in the parent timeline of a Pod",
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateContainerWaiting,
				SourceLogType: LogTypeAudit,
				Description:   "The container is not started yet and waiting for something.(Example: Pulling images, mounting volumes ...etc)",
			},
			{
				State:         RevisionStateContainerRunningNonReady,
				SourceLogType: LogTypeAudit,
				Description:   "The container is started but the readiness is not ready.",
			},
			{
				State:         RevisionStateContainerRunningReady,
				SourceLogType: LogTypeAudit,
				Description:   "The container is started and the readiness is ready",
			},
			{
				State:         RevisionStateContainerTerminatedWithSuccess,
				SourceLogType: LogTypeAudit,
				Description:   "The container is already terminated with successful exit code = 0",
			},
			{
				State:         RevisionStateContainerTerminatedWithError,
				SourceLogType: LogTypeAudit,
				Description:   "The container is already terminated with errornous exit code != 0",
			},
		},
		GeneratableEvents: []GeneratableEventInfo{
			{
				SourceLogType: LogTypeContainer,
				Description:   "A container log on stdout/etderr",
			},
			{
				SourceLogType: LogTypeNode,
				Description:   "kubelet/containerd logs associated with the container",
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
		Description:          "A component running inside of the parent timeline of a Node",
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateInferred,
				SourceLogType: LogTypeNode,
				Description:   "The component is infrred to be running because of the logs from it",
			},
			{
				State:         RevisionStateExisting,
				SourceLogType: LogTypeNode,
				Description:   "The component is running running. (Few node components supports this state because the parser knows logs on startup for specific components)",
			},
			{
				State:         RevisionStateDeleted,
				SourceLogType: LogTypeNode,
				Description:   "The component is no longer running. (Few node components supports this state because the parser knows logs on termination for specific components)",
			},
		},
		GeneratableEvents: []GeneratableEventInfo{
			{
				SourceLogType: LogTypeNode,
				Description:   "A log from the component on the log",
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
		LongName:             "Network Endpoint Group timeline",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#A52A2A",
		Hint:                 "Pod serving status obtained from the associated NEG status",
		SortPriority:         20500, // later than endpoint slice
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateConditionTrue,
				SourceLogType: LogTypeNetworkAPI,
				Description:   "indicates the NEG is already attached to the Pod.",
			},
			{
				State:         RevisionStateConditionFalse,
				SourceLogType: LogTypeNetworkAPI,
				Description:   "indicates the NEG is detached from the Pod",
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
				Description:   "Autoscaler logs associated to a MIG(e.g The mig was scaled up by the austoscaler)",
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
				Description:   "A log from the control plane component",
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
				Description:   "A serialport log from the node",
			},
		},
	},
	RelationshipAirflowTaskInstance: {
		Visible:              true,
		EnumKeyName:          "RelationshipAirflowTaskInstance",
		Label:                "task",
		LongName:             "@task(Operator, Sensor, etc)",
		LabelColor:           "#FFFFFF",
		LabelBackgroundColor: "#377e22",
		Hint:                 "Task is the basic unit of execution in Airflow",
		SortPriority:         1501,
		GeneratableRevisions: []GeneratableRevisionInfo{
			{
				State:         RevisionStateComposerTiDeferred,
				SourceLogType: LogTypeComposerEnvironment,
				Description:   "Ti.state = deferred",
			},
			{
				State:         RevisionStateComposerTiFailed,
				SourceLogType: LogTypeComposerEnvironment,
				Description:   "Ti.state = failed",
			},
			{
				State:         RevisionStateComposerTiRemoved,
				SourceLogType: LogTypeComposerEnvironment,
				Description:   "Ti.state = removed",
			},
			{
				State:         RevisionStateComposerTiRunning,
				SourceLogType: LogTypeComposerEnvironment,
				Description:   "Ti.state = running",
			},
			{
				State:         RevisionStateComposerTiScheduled,
				SourceLogType: LogTypeComposerEnvironment,
				Description:   "Ti.state = scheduled",
			},
			{
				State:         RevisionStateComposerTiSuccess,
				SourceLogType: LogTypeComposerEnvironment,
				Description:   "Ti.state = success",
			},
			{
				State:         RevisionStateComposerTiQueued,
				SourceLogType: LogTypeComposerEnvironment,
				Description:   "Ti.state = queued",
			},
			{
				State:         RevisionStateComposerTiUpForRetry,
				SourceLogType: LogTypeComposerEnvironment,
				Description:   "Ti.state = up_for_retry",
			},
			{
				State:         RevisionStateComposerTiUpForReschedule,
				SourceLogType: LogTypeComposerEnvironment,
				Description:   "Ti.state = reschedule",
			},
			{
				State:         RevisionStateComposerTiZombie,
				SourceLogType: LogTypeComposerEnvironment,
				Description:   "Ti.state = zombie",
			},
			{
				State:         RevisionStateComposerTiUpstreamFailed,
				SourceLogType: LogTypeComposerEnvironment,
				Description:   "Ti.state = upstream_failed",
			},
			{
				State:         RevisionStateComposerTiRestarting,
				SourceLogType: LogTypeControlPlaneComponent,
				Description:   "Ti.state = restarting",
			},
		},
	},
}
