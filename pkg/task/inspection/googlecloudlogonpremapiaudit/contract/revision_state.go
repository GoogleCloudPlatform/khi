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
		"Provisioning",
		"deployed_code_history",
		"Resource is being provisioned",
		style.MustForceConvertSRGBHex("#6666ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateExisting represents the resource existing state.
	RevisionStateExisting = style.MustRegisterRevisionState(
		"Existing",
		"check_circle",
		"Resource exists",
		style.MustForceConvertSRGBHex("#00aa00"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateDeleting represents the resource deleting state.
	RevisionStateDeleting = style.MustRegisterRevisionState(
		"Deleting",
		"delete_sweep",
		"Resource is being deleted",
		style.MustForceConvertSRGBHex("#ff6666"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateDeleted represents the resource deleted state.
	RevisionStateDeleted = style.MustRegisterRevisionState(
		"Deleted",
		"cancel",
		"Resource is deleted",
		style.MustForceConvertSRGBHex("#aa0000"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)

	// RevisionStateOperationStarted represents the operation started state.
	RevisionStateOperationStarted = style.MustRegisterRevisionState(
		"OperationStarted",
		"play_arrow",
		"Operation started",
		style.MustForceConvertSRGBHex("#00bb00"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateOperationFinished represents the operation finished state.
	RevisionStateOperationFinished = style.MustRegisterRevisionState(
		"OperationFinished",
		"stop",
		"Operation finished",
		style.MustForceConvertSRGBHex("#777777"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
)
