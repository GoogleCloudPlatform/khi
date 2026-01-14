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

type RevisionVerb int

const (
	RevisionVerbUnknown          RevisionVerb = 0
	RevisionVerbCreate           RevisionVerb = 1
	RevisionVerbDelete           RevisionVerb = 2
	RevisionVerbUpdate           RevisionVerb = 3
	RevisionVerbPatch            RevisionVerb = 4
	RevisionVerbDeleteCollection RevisionVerb = 5

	RevisionVerbReady    RevisionVerb = 6
	RevisionVerbNonReady RevisionVerb = 7

	RevisionVerbOperationStart  RevisionVerb = 8
	RevisionVerbOperationFinish RevisionVerb = 9

	RevisionVerbStatusUnknown RevisionVerb = 10
	RevisionVerbStatusTrue    RevisionVerb = 11
	RevisionVerbStatusFalse   RevisionVerb = 12

	RevisionVerbContainerWaiting  RevisionVerb = 13
	RevisionVerbContainerReady    RevisionVerb = 14
	RevisionVerbContainerNonReady RevisionVerb = 15
	RevisionVerbContainerSuccess  RevisionVerb = 16
	RevisionVerbContainerError    RevisionVerb = 17

	RevisionVerbComposerTaskInstanceScheduled       RevisionVerb = 18
	RevisionVerbComposerTaskInstanceQueued          RevisionVerb = 19
	RevisionVerbComposerTaskInstanceRunning         RevisionVerb = 20
	RevisionVerbComposerTaskInstanceUpForRetry      RevisionVerb = 21
	RevisionVerbComposerTaskInstanceSuccess         RevisionVerb = 22
	RevisionVerbComposerTaskInstanceFailed          RevisionVerb = 23
	RevisionVerbComposerTaskInstanceDeferred        RevisionVerb = 24
	RevisionVerbComposerTaskInstanceUpForReschedule RevisionVerb = 25
	RevisionVerbComposerTaskInstanceRemoved         RevisionVerb = 26
	RevisionVerbComposerTaskInstanceUpstreamFailed  RevisionVerb = 27
	RevisionVerbComposerTaskInstanceZombie          RevisionVerb = 28
	RevisionVerbComposerTaskInstanceStats           RevisionVerb = 29
	RevisionVerbComposerTaskInstanceUnimplemented   RevisionVerb = 30

	RevisionVerbTerminating RevisionVerb = 31 // Added since 0.41 for endpoint slice

	revisionVerbUnusedEnd // Adds items above. This value is used for counting items in this enum to test.
)

type RevisionVerbFrontendMetadata struct {
	// EnumKeyName is the name of this enum value. Must match with the enum key.
	EnumKeyName string
	// Label string shown on frontnend to indicate the verb.
	Label       string
	CSSSelector string
	// Background color of the label on log pane and the diamond shape on timeline view.
	LabelBackgroundColor HDRColor4
}

