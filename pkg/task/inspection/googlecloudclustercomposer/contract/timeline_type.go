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

package googlecloudclustercomposer_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style TimelineTypes.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (

	// TimelineTypeAirflow is the style for an Airflow environment.
	TimelineTypeAirflow = style.MustRegisterTimelineType(
		"Airflow",
		"Timeline representing a Managed Airflow environment",
		"settings",
		0.6,
		style.Color{R: 0.82, G: 0.28, B: 0.19, A: 1},
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		style.ColorBlack,
		true,
		20,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeDAGs is the root style for DAGs hierarchy.
	TimelineTypeDAGs = style.MustRegisterTimelineType(
		"DAGs",
		"Grouping timeline for Airflow DAGs",
		"folder",
		0.6,
		style.Color{R: 0.93, G: 0.49, B: 0.38, A: 1},
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		style.ColorBlack,
		true,
		30,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeAirflowDAG is the style for a single DAG.
	TimelineTypeAirflowDAG = style.MustRegisterTimelineType(
		"DAG",
		"Timeline representing an Airflow DAG",
		"account_tree",
		0.6,
		style.MustForceConvertSRGBHex("#444444"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		style.ColorBlack,
		true,
		40,
		style.ChronologicalSortPolicy(2),
	)

	// TimelineTypeAirflowDAGRun is the style for a DAG run.
	TimelineTypeAirflowDAGRun = style.MustRegisterTimelineType(
		"DAG Run",
		"Timeline representing an Airflow DAG run",
		"play_circle",
		0.6,
		style.MustForceConvertSRGBHex("#CCCCCC"),
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		style.ColorBlack,
		true,
		50,
		style.ChronologicalSortPolicy(1),
	)

	// TimelineTypeAirflowTaskInstance is the style for a TaskInstance.
	TimelineTypeAirflowTaskInstance = style.MustRegisterTimelineType(
		"Task Instance",
		"Execution states of the Airflow task instance",
		"mode_fan",
		0.7,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		style.ColorBlack,
		true,
		1501,
		style.ChronologicalSortPolicy(1),
	)

	// TimelineTypeComponents is the category style for components.
	TimelineTypeComponents = style.MustRegisterTimelineType(
		"Components",
		"Grouping timeline for Airflow backend components",
		"apps",
		0.6,
		style.Color{R: 0.93, G: 0.49, B: 0.38, A: 1},
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		style.ColorBlack,
		true,
		70,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeDAGFiles is the category style for DAG Processor Manager stats.
	TimelineTypeDAGFiles = style.MustRegisterTimelineType(
		"DAG files",
		"Grouping timeline for parsed DAG files",
		"folder",
		0.6,
		style.Color{R: 0.93, G: 0.49, B: 0.38, A: 1},
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		style.ColorBlack,
		true,
		60,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeDAGFile is the style for a parsed DAG file.
	TimelineTypeDAGFile = style.MustRegisterTimelineType(
		"DAG File",
		"Timeline representing an Airflow DAG definition file",
		"description",
		0.6,
		style.MustForceConvertSRGBHex("#444444"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		style.ColorBlack,
		true,
		80,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeDAGProcessorManagerInstance is the style for the manager instance that processed the file.
	TimelineTypeDAGProcessorManagerInstance = style.MustRegisterTimelineType(
		"Parser",
		"Logs of the DAG Processor Manager instance. Same DAG file can be parsed from multiple DAG Processor Manager instances at the same time thus this is shown as separated timelines.",
		"terminal",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		style.ColorBlack,
		true,
		90,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeAirflowComponent is the style for Airflow components.
	TimelineTypeAirflowComponent = style.MustRegisterTimelineType(
		"Component",
		"Logs of the generic Airflow component",
		"extension",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#F5F5F5"),
		style.ColorBlack,
		true,
		100,
		style.AlphabeticalSortPolicy(),
	)
)
