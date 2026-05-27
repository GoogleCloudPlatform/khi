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
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style TimelineTypes.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	TimelineTypeGKE = style.MustRegisterTimelineType(
		"gke",
		"GKE Control Plane and lifecycle logs",
		"cloud",
		1.0,
		style.MustForceConvertSRGBHex("#4285F4"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#4285F4"),
		true,
		40,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeGKENodePool = style.MustRegisterTimelineType(
		"nodepool",
		"GKE Nodepool layer",
		"dns",
		0.9,
		style.MustForceConvertSRGBHex("#c5dbff"),
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#E0E0E0"),
		true,
		45,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeOperation = style.MustRegisterTimelineType(
		"operation",
		"GCP operations associated with this resource",
		"engineering",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.ColorBlack,
		true,
		3000,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeGCPProject = style.MustRegisterTimelineType(
		"project",
		"Google Cloud Project",
		"cloud",
		1.0,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#4285F4"),
		true,
		30,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeGCPResourceType = style.MustRegisterTimelineType(
		"gcp_resource_type",
		"GCP Resource Type",
		"category",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#34A853"),
		true,
		31,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeGCPResource = style.MustRegisterTimelineType(
		"gcp_resource",
		"GCP Resource",
		"deployed_code",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#FBBC05"),
		true,
		32,
		style.AlphabeticalSortPolicy(),
	)
)
