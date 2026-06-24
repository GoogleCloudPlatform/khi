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

package googlecloudcommon_contract

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style RevisionStates.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	RevisionStateOperationStarted = style.MustRegisterRevisionState(
		"Processing operation",
		"change_circle",
		"The GCP API long-running operation is currently in progress.",
		style.Color{R: 0.012, G: 0.671, B: 0.012, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateOperationSucceed = style.MustRegisterRevisionState(
		"Operation succeeded",
		"check_circle",
		"The GCP API long-running operation has completed successfully.",
		style.Color{R: 0.812, G: 0.812, B: 0.812, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateOperationFailed = style.MustRegisterRevisionState(
		"Operation failed",
		"error",
		"The GCP API long-running operation has failed.",
		style.Color{R: 1.000, G: 0.000, B: 0.000, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateOperationStartedLogNotFound = style.MustRegisterRevisionState(
		"Operation started, but starting log not found",
		"unknown_document",
		"The operation was started, but the starting log entry was not found in the selected time range. Try adjusting the time range.",
		style.Color{R: 0.012, G: 0.671, B: 0.012, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
)
