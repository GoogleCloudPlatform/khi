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

type RevisionState int

const (
	RevisionStateInferred RevisionState = 0
	RevisionStateExisting RevisionState = 1
	RevisionStateDeleted  RevisionState = 2

	RevisionStateConditionTrue    RevisionState = 3
	RevisionStateConditionFalse   RevisionState = 4
	RevisionStateConditionUnknown RevisionState = 5

	RevisionStateOperationStarted  RevisionState = 6
	RevisionStateOperationFinished RevisionState = 7

	RevisionStateContainerWaiting               RevisionState = 8
	RevisionStateContainerRunningNonReady       RevisionState = 9
	RevisionStateContainerRunningReady          RevisionState = 10
	RevisionStateContainerTerminatedWithSuccess RevisionState = 11
	RevisionStateContainerTerminatedWithError   RevisionState = 12

	// Cloud Composer
	RevisionStateComposerTiScheduled       RevisionState = 13
	RevisionStateComposerTiQueued          RevisionState = 14
	RevisionStateComposerTiRunning         RevisionState = 15
	RevisionStateComposerTiDeferred        RevisionState = 16
	RevisionStateComposerTiSuccess         RevisionState = 17
	RevisionStateComposerTiFailed          RevisionState = 18
	RevisionStateComposerTiUpForRetry      RevisionState = 19
	RevisionStateComposerTiRestarting      RevisionState = 20
	RevisionStateComposerTiRemoved         RevisionState = 21
	RevisionStateComposerTiUpstreamFailed  RevisionState = 22
	RevisionStateComposerTiZombie          RevisionState = 23
	RevisionStateComposerTiUpForReschedule RevisionState = 24

	RevisionStateDeleting            RevisionState = 25 // Added since 0.41
	RevisionStateEndpointReady       RevisionState = 26
	RevisionStateEndpointTerminating RevisionState = 27
	RevisionStateEndpointUnready     RevisionState = 28

	RevisionStateProvisioning RevisionState = 29 // Added since 0.42

	revisionStateUnusedEnd // Adds items above. This value is used for counting items in this enum to test.
)

type RevisionStateFrontendMetadata struct {
	// EnumKeyName is the name of this enum value. Must match with the enum key.
	EnumKeyName string

	// CSSSelector is used for CSS class name. it must be valid as the css class name
	CSSSelector string

	// Label is human readable text explaining this state.
	Label string

	// BackgroundColor is used for rendering the revision rectangles in timeline view.
	BackgroundColor string
}

