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

package googlecloudlogk8scontainer_contract

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style RevisionStates.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	RevisionStateContainerWaiting = style.MustRegisterRevisionState(
		"Container is waiting",
		"deployed_code_history",
		"The container is waiting to start (e.g., image pull or init container execution is in progress).",
		style.MustForceConvertSRGBHex("#4444ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateContainerRunningNonReady = style.MustRegisterRevisionState(
		"Container is not ready",
		"heart_broken",
		"The container is running but has not passed its readiness probe.",
		style.MustForceConvertSRGBHex("#EE4400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateContainerRunningReady = style.MustRegisterRevisionState(
		"Container is ready",
		"heart_check",
		"The container is running and has successfully passed its readiness probe.",
		style.MustForceConvertSRGBHex("#007700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateContainerTerminatedWithSuccess = style.MustRegisterRevisionState(
		"Container exited successfully",
		"check_circle",
		"The container has terminated successfully with an exit code of `0`.",
		style.MustForceConvertSRGBHex("#113333"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateContainerTerminatedWithError = style.MustRegisterRevisionState(
		"Container exited with error",
		"error",
		"The container has terminated in failure with a non-zero exit code.",
		style.MustForceConvertSRGBHex("#551111"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateContainerStatusNotAvailable = style.MustRegisterRevisionState(
		"Container status is unavailable",
		"unknown_document",
		`The container status is unknown or not yet reported in the log data.

**Tip**: Consider expanding the query time range to capture complete container status events.`,
		style.MustForceConvertSRGBHex("#666666"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
	RevisionStateContainerStarted = style.MustRegisterRevisionState(
		"Container is started, readiness unknown",
		"siren_question",
		`The container has started, but no readiness probe information has been recorded yet.

**Tip**: Consider expanding the query time range to observe readiness probe events.`,
		style.MustForceConvertSRGBHex("#997700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
)
