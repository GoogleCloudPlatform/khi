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
		"Control plane operations and lifecycle logs of the GKE cluster",
		"cloud",
		1.0,
		style.Color{R: 0.780, G: 0.863, B: 1.000, A: 1.0},
		style.ColorBlack,
		style.Color{R: 0.780, G: 0.863, B: 1.000, A: 1.0},
		style.ColorBlack,
		true,
		40,
		style.AlphabeticalSortPolicy(),
	)
	// TimelineTypeGKEControlPlanes is the timeline style for a GKE control planes folder.
	TimelineTypeGKEControlPlanes = style.MustRegisterTimelineType(
		"controlplanes",
		"Control Plane",
		"category",
		0.6,
		style.Color{R: 0.361, G: 0.604, B: 1.000, A: 1.0},
		style.ColorWhite,
		style.Color{R: 0.361, G: 0.604, B: 1.000, A: 1.0},
		style.ColorWhite,
		false,
		3002,
		style.AlphabeticalSortPolicy(),
	)
	// TimelineTypeGKENodePools is the timeline style for a GKE node pools folder.
	TimelineTypeGKENodePools = style.MustRegisterTimelineType(
		"nodepools",
		"Node Pools",
		"dns",
		0.6,
		style.Color{R: 0.361, G: 0.604, B: 1.000, A: 1.0},
		style.ColorWhite,
		style.Color{R: 0.361, G: 0.604, B: 1.000, A: 1.0},
		style.ColorWhite,
		false,
		3001, // Must be higher than Operations
		style.AlphabeticalSortPolicy(),
	)
	// TimelineTypeOtherGKEResources is the timeline style for other GKE resources.
	TimelineTypeOtherGKEResources = style.MustRegisterTimelineType(
		"other_gke_resources",
		"Other GKE Resources",
		"category",
		0.6,
		style.Color{R: 0.361, G: 0.604, B: 1.000, A: 1.0},
		style.ColorWhite,
		style.Color{R: 0.361, G: 0.604, B: 1.000, A: 1.0},
		style.ColorWhite,
		false,
		3003,
		style.AlphabeticalSortPolicy(),
	)
	// TimelineTypeGKENodePool is the style for a GKE node pool.
	TimelineTypeGKENodePool = style.MustRegisterTimelineType(
		"nodepool",
		"Grouping timeline for GKE nodepools",
		"dns",
		0.8,
		style.Color{R: 0.941, G: 0.965, B: 1.000, A: 1.0},
		style.ColorBlack,
		style.Color{R: 0.941, G: 0.965, B: 1.000, A: 1.0},
		style.ColorBlack,
		true,
		45,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeOperation = style.MustRegisterTimelineType(
		"operation",
		"Google Cloud operations associated with the resource",
		"engineering",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.ColorBlack,
		style.ColorWhite,
		true,
		3000,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeGCPProject = style.MustRegisterTimelineType(
		"project",
		"Timeline representing a Google Cloud project",
		"cloud",
		1.0,
		style.Color{R: 0.102, G: 0.451, B: 0.910, A: 1.0},
		style.ColorWhite,
		style.Color{R: 0.102, G: 0.451, B: 0.910, A: 1.0},
		style.ColorWhite,
		true,
		30,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeGCPResourceType = style.MustRegisterTimelineType(
		"gcp_resource_type",
		"Grouping timeline for Google Cloud resource types",
		"category",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#34A853"),
		style.ColorWhite,
		true,
		31,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeGCPResource = style.MustRegisterTimelineType(
		"gcp_resource",
		"Timeline representing a Google Cloud resource",
		"deployed_code",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#FBBC05"),
		style.ColorBlack,
		true,
		32,
		style.AlphabeticalSortPolicy(),
	)
)
