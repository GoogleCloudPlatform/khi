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

package googlecloudloggkeautoscaler_contract

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style RevisionStates.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	RevisionAutoscalerNoError = style.MustRegisterRevisionState(
		"Autoscaler has no error",
		"heart_check",
		"The GKE Cluster Autoscaler is operating normally without any reported errors.",
		style.MustForceConvertSRGBHex("#004400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionAutoscalerHasErrors = style.MustRegisterRevisionState(
		"Autoscaler has errors",
		"heart_broken",
		"The GKE Cluster Autoscaler has encountered errors preventing normal autoscaling operations.",
		style.MustForceConvertSRGBHex("#EE4400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
)
