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

package googlecloudlogk8sevent_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

var (
	// TimelineTypeEventExporter is the timeline style for the GKE Event Exporter.
	TimelineTypeEventExporter = style.MustRegisterTimelineType(
		"event-exporter",
		"GKE Event Exporter logs",
		"dns",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#4285F4"),
		style.ColorWhite,
		true,
		1100,
		style.AlphabeticalSortPolicy(),
	)
)
