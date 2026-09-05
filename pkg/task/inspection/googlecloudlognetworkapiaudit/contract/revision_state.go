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

package googlecloudlognetworkapiaudit_contract

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

var (
	// RevisionStateNEGEndpointAttaching indicates that the network endpoint is currently being attached to the NEG.
	RevisionStateNEGEndpointAttaching = style.MustRegisterRevisionState(
		"Endpoint is being attached",
		"deployed_code_history",
		"The network endpoint is currently being attached to the NEG.",
		style.MustForceConvertSRGBHex("#6666ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateNEGEndpointAttached indicates that the network endpoint is attached and active in the NEG.
	RevisionStateNEGEndpointAttached = style.MustRegisterRevisionState(
		"Endpoint is attached",
		"deployed_code",
		"The network endpoint is attached to the NEG and ready to receive traffic.",
		style.MustForceConvertSRGBHex("#007700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateNEGEndpointDetaching indicates that the network endpoint is currently being detached from the NEG.
	RevisionStateNEGEndpointDetaching = style.MustRegisterRevisionState(
		"Endpoint is being detached",
		"auto_delete",
		"The network endpoint is currently being detached from the NEG (e.g. connection draining).",
		style.MustForceConvertSRGBHex("#CC5500"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateNEGEndpointDetached indicates that the network endpoint has been detached from the NEG.
	RevisionStateNEGEndpointDetached = style.MustRegisterRevisionState(
		"Endpoint is detached",
		"delete_forever",
		"The network endpoint has been detached from the NEG.",
		style.MustForceConvertSRGBHex("#CC0000"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)

	// RevisionStateNEGEndpointExistingLogNotFound indicates that the network endpoint existed in the NEG prior to detach, but its attach log was not found in the selected time range.
	RevisionStateNEGEndpointExistingLogNotFound = style.MustRegisterRevisionState(
		"Endpoint exists, but attach log not found",
		"deployed_code",
		"The network endpoint existed in the NEG prior to detach, but the attach log was not found in the selected time range. Its readiness or health prior to this event cannot be determined.",
		style.MustForceConvertSRGBHex("#cea700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
)
