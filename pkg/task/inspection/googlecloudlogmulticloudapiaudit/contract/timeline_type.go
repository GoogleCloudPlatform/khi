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

package googlecloudlogmulticloudapiaudit_contract

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

var (
	// TimelineTypeProject is the timeline type for a Google Cloud project.
	TimelineTypeProject = style.MustRegisterTimelineType(
		"project",
		"Google Cloud Project",
		"cloud",
		0.6,
		style.MustForceConvertSRGBHex("#111111"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		style.ColorBlack,
		true,
		50,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeMultiCloudCluster is the timeline type style for MultiCloud Cluster resources.
	TimelineTypeMultiCloudCluster = style.MustRegisterTimelineType(
		"multicloudCluster",
		"MultiCloud Cluster",
		"dns",
		1.0,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#4285F4"),
		style.ColorWhite,
		true,
		1000,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeMultiCloudNodepool is the timeline type style for MultiCloud Nodepool resources.
	TimelineTypeMultiCloudNodepool = style.MustRegisterTimelineType(
		"multicloudNodepool",
		"MultiCloud NodePool",
		"list",
		1.0,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#34A853"),
		style.ColorWhite,
		true,
		1100,
		style.AlphabeticalSortPolicy(),
	)

	// RevisionStateProvisioning is the style for a resource that is being provisioned.
	RevisionStateProvisioning = style.MustRegisterRevisionState(
		"Resource is being provisioned",
		"deployed_code_history",
		"Resource is being provisioned",
		style.MustForceConvertSRGBHex("#6666ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
)
