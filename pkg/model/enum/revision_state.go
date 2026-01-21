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

type RevisionStateStyle int

const (
	RevisionStateStyleNormal      RevisionStateStyle = 0
	RevisionStateStyleDeleted     RevisionStateStyle = 1
	RevisionStateStylePartialInfo RevisionStateStyle = 2
)

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

	RevisionAutoscalerNoError   RevisionState = 30 // Added since 0.49
	RevisionAutoscalerHasErrors RevisionState = 31

	RevisionStateConditionNotGiven        RevisionState = 32 // Added since 0.50
	RevisionStateConditionNoAvailableInfo RevisionState = 33 // Added since 0.50

	RevisionStatePodPhasePending   RevisionState = 34 // Added since 0.50
	RevisionStatePodPhaseScheduled RevisionState = 35 // Added since 0.50
	RevisionStatePodPhaseRunning   RevisionState = 36 // Added since 0.50
	RevisionStatePodPhaseSucceeded RevisionState = 37 // Added since 0.50
	RevisionStatePodPhaseFailed    RevisionState = 38 // Added since 0.50
	RevisionStatePodPhaseUnknown   RevisionState = 39 // Added since 0.50

	RevisionStateContainerStatusNotAvailable RevisionState = 40 // Added since 0.50
	RevisionStateContainerStarted            RevisionState = 41 // Added since 0.50

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
	BackgroundColor HDRColor4

	// Icon is used for rendering the icon in timeline view.
	Icon string

	// Style decides non color styling of the revision like stripes depending on its revision purpose.
	Style RevisionStateStyle
}

