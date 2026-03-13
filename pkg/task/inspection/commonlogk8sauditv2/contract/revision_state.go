// Copyright 2026 Google LLC
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
	khifilev4 "github.com/GoogleCloudPlatform/khi/pkg/generated/proto/khifile/v4"
)

var (
	// Condition States
	RevisionStateConditionTrue = &khifilev4.RevisionState{
		Label:           "State is 'True'",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#004400FF"),
		Icon:            "lightbulb",
	}
	RevisionStateConditionFalse = &khifilev4.RevisionState{
		Label:           "State is 'False'",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#EE4400FF"),
		Icon:            "light_off",
	}
	RevisionStateConditionUnknown = &khifilev4.RevisionState{
		Label:           "State is 'Unknown'",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#663366FF"),
		Icon:            "siren_question",
	}
	RevisionStateConditionNotGiven = &khifilev4.RevisionState{
		Label:           "Condition is not defined at this moment",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#666666FF"),
		Icon:            "select",
		Style:           khifilev4.RevisionStateStyle_DELETED,
	}
	RevisionStateConditionNoAvailableInfo = &khifilev4.RevisionState{
		Label:           "No enough information to show condition",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#997700FF"),
		Icon:            "unknown_document",
	}

	// Container States
	RevisionStateContainerWaiting = &khifilev4.RevisionState{
		Label:           "Waiting for starting container",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#4444FFFF"),
		Icon:            "deployed_code_history",
		Style:           khifilev4.RevisionStateStyle_DELETED,
	}
	RevisionStateContainerRunningNonReady = &khifilev4.RevisionState{
		Label:           "Container is not ready",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#EE4400FF"),
		Icon:            "heart_broken",
	}
	RevisionStateContainerRunningReady = &khifilev4.RevisionState{
		Label:           "Container is ready",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#007700FF"),
		Icon:            "heart_check",
	}
	RevisionStateContainerTerminatedWithSuccess = &khifilev4.RevisionState{
		Label:           "Container exited with healthy exit code",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#113333FF"),
		Icon:            "check_circle",
		Style:           khifilev4.RevisionStateStyle_DELETED,
	}
	RevisionStateContainerTerminatedWithError = &khifilev4.RevisionState{
		Label:           "Container exited with erroneous exit code",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#551111FF"),
		Icon:            "error",
		Style:           khifilev4.RevisionStateStyle_DELETED,
	}
	RevisionStateContainerStatusNotAvailable = &khifilev4.RevisionState{
		Label:           "Container status is not available",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#666666FF"),
		Icon:            "unknown_document",
		Style:           khifilev4.RevisionStateStyle_PARTIAL_INFO,
	}
	RevisionStateContainerStarted = &khifilev4.RevisionState{
		Label:           "Container is started but readiness info is not available",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#997700FF"),
		Icon:            "siren_question",
		Style:           khifilev4.RevisionStateStyle_PARTIAL_INFO,
	}

	// Endpoint States
	RevisionStateEndpointReady = &khifilev4.RevisionState{
		Label:           "Endpoint is ready",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#004400FF"),
		Icon:            "heart_check",
	}
	RevisionStateEndpointTerminating = &khifilev4.RevisionState{
		Label:           "Endpoint is being terminated",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#CEA700FF"),
		Icon:            "auto_delete",
	}
	RevisionStateEndpointUnready = &khifilev4.RevisionState{
		Label:           "Endpoint is not ready",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#EE4400FF"),
		Icon:            "heart_broken",
	}

	// Pod Phase States
	RevisionStatePodPhasePending = &khifilev4.RevisionState{
		Label:           "Pod is pending",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#666666FF"),
		Icon:            "hourglass_empty",
	}
	RevisionStatePodPhaseScheduled = &khifilev4.RevisionState{
		Label:           "Pod is scheduled",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#4444FFFF"),
		Icon:            "schedule",
	}
	RevisionStatePodPhaseRunning = &khifilev4.RevisionState{
		Label:           "Pod is running",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#004400FF"),
		Icon:            "motion_play",
	}
	RevisionStatePodPhaseSucceeded = &khifilev4.RevisionState{
		Label:           "Pod is succeeded",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#113333FF"),
		Icon:            "check_circle",
		Style:           khifilev4.RevisionStateStyle_DELETED,
	}
	RevisionStatePodPhaseFailed = &khifilev4.RevisionState{
		Label:           "Pod is failed",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#331111FF"),
		Icon:            "error",
		Style:           khifilev4.RevisionStateStyle_DELETED,
	}
	RevisionStatePodPhaseUnknown = &khifilev4.RevisionState{
		Label:           "Pod status is not available from current log range",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#997700FF"),
		Icon:            "unknown_document",
		Style:           khifilev4.RevisionStateStyle_PARTIAL_INFO,
	}

	RevisionStates = []*khifilev4.RevisionState{
		RevisionStateConditionTrue,
		RevisionStateConditionFalse,
		RevisionStateConditionUnknown,
		RevisionStateConditionNotGiven,
		RevisionStateConditionNoAvailableInfo,
		RevisionStateContainerWaiting,
		RevisionStateContainerRunningNonReady,
		RevisionStateContainerRunningReady,
		RevisionStateContainerTerminatedWithSuccess,
		RevisionStateContainerTerminatedWithError,
		RevisionStateContainerStatusNotAvailable,
		RevisionStateContainerStarted,
		RevisionStateEndpointReady,
		RevisionStateEndpointTerminating,
		RevisionStateEndpointUnready,
		RevisionStatePodPhasePending,
		RevisionStatePodPhaseScheduled,
		RevisionStatePodPhaseRunning,
		RevisionStatePodPhaseSucceeded,
		RevisionStatePodPhaseFailed,
		RevisionStatePodPhaseUnknown,
	}
)
