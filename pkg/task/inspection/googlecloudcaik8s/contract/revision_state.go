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

package googlecloudcaik8s_contract

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style RevisionStates.
var (
	// RevisionStateK8sResourceExistingFromCAI indicates the Kubernetes resource existed at the start of inspection, discovered via CAI.
	RevisionStateK8sResourceExistingFromCAI = style.MustRegisterRevisionState(
		"Resource Existing(Asset Inventory)",
		"deployed_code",
		"The Kubernetes resource existed at the beginning of the inspection time range, discovered via Cloud Asset Inventory.",
		style.Color{R: 0.35, G: 0.55, B: 0.95, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateGKEClusterExistingFromCAI indicates the GKE cluster existed, discovered via CAI.
	RevisionStateGKEClusterExistingFromCAI = style.MustRegisterRevisionState(
		"Cluster Existing(Asset Inventory)",
		"deployed_code",
		"The GKE cluster existed at the beginning of the inspection time range, discovered via Cloud Asset Inventory.",
		style.Color{R: 0.35, G: 0.55, B: 0.95, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateGKENodePoolExistingFromCAI indicates the GKE node pool existed, discovered via CAI.
	RevisionStateGKENodePoolExistingFromCAI = style.MustRegisterRevisionState(
		"Node Pool Existing(Asset Inventory)",
		"deployed_code",
		"The GKE node pool existed at the beginning of the inspection time range, discovered via Cloud Asset Inventory.",
		style.Color{R: 0.35, G: 0.55, B: 0.95, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateGKENodePoolExistenceUndetermined indicates the GKE node pool existence was undetermined before the first recorded snapshot.
	RevisionStateGKENodePoolExistenceUndetermined = style.MustRegisterRevisionState(
		"Node Pool existence is undetermined",
		"help_outline",
		"The node pool may have existed because the parent cluster was active, but its existence is undetermined prior to the first recorded asset snapshot.",
		style.MustForceConvertSRGBHex("#888899"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
)