var RevisionStates = map[RevisionState]RevisionStateFrontendMetadata{
	RevisionStateInferred: {
		EnumKeyName:     "RevisionStateInferred",
		BackgroundColor: mustHexToHDRColor4("#999922"),
		CSSSelector:     "inferred",
		Label:           "Resource may be existing",
		Icon:            "unknown_document",
		Style:           RevisionStateStylePartialInfo,
	},
	RevisionStateExisting: {
		EnumKeyName:     "RevisionStateExisting",
		BackgroundColor: mustHexToHDRColor4("#0000FF"),
		CSSSelector:     "existing",
		Label:           "Resource is existing",
		Icon:            "deployed_code",
	},
	RevisionStateDeleted: {
		EnumKeyName:     "RevisionStateDeleted",
		BackgroundColor: mustHexToHDRColor4("#CC0000"),
		CSSSelector:     "deleted",
		Label:           "Resource is deleted",
		Icon:            "delete_forever",
		Style:           RevisionStateStyleDeleted,
	},
	RevisionStateConditionTrue: {
		EnumKeyName:     "RevisionStateConditionTrue",
		BackgroundColor: mustHexToHDRColor4("#004400"),
		CSSSelector:     "condition_true",
		Label:           "State is 'True'",
		Icon:            "lightbulb",
	},
	RevisionStateConditionFalse: {
		EnumKeyName:     "RevisionStateConditionFalse",
		BackgroundColor: mustHexToHDRColor4("#EE4400"),
		CSSSelector:     "condition_false",
		Label:           "State is 'False'",
		Icon:            "light_off",
	},
	RevisionStateConditionUnknown: {
		EnumKeyName:     "RevisionStateConditionUnknown",
		BackgroundColor: mustHexToHDRColor4("#663366"),
		CSSSelector:     "condition_unknown",
		Label:           "State is 'Unknown'",
		Icon:            "siren_question",
	},
	RevisionStateOperationStarted: {
		EnumKeyName:     "RevisionStateOperationStarted",
		BackgroundColor: mustHexToHDRColor4("#004400"),
		CSSSelector:     "operation_started",
		Label:           "Processing operation",
		Icon:            "change_circle",
	},
	RevisionStateOperationFinished: {
		EnumKeyName:     "RevisionStateOperationFinished",
		BackgroundColor: mustHexToHDRColor4("#333333"),
		CSSSelector:     "operation_finished",
		Label:           "Operation is finished",
		Icon:            "check_circle",
		Style:           RevisionStateStyleDeleted,
	},
	RevisionStateContainerWaiting: {
		EnumKeyName:     "RevisionStateContainerWaiting",
		BackgroundColor: mustHexToHDRColor4("#4444ff"),
		CSSSelector:     "container_waiting",
		Label:           "Waiting for starting container",
		Icon:            "deployed_code_history",
		Style:           RevisionStateStyleDeleted,
	},
	RevisionStateContainerRunningNonReady: {
		EnumKeyName:     "RevisionStateContainerRunningNonReady",
		BackgroundColor: mustHexToHDRColor4("#EE4400"),
		CSSSelector:     "container_running_non_ready",
		Label:           "Container is not ready",
		Icon:            "heart_broken",
	},
	RevisionStateContainerRunningReady: {
		EnumKeyName:     "RevisionStateContainerRunningReady",
		BackgroundColor: mustHexToHDRColor4("#007700"),
		CSSSelector:     "container_running_ready",
		Label:           "Container is ready",
		Icon:            "heart_check",
	},
	RevisionStateContainerTerminatedWithSuccess: {
		EnumKeyName:     "RevisionStateContainerTerminatedWithSuccess",
		BackgroundColor: mustHexToHDRColor4("#113333"),
		CSSSelector:     "container_terminated_success",
		Label:           "Container exited with healthy exit code",
		Style:           RevisionStateStyleDeleted,
		Icon:            "check_circle",
	},
	RevisionStateContainerTerminatedWithError: {
		EnumKeyName:     "RevisionStateContainerTerminatedWithError",
		BackgroundColor: mustHexToHDRColor4("#551111"),
		CSSSelector:     "container_terminated_error",
		Label:           "Container exited with erroneous exit code",
		Style:           RevisionStateStyleDeleted,
		Icon:            "error",
	},
	// Cloud Composer
	RevisionStateComposerTiScheduled: {
		EnumKeyName:     "RevisionStateComposerTiScheduled",
		BackgroundColor: mustHexToHDRColor4("#d1b48c"),
		CSSSelector:     "composer_ti_scheduled",
		Label:           "Task instance is scheduled",
	},
	RevisionStateComposerTiQueued: {
		EnumKeyName:     "RevisionStateComposerTiQueued",
		BackgroundColor: mustHexToHDRColor4("#808080"),
		CSSSelector:     "composer_ti_queued",
		Label:           "Task instance is queued",
	},
	RevisionStateComposerTiRunning: {
		EnumKeyName:     "RevisionStateComposerTiRunning",
		BackgroundColor: mustHexToHDRColor4("#00ff01"),
		CSSSelector:     "composer_ti_running",
		Label:           "Task instance is running",
	},
	RevisionStateComposerTiDeferred: {
		EnumKeyName:     "RevisionStateComposerTiDeferred",
		BackgroundColor: mustHexToHDRColor4("#9470dc"),
		CSSSelector:     "composer_ti_deferred",
		Label:           "Task instance is deferrd",
	},
	RevisionStateComposerTiSuccess: {
		EnumKeyName:     "RevisionStateComposerTiSuccess",
		BackgroundColor: mustHexToHDRColor4("#008001"),
		CSSSelector:     "composer_ti_success",
		Label:           "Task instance completed with success state",
	},
	RevisionStateComposerTiFailed: {
		EnumKeyName:     "RevisionStateComposerTiFailed",
		BackgroundColor: mustHexToHDRColor4("#fe0000"),
		CSSSelector:     "composer_ti_failed",
		Label:           "Task instance completed with errournous state",
	},
	RevisionStateComposerTiUpForRetry: {
		EnumKeyName:     "RevisionStateComposerTiUpForRetry",
		BackgroundColor: mustHexToHDRColor4("#fed700"),
		CSSSelector:     "composer_ti_up_for_retry",
		Label:           "Task instance is waiting for next retry",
	},
	RevisionStateComposerTiRestarting: {
		EnumKeyName:     "RevisionStateComposerTiRestarting",
		BackgroundColor: mustHexToHDRColor4("#ee82ef"),
		CSSSelector:     "composer_ti_restarting",
		Label:           "Task instance is being restarted",
	},
	RevisionStateComposerTiRemoved: {
		EnumKeyName:     "RevisionStateComposerTiRemoved",
		BackgroundColor: mustHexToHDRColor4("#d3d3d3"),
		CSSSelector:     "composer_ti_removed",
		Label:           "Task instance is removed",
	},
	RevisionStateComposerTiUpstreamFailed: {
		EnumKeyName:     "RevisionStateComposerTiUpstreamFailed",
		BackgroundColor: mustHexToHDRColor4("#ffa11b"),
		CSSSelector:     "composer_ti_upstream_failed",
		Label:           "Upstream of this task is failed",
	},
	RevisionStateComposerTiZombie: {
		EnumKeyName:     "RevisionStateComposerTiZombie",
		BackgroundColor: mustHexToHDRColor4("#4b0082"),
		CSSSelector:     "composer_ti_zombie",
		Label:           "Task instance is being zombie",
	},
	RevisionStateComposerTiUpForReschedule: {
		EnumKeyName:     "RevisionStateComposerTiUpForReschedule",
		BackgroundColor: mustHexToHDRColor4("#808080"),
		CSSSelector:     "composer_ti_up_for_reschedule",
		Label:           "Task instance is waiting for being rescheduled",
	},
	RevisionStateDeleting: {
		EnumKeyName:     "RevisionStateDeleting",
		BackgroundColor: mustHexToHDRColor4("#CC5500"),
		CSSSelector:     "deleting",
		Label:           "Resource is under deleting with graceful period",
		Icon:            "auto_delete",
	},
	RevisionStateEndpointReady: {
		EnumKeyName:     "RevisionStateEndpointReady",
		BackgroundColor: mustHexToHDRColor4("#004400"),
		CSSSelector:     "ready",
		Label:           "Endpoint is ready",
		Icon:            "heart_check",
	},
	RevisionStateEndpointUnready: {
		EnumKeyName:     "RevisionStateEndpointUnready",
		BackgroundColor: mustHexToHDRColor4("#EE4400"),
		CSSSelector:     "unready",
		Label:           "Endpoint is not ready",
		Icon:            "heart_broken",
	},
	RevisionStateEndpointTerminating: {
		EnumKeyName:     "RevisionStateEndpointTerminating",
		BackgroundColor: mustHexToHDRColor4("#cea700"),
		CSSSelector:     "terminating",
		Label:           "Endpoint is being terminated",
		Icon:            "auto_delete",
	},
	RevisionStateProvisioning: {
		EnumKeyName:     "RevisionStateProvisioning",
		BackgroundColor: mustHexToHDRColor4("#6666ff"),
		CSSSelector:     "provisioning",
		Label:           "Resource is being provisioned",
		Icon:            "deployed_code_history",
	},
	RevisionAutoscalerNoError: {
		EnumKeyName:     "RevisionAutoscalerNoError",
		BackgroundColor: mustHexToHDRColor4("#004400"),
		CSSSelector:     "autoscaler_no_error",
		Label:           "Autoscaler has no error",
		Icon:            "heart_check",
	},
	RevisionAutoscalerHasErrors: {
		EnumKeyName:     "RevisionAutoscalerHasErrors",
		BackgroundColor: mustHexToHDRColor4("#EE4400"),
		CSSSelector:     "autoscaler_has_errors",
		Label:           "Autoscaler has errors",
		Icon:            "heart_broken",
	},
	RevisionStateConditionNotGiven: {
		EnumKeyName:     "RevisionStateConditionNotGiven",
		BackgroundColor: mustHexToHDRColor4("#666666"),
		CSSSelector:     "condition_not_given",
		Label:           "Condition is not defined at this moment",
		Icon:            "select",
		Style:           RevisionStateStyleDeleted,
	},
	RevisionStateConditionNoAvailableInfo: {
		EnumKeyName:     "RevisionStateConditionNoAvailableInfo",
		BackgroundColor: mustHexToHDRColor4("#997700"),
		CSSSelector:     "condition_no_available_info",
		Label:           "No enough information to show condition",
		Icon:            "unknown_document",
	},
	RevisionStatePodPhasePending: {
		EnumKeyName:     "RevisionStatePodPhasePending",
		BackgroundColor: mustHexToHDRColor4("#666666"),
		CSSSelector:     "pod_phase_pending",
		Label:           "Pod is pending",
		Icon:            "hourglass_empty",
	},
	RevisionStatePodPhaseScheduled: {
		EnumKeyName:     "RevisionStatePodPhaseScheduled",
		BackgroundColor: mustHexToHDRColor4("#4444ff"),
		CSSSelector:     "pod_phase_scheduled",
		Label:           "Pod is scheduled",
		Icon:            "schedule",
	},
	RevisionStatePodPhaseRunning: {
		EnumKeyName:     "RevisionStatePodPhaseRunning",
		BackgroundColor: mustHexToHDRColor4("#004400"),
		CSSSelector:     "pod_phase_running",
		Label:           "Pod is running",
		Icon:            "motion_play",
	},
	RevisionStatePodPhaseSucceeded: {
		EnumKeyName:     "RevisionStatePodPhaseSucceeded",
		BackgroundColor: mustHexToHDRColor4("#113333"),
		CSSSelector:     "pod_phase_succeeded",
		Label:           "Pod is succeeded",
		Icon:            "check_circle",
		Style:           RevisionStateStyleDeleted,
	},
	RevisionStatePodPhaseFailed: {
		EnumKeyName:     "RevisionStatePodPhaseFailed",
		BackgroundColor: mustHexToHDRColor4("#331111"),
		CSSSelector:     "pod_phase_failed",
		Label:           "Pod is failed",
		Icon:            "error",
		Style:           RevisionStateStyleDeleted,
	},
	RevisionStatePodPhaseUnknown: {
		EnumKeyName:     "RevisionStatePodPhaseUnknown",
		BackgroundColor: mustHexToHDRColor4("#997700"),
		CSSSelector:     "pod_phase_unknown",
		Label:           "Pod status is not available from current log range",
		Style:           RevisionStateStylePartialInfo,
		Icon:            "unknown_document",
	},
	RevisionStateContainerStatusNotAvailable: {
		EnumKeyName:     "RevisionStateContainerStatusNotAvailable",
		BackgroundColor: mustHexToHDRColor4("#666666"),
		CSSSelector:     "container_status_not_available",
		Label:           "Container status is not available",
		Style:           RevisionStateStylePartialInfo,
		Icon:            "unknown_document",
	},
	RevisionStateContainerStarted: {
		EnumKeyName:     "RevisionStateContainerStarted",
		BackgroundColor: mustHexToHDRColor4("#997700"),
		CSSSelector:     "container_started",
		Label:           "Container is started but readiness info is not available",
		Style:           RevisionStateStylePartialInfo,
		Icon:            "siren_question",
	},
}
