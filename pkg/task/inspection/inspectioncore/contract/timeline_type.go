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

package inspectioncore_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style TimelineTypes.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	TimelineTypeK8sCluster = style.MustRegisterTimelineType(
		"k8sCluster",
		"Kubernetes Cluster layer",
		"cloud",
		0.5,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		true,
		50,
	)
	TimelineTypeAPIVersion = style.MustRegisterTimelineType(
		"apiVersion",
		"Kubernetes API Version layer",
		"api",
		0.55,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#EEEEEE"),
		true,
		100,
	)
	TimelineTypeKind = style.MustRegisterTimelineType(
		"kind",
		"Kubernetes API Resource Kind layer",
		"workspaces",
		0.55,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#DDDDDD"),
		true,
		200,
	)
	TimelineTypeNamespace = style.MustRegisterTimelineType(
		"namespace",
		"Kubernetes Namespace layer",
		"folder",
		0.55,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#D0D0D0"),
		true,
		300,
	)
	TimelineTypeResource = style.MustRegisterTimelineType(
		"resource",
		"General resource lifecycle and logs",
		"description",
		1.0,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#CCCCCC"),
		true,
		1000,
	)
	TimelineTypeSubresource = style.MustRegisterTimelineType(
		"subresource",
		"General subresource lifecycle and logs",
		"page_info",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#BBBBBB"),
		true,
		1200,
	)
)
