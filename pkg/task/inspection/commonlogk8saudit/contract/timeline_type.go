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

package commonlogk8saudit_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style TimelineTypes.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	TimelineTypeResourceCondition = style.MustRegisterTimelineType(
		"condition",
		"Resource conditions from .status.conditions",
		"conditions",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#4c29e8"),
		true,
		2000,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeEndpointSlice = style.MustRegisterTimelineType(
		"endpoint",
		"Pod serving status from EndpointSlice",
		"line_end_diamond",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#008000"),
		true,
		20000,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeOwnerReference = style.MustRegisterTimelineType(
		"owns",
		"Child resource from .metadata.ownerReferences",
		"link",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#33DD88"),
		true,
		7000,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypePodPhase = style.MustRegisterTimelineType(
		"pod",
		"Pod status on the node from .status.phase",
		"token",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#FF8855"),
		true,
		8000,
		style.AlphabeticalSortPolicy(),
	)
	// TimelineTypeContainer is the timeline type style for Kubernetes containers.
	TimelineTypeContainer = style.MustRegisterTimelineType(
		"container",
		"Container status and logs",
		"activity_zone",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#fe9bab"),
		true,
		5000,
		style.AlphabeticalSortPolicy(),
	)
)
