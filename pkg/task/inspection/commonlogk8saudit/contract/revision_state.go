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

package commonlogk8saudit_contract

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style RevisionStates.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	RevisionStateConditionTrue = style.MustRegisterRevisionState(
		"State is 'True'",
		"lightbulb",
		"State is 'True'",
		style.MustForceConvertSRGBHex("#004400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateConditionFalse = style.MustRegisterRevisionState(
		"State is 'False'",
		"light_off",
		"State is 'False'",
		style.MustForceConvertSRGBHex("#EE4400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateConditionUnknown = style.MustRegisterRevisionState(
		"State is 'Unknown'",
		"siren_question",
		"State is 'Unknown'",
		style.MustForceConvertSRGBHex("#663366"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateConditionNotGiven = style.MustRegisterRevisionState(
		"Condition is not defined at this moment",
		"select",
		"Condition is not defined at this moment",
		style.MustForceConvertSRGBHex("#666666"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateConditionNoAvailableInfo = style.MustRegisterRevisionState(
		"No enough information to show condition",
		"unknown_document",
		"No enough information to show condition",
		style.MustForceConvertSRGBHex("#997700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateEndpointReady = style.MustRegisterRevisionState(
		"Endpoint is ready",
		"heart_check",
		"Endpoint is ready",
		style.MustForceConvertSRGBHex("#004400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateEndpointUnready = style.MustRegisterRevisionState(
		"Endpoint is not ready",
		"heart_broken",
		"Endpoint is not ready",
		style.MustForceConvertSRGBHex("#EE4400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateEndpointTerminating = style.MustRegisterRevisionState(
		"Endpoint is being terminated",
		"auto_delete",
		"Endpoint is being terminated",
		style.MustForceConvertSRGBHex("#cea700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStatePodPhasePending = style.MustRegisterRevisionState(
		"Pod is pending",
		"hourglass_empty",
		"Pod is pending",
		style.MustForceConvertSRGBHex("#666666"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStatePodPhaseScheduled = style.MustRegisterRevisionState(
		"Pod is scheduled",
		"schedule",
		"Pod is scheduled",
		style.MustForceConvertSRGBHex("#4444ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStatePodPhaseRunning = style.MustRegisterRevisionState(
		"Pod is running",
		"motion_play",
		"Pod is running",
		style.MustForceConvertSRGBHex("#004400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStatePodPhaseSucceeded = style.MustRegisterRevisionState(
		"Pod is succeeded",
		"check_circle",
		"Pod is succeeded",
		style.MustForceConvertSRGBHex("#113333"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStatePodPhaseFailed = style.MustRegisterRevisionState(
		"Pod is failed",
		"error",
		"Pod is failed",
		style.MustForceConvertSRGBHex("#331111"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStatePodPhaseUnknown = style.MustRegisterRevisionState(
		"Pod status is not available from current log range",
		"unknown_document",
		"Pod status is not available from current log range",
		style.MustForceConvertSRGBHex("#997700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
	RevisionStateContainerWaiting = style.MustRegisterRevisionState(
		"Waiting for starting container",
		"deployed_code_history",
		"Waiting for starting container",
		style.MustForceConvertSRGBHex("#4444ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateContainerRunningNonReady = style.MustRegisterRevisionState(
		"Container is not ready",
		"heart_broken",
		"Container is not ready",
		style.MustForceConvertSRGBHex("#EE4400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateContainerRunningReady = style.MustRegisterRevisionState(
		"Container is ready",
		"heart_check",
		"Container is ready",
		style.MustForceConvertSRGBHex("#007700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateContainerTerminatedWithSuccess = style.MustRegisterRevisionState(
		"Container exited with healthy exit code",
		"check_circle",
		"Container exited with healthy exit code",
		style.MustForceConvertSRGBHex("#113333"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateContainerTerminatedWithError = style.MustRegisterRevisionState(
		"Container exited with erroneous exit code",
		"error",
		"Container exited with erroneous exit code",
		style.MustForceConvertSRGBHex("#551111"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateContainerStatusNotAvailable = style.MustRegisterRevisionState(
		"Container status is not available",
		"unknown_document",
		"Container status is not available",
		style.MustForceConvertSRGBHex("#666666"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
	RevisionStateContainerStarted = style.MustRegisterRevisionState(
		"Container is started but readiness info is not available",
		"siren_question",
		"Container is started but readiness info is not available",
		style.MustForceConvertSRGBHex("#997700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
)