var RevisionVerbs = map[RevisionVerb]RevisionVerbFrontendMetadata{
	RevisionVerbUnknown: {
		EnumKeyName:          "RevisionVerbUnknown",
		Label:                "Unknown",
		CSSSelector:          "unknown",
		LabelBackgroundColor: mustHexToHDRColor4("#CC33CC"),
	},
	RevisionVerbCreate: {
		EnumKeyName:          "RevisionVerbCreate",
		Label:                "Create",
		CSSSelector:          "create",
		LabelBackgroundColor: mustHexToHDRColor4("#1E88E5"),
	},
	RevisionVerbDelete: {
		EnumKeyName:          "RevisionVerbDelete",
		Label:                "Delete",
		CSSSelector:          "delete",
		LabelBackgroundColor: mustHexToHDRColor4("#F54945"),
	},
	RevisionVerbUpdate: {
		EnumKeyName:          "RevisionVerbUpdate",
		Label:                "Update",
		CSSSelector:          "update",
		LabelBackgroundColor: mustHexToHDRColor4("#FDD835"),
	},
	RevisionVerbPatch: {
		EnumKeyName:          "RevisionVerbPatch",
		Label:                "Patch",
		CSSSelector:          "patch",
		LabelBackgroundColor: mustHexToHDRColor4("#FDD835"),
	},
	RevisionVerbDeleteCollection: {
		EnumKeyName:          "RevisionVerbDeleteCollection",
		Label:                "DeleteCollection",
		CSSSelector:          "delete-collection",
		LabelBackgroundColor: mustHexToHDRColor4("#F54945"),
	},
	RevisionVerbReady: {
		EnumKeyName:          "RevisionVerbReady",
		Label:                "Ready",
		CSSSelector:          "ready",
		LabelBackgroundColor: mustHexToHDRColor4("#22CC22"),
	},
	RevisionVerbNonReady: {
		EnumKeyName:          "RevisionVerbNonReady",
		Label:                "NonReady",
		CSSSelector:          "non-ready",
		LabelBackgroundColor: mustHexToHDRColor4("#FF7700"),
	},
	RevisionVerbTerminating: {
		EnumKeyName:          "RevisionVerbTerminating",
		Label:                "Terminating",
		CSSSelector:          "terminating",
		LabelBackgroundColor: mustHexToHDRColor4("#FFAA00"),
	},
	RevisionVerbOperationStart: {
		EnumKeyName:          "RevisionVerbOperationStart",
		Label:                "Start",
		CSSSelector:          "operation-start",
		LabelBackgroundColor: mustHexToHDRColor4("#22CC22"),
	},
	RevisionVerbOperationFinish: {
		EnumKeyName:          "RevisionVerbOperationFinish",
		Label:                "Finish",
		CSSSelector:          "operation-finish",
		LabelBackgroundColor: mustHexToHDRColor4("#9999CC"),
	},
	RevisionVerbStatusUnknown: {
		EnumKeyName:          "RevisionVerbStatusUnknown",
		Label:                "Condition(Unknown)",
		CSSSelector:          "status-unknown",
		LabelBackgroundColor: mustHexToHDRColor4("#AA66AA"),
	},
	RevisionVerbStatusTrue: {
		EnumKeyName:          "RevisionVerbStatusTrue",
		Label:                "Condition(True)",
		CSSSelector:          "status-true",
		LabelBackgroundColor: mustHexToHDRColor4("#22CC22"),
	},
	RevisionVerbStatusFalse: {
		EnumKeyName:          "RevisionVerbStatusFalse",
		Label:                "Condition(False)",
		CSSSelector:          "status-false",
		LabelBackgroundColor: mustHexToHDRColor4("#FF7700"),
	},
	RevisionVerbContainerWaiting: {
		EnumKeyName:          "RevisionVerbContainerWaiting",
		Label:                "Waiting",
		LabelBackgroundColor: mustHexToHDRColor4("#FDD835"),
		CSSSelector:          "container-waiting",
	},
	RevisionVerbContainerReady: {
		EnumKeyName:          "RevisionVerbContainerReady",
		Label:                "Ready",
		LabelBackgroundColor: mustHexToHDRColor4("#22CC22"),
		CSSSelector:          "container-ready",
	},
	RevisionVerbContainerNonReady: {
		EnumKeyName:          "RevisionVerbContainerNonReady",
		Label:                "NonReady",
		LabelBackgroundColor: mustHexToHDRColor4("#FF7700"),
		CSSSelector:          "container-non-ready",
	},
	RevisionVerbContainerSuccess: {
		EnumKeyName:          "RevisionVerbContainerSuccess",
		Label:                "Success",
		LabelBackgroundColor: mustHexToHDRColor4("#007700"),
		CSSSelector:          "container-success",
	},
	RevisionVerbContainerError: {
		EnumKeyName:          "RevisionVerbContainerError",
		Label:                "Error",
		LabelBackgroundColor: mustHexToHDRColor4("#A51915"),
		CSSSelector:          "container-error",
	},
	RevisionVerbComposerTaskInstanceScheduled: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceScheduled",
		Label:                "Scheduled",
		LabelBackgroundColor: mustHexToHDRColor4("#1E88E5"),
		CSSSelector:          "composer-taskinstance-scheduled",
	},
	RevisionVerbComposerTaskInstanceQueued: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceQueued",
		Label:                "Queued",
		LabelBackgroundColor: mustHexToHDRColor4("#22CC22"),
		CSSSelector:          "composer-taskinstance-queued",
	},
	RevisionVerbComposerTaskInstanceRunning: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceRunning",
		Label:                "Running",
		LabelBackgroundColor: mustHexToHDRColor4("#22CC22"),
		CSSSelector:          "composer-taskinstance-running",
	},
	RevisionVerbComposerTaskInstanceUpForRetry: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceUpForRetry",
		Label:                "UpForRetry",
		LabelBackgroundColor: mustHexToHDRColor4("#FF7700"),
		CSSSelector:          "composer-taskinstance-upforretry",
	},
	RevisionVerbComposerTaskInstanceSuccess: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceSuccess",
		Label:                "Success",
		LabelBackgroundColor: mustHexToHDRColor4("#22CC22"),
		CSSSelector:          "composer-taskinstance-success",
	},
	RevisionVerbComposerTaskInstanceFailed: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceFailed",
		Label:                "Failed",
		LabelBackgroundColor: mustHexToHDRColor4("#A51915"),
		CSSSelector:          "composer-taskinstance-failed",
	},
	RevisionVerbComposerTaskInstanceDeferred: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceDeferred",
		Label:                "Deferred",
		LabelBackgroundColor: mustHexToHDRColor4("#9470DC"),
		CSSSelector:          "composer-taskinstance-deferred",
	},
	RevisionVerbComposerTaskInstanceUpForReschedule: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceUpForReschedule",
		Label:                "UpForReschedule",
		LabelBackgroundColor: mustHexToHDRColor4("#FF7700"),
		CSSSelector:          "composer-taskinstance-upforreschedule",
	},
	RevisionVerbComposerTaskInstanceRemoved: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceRemoved",
		Label:                "Removed",
		LabelBackgroundColor: mustHexToHDRColor4("#A51915"),
		CSSSelector:          "composer-taskinstance-removed",
	},
	RevisionVerbComposerTaskInstanceUpstreamFailed: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceUpstreamFailed",
		Label:                "UpstreamFailed",
		LabelBackgroundColor: mustHexToHDRColor4("#A51915"),
		CSSSelector:          "composer-taskinstance-upstreamfailed",
	},
	RevisionVerbComposerTaskInstanceZombie: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceZombie",
		Label:                "Zombie",
		LabelBackgroundColor: mustHexToHDRColor4("#696969"),
		CSSSelector:          "composer-taskinstance-zombie",
	},
	RevisionVerbComposerTaskInstanceStats: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceStats",
		Label:                "Stats",
		LabelBackgroundColor: mustHexToHDRColor4("#DDDDDD"),
		CSSSelector:          "composer-taskinstance-stats",
	},
	RevisionVerbComposerTaskInstanceUnimplemented: {
		EnumKeyName:          "RevisionVerbComposerTaskInstanceUnimplemented",
		Label:                "Unimplemented",
		LabelBackgroundColor: mustHexToHDRColor4("#DDDDDD"),
		CSSSelector:          "composer-taskinstance-unimplemented",
	},
}
