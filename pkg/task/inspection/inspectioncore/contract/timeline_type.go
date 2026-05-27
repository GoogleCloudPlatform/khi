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
		0.6,
		style.MustForceConvertSRGBHex("#111111"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		true,
		50,
		style.AlphabeticalSortPolicy(),
	)
	TimelineTypeAPIVersion = style.MustRegisterTimelineType(
		"apiVersion",
		"Kubernetes API Version layer",
		"api",
		0.6,
		style.MustForceConvertSRGBHex("#3949ab"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#EEEEEE"),
		true,
		100,
		style.AlphabeticalSortPolicy("core/v1", "apps/v1"),
	)
	TimelineTypeKind = style.MustRegisterTimelineType(
		"kind",
		"Kubernetes API Resource Kind layer",
		"workspaces",
		0.6,
		style.MustForceConvertSRGBHex("#3949ab"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#DDDDDD"),
		true,
		200,
		style.AlphabeticalSortPolicy("node", "pod", "service", "replicaset", "deployment", "statefulset", "daemonset", "job", "cronjob"),
	)
	TimelineTypeNamespace = style.MustRegisterTimelineType(
		"namespace",
		"Kubernetes Namespace layer",
		"folder",
		0.6,
		style.MustForceConvertSRGBHex("#444444"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#D0D0D0"),
		true,
		300,
		style.AlphabeticalSortPolicy("kube-system", "default"),
	)
	TimelineTypeResource = style.MustRegisterTimelineType(
		"resource",
		"General resource lifecycle and logs",
		"description",
		1.0,
		style.MustForceConvertSRGBHex("#CCCCCC"),
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#CCCCCC"),
		true,
		1000,
		style.ChronologicalSortPolicy(0),
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
		style.AlphabeticalSortPolicy(),
	)
)
