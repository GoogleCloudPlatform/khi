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

package csmcp

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

var (
	// TimelineTypeCSMCPPodLog is the timeline type style for CSM CP Pod Logs.
	TimelineTypeCSMCPPodLog = style.MustRegisterTimelineType(
		"csmcp-log",
		"CSM CP Log associated with the Pod",
		"dns",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.Color{R: 1.0, G: 133.0 / 255.0, B: 0.0, A: 1.0},
		style.ColorBlack,
		true,
		100000,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeCSMCPConnection is the timeline type style for CSM CP Connections.
	TimelineTypeCSMCPConnection = style.MustRegisterTimelineType(
		"csmcp-connection",
		"CSM CP Connection",
		"cable",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.Color{R: 1.0, G: 133.0 / 255.0, B: 0.0, A: 1.0},
		style.ColorBlack,
		true,
		100001,
		style.ChronologicalSortPolicy(1),
	)
)