var RevisionStates = map[RevisionState]RevisionStateFrontendMetadata{
	RevisionStateInferred: {
		EnumKeyName:     "RevisionStateInferred",
		BackgroundColor: "#997700",
		CSSSelector:     "inferred",
		Label:           "Resource may be existing",
	},
	RevisionStateExisting: {
		EnumKeyName:     "RevisionStateExisting",
		BackgroundColor: "#0000FF",
		CSSSelector:     "existing",
		Label:           "Resource is existing",
	},
	RevisionStateDeleted: {
		EnumKeyName:     "RevisionStateDeleted",
		BackgroundColor: "#CC0000",
		CSSSelector:     "deleted",
		Label:           "Resource is deleted",
	},
	RevisionStateConditionTrue: {
		EnumKeyName:     "RevisionStateConditionTrue",
		BackgroundColor: "#004400",
		CSSSelector:     "condition_true",
		Label:           "State is 'True'",
	},
	RevisionStateConditionFalse: {
		EnumKeyName:     "RevisionStateConditionFalse",
		BackgroundColor: "#EE4400",
		CSSSelector:     "condition_false",
		Label:           "State is 'False'",
	},
	RevisionStateConditionUnknown: {
		EnumKeyName:     "RevisionStateConditionUnknown",
		BackgroundColor: "#663366",
		CSSSelector:     "condition_unknown",
		Label:           "State is 'Unknown'",
	},
	RevisionStateOperationStarted: {
		EnumKeyName:     "RevisionStateOperationStarted",
		BackgroundColor: "#004400",
		CSSSelector:     "operation_started",
		Label:           "Processing operation",
	},
	RevisionStateOperationFinished: {
		EnumKeyName:     "RevisionStateOperationFinished",
		BackgroundColor: "#333333",
		CSSSelector:     "operation_finished",
		Label:           "Operation is finished",
	},
	RevisionStateContainerWaiting: {
		EnumKeyName:     "RevisionStateContainerWaiting",
		BackgroundColor: "#997700",
		CSSSelector:     "container_waiting",
		Label:           "Waiting for starting container",
	},
	RevisionStateContainerRunningNonReady: {
		EnumKeyName:     "RevisionStateContainerRunningNonReady",
		BackgroundColor: "#EE4400",
		CSSSelector:     "container_running_non_ready",
		Label:           "Container is not ready",
	},
	RevisionStateContainerRunningReady: {
		EnumKeyName:     "RevisionStateContainerRunningReady",
		BackgroundColor: "#007700",
		CSSSelector:     "container_running_ready",
		Label:           "Container is ready",
	},
	RevisionStateContainerTerminatedWithSuccess: {
		EnumKeyName:     "RevisionStateContainerTerminatedWithSuccess",
		BackgroundColor: "#113333",
		CSSSelector:     "container_terminated_success",
		Label:           "Container exited with healthy exit code",
	},
	RevisionStateContainerTerminatedWithError: {
		EnumKeyName:     "RevisionStateContainerTerminatedWithError",
		BackgroundColor: "#331111",
		CSSSelector:     "container_terminated_error",
		Label:           "Container exited with errornous exit code",
	},
	// Cloud Composer
	RevisionStateComposerTiScheduled: {
		EnumKeyName:     "RevisionStateComposerTiScheduled",
		BackgroundColor: "#d1b48c",
		CSSSelector:     "composer_ti_scheduled",
		Label:           "Task instance is scheduled",
	},
	RevisionStateComposerTiQueued: {
		EnumKeyName:     "RevisionStateComposerTiQueued",
		BackgroundColor: "#808080",
		CSSSelector:     "composer_ti_queued",
		Label:           "Task instance is queued",
	},
	RevisionStateComposerTiRunning: {
		EnumKeyName:     "RevisionStateComposerTiRunning",
		BackgroundColor: "#00ff01",
		CSSSelector:     "composer_ti_running",
		Label:           "Task instance is running",
	},
	RevisionStateComposerTiDeferred: {
		EnumKeyName:     "RevisionStateComposerTiDeferred",
		BackgroundColor: "#9470dc",
		CSSSelector:     "composer_ti_deferred",
		Label:           "Task instance is deferrd",
	},
	RevisionStateComposerTiSuccess: {
		EnumKeyName:     "RevisionStateComposerTiSuccess",
		BackgroundColor: "#008001",
		CSSSelector:     "composer_ti_success",
		Label:           "Task instance completed with success state",
	},
	RevisionStateComposerTiFailed: {
		EnumKeyName:     "RevisionStateComposerTiFailed",
		BackgroundColor: "#fe0000",
		CSSSelector:     "composer_ti_failed",
		Label:           "Task instance completed with errournous state",
	},
	RevisionStateComposerTiUpForRetry: {
		EnumKeyName:     "RevisionStateComposerTiUpForRetry",
		BackgroundColor: "#fed700",
		CSSSelector:     "composer_ti_up_for_retry",
		Label:           "Task instance is waiting for next retry",
	},
	RevisionStateComposerTiRestarting: {
		EnumKeyName:     "RevisionStateComposerTiRestarting",
		BackgroundColor: "#ee82ef",
		CSSSelector:     "composer_ti_restarting",
		Label:           "Task instance is being restarted",
	},
	RevisionStateComposerTiRemoved: {
		EnumKeyName:     "RevisionStateComposerTiRemoved",
		BackgroundColor: "#d3d3d3",
		CSSSelector:     "composer_ti_removed",
		Label:           "Task instance is removed",
	},
	RevisionStateComposerTiUpstreamFailed: {
		EnumKeyName:     "RevisionStateComposerTiUpstreamFailed",
		BackgroundColor: "#ffa11b",
		CSSSelector:     "composer_ti_upstream_failed",
		Label:           "Upstream of this task is failed",
	},
	RevisionStateComposerTiZombie: {
		EnumKeyName:     "RevisionStateComposerTiZombie",
		BackgroundColor: "#696969",
		CSSSelector:     "composer_ti_zombie",
		Label:           "Task instance is being zombie",
	},
	RevisionStateComposerTiUpForReschedule: {
		EnumKeyName:     "RevisionStateComposerTiUpForReschedule",
		BackgroundColor: "#808080",
		CSSSelector:     "composer_ti_up_for_reschedule",
		Label:           "Task instance is waiting for being rescheduled",
	},
	RevisionStateDeleting: {
		EnumKeyName:     "RevisionStateDeleting",
		BackgroundColor: "#CC5500",
		CSSSelector:     "deleting",
		Label:           "Resource is under deleting with graceful period",
	},
	RevisionStateEndpointReady: {
		EnumKeyName:     "RevisionStateEndpointReady",
		BackgroundColor: "#004400",
		CSSSelector:     "ready",
		Label:           "Endpoint is ready",
	},
	RevisionStateEndpointUnready: {
		EnumKeyName:     "RevisionStateEndpointUnready",
		BackgroundColor: "#EE4400",
		CSSSelector:     "unready",
		Label:           "Endpoint is not ready",
	},
	RevisionStateEndpointTerminating: {
		EnumKeyName:     "RevisionStateEndpointTerminating",
		BackgroundColor: "#fed700",
		CSSSelector:     "terminating",
		Label:           "Endpoint is being terminated",
	},
	RevisionStateProvisioning: {
		EnumKeyName:     "RevisionStateProvisioning",
		BackgroundColor: "#4444ff",
		CSSSelector:     "provisioning",
		Label:           "Resource is being provisioned",
	},
}
