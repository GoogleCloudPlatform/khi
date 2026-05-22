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
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

var (
	// TimelineTypeOnPremProject is the timeline type style for On-Prem Projects.
	TimelineTypeOnPremProject = style.MustRegisterTimelineType(
		"onpremProject",
		"On-Prem Project",
		"cloud",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#4285F4"),
		true,
		10000,
	)

	// TimelineTypeOnPremCluster is the timeline type style for On-Prem Clusters.
	TimelineTypeOnPremCluster = style.MustRegisterTimelineType(
		"onpremCluster",
		"On-Prem Cluster",
		"dns",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#34A853"),
		true,
		10100,
	)

	// TimelineTypeOnPremNodePool is the timeline type style for On-Prem NodePools.
	TimelineTypeOnPremNodePool = style.MustRegisterTimelineType(
		"onpremNodePool",
		"On-Prem NodePool",
		"workspaces",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#FBBC05"),
		true,
		10200,
	)
)
