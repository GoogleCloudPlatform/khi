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

package googlecloudlogonpremapiaudit_contract

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

var (
	// RevisionStateProvisioning represents the resource provisioning state.
	RevisionStateProvisioning = style.MustRegisterRevisionState(
		"Resource is being provisioned",
		"deployed_code_history",
		"The on-prem GKE API resource is currently being provisioned.",
		style.MustForceConvertSRGBHex("#6666ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateExisting represents the resource existing state.
	RevisionStateExisting = style.MustRegisterRevisionState(
		"Resource exists",
		"check_circle",
		"The on-prem GKE API resource exists and is active.",
		style.MustForceConvertSRGBHex("#00aa00"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateDeleting represents the resource deleting state.
	RevisionStateDeleting = style.MustRegisterRevisionState(
		"Resource is being deleted",
		"delete_sweep",
		"The on-prem GKE API resource is in the process of being deleted.",
		style.MustForceConvertSRGBHex("#ff6666"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateDeleted represents the resource deleted state.
	RevisionStateDeleted = style.MustRegisterRevisionState(
		"Resource is deleted",
		"cancel",
		"The on-prem GKE API resource has been deleted.",
		style.MustForceConvertSRGBHex("#aa0000"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)

	// RevisionStateOperationStarted represents the operation started state.
	RevisionStateOperationStarted = style.MustRegisterRevisionState(
		"Processing operation",
		"play_arrow",
		"The on-prem GKE API resource is processing a long-running operation.",
		style.MustForceConvertSRGBHex("#00bb00"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateOperationFinished represents the operation finished state.
	RevisionStateOperationFinished = style.MustRegisterRevisionState(
		"Operation is finished",
		"stop",
		"The on-prem GKE API resource has finished the long-running operation.",
		style.MustForceConvertSRGBHex("#777777"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
)
