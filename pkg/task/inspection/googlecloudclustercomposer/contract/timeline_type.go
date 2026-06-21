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

	// TimelineTypeComposerEnvironment is the style for a Cloud Composer environment.
	// Background is set to #377e22.
	TimelineTypeComposerEnvironment = style.MustRegisterTimelineType(
		"Airflow",
		"Timeline representing a Managed Airflow environment",
		"settings",
		0.6,
		style.MustForceConvertSRGBHex("#377e22"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#377e22"),
		style.ColorWhite,
		true,
		20,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeDAGs is the root style for DAGs hierarchy.
	// Progressive lighter green background #5cb239.
	TimelineTypeDAGs = style.MustRegisterTimelineType(
		"dags",
		"Grouping timeline for Airflow DAGs",
		"folder",
		0.6,
		style.MustForceConvertSRGBHex("#5cb239"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#5cb239"),
		style.ColorWhite,
		true,
		30,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeAirflowDAG is the style for a single DAG.
	// Progressive lighter green background #89ca6a.
	TimelineTypeAirflowDAG = style.MustRegisterTimelineType(
		"airflow_dag",
		"Timeline representing an Airflow DAG",
		"account_tree",
		0.6,
		style.MustForceConvertSRGBHex("#89ca6a"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#89ca6a"),
		style.ColorWhite,
		true,
		40,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeAirflowDAGRun is the style for a DAG run.
	// Progressive lighter green background #bce3a5.
	TimelineTypeAirflowDAGRun = style.MustRegisterTimelineType(
		"airflow_dag_run",
		"Timeline representing an Airflow DAG run",
		"play_circle",
		0.6,
		style.MustForceConvertSRGBHex("#bce3a5"),
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#bce3a5"),
		style.ColorBlack,
		true,
		50,
		style.ChronologicalSortPolicy(1),
	)

	// TimelineTypeAirflowTaskInstance is the style for a TaskInstance.
	// As it contains raw log/revisions, its background is set to White.
	TimelineTypeAirflowTaskInstance = style.MustRegisterTimelineType(
		"task",
		"Execution states of the Airflow task instance",
		"mode_fan",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#377e22"),
		style.ColorWhite,
		true,
		1501,
		style.ChronologicalSortPolicy(1),
	)

	// TimelineTypeComponents is the category style for components.
	// Progressive lighter green background #5cb239.
	TimelineTypeComponents = style.MustRegisterTimelineType(
		"airflow_components",
		"Grouping timeline for Airflow backend components",
		"apps",
		0.6,
		style.MustForceConvertSRGBHex("#5cb239"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#5cb239"),
		style.ColorWhite,
		true,
		60,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeDAGProcessorManager is the category style for DAG Processor Manager stats.
	// Progressive lighter green background #5cb239.
	TimelineTypeDAGProcessorManager = style.MustRegisterTimelineType(
		"dag_files",
		"Grouping timeline for parsed DAG files",
		"folder",
		0.6,
		style.MustForceConvertSRGBHex("#5cb239"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#5cb239"),
		style.ColorWhite,
		true,
		70,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeDAGFile is the style for a parsed DAG file.
	// Progressive lighter green background #89ca6a.
	TimelineTypeDAGFile = style.MustRegisterTimelineType(
		"dag_file",
		"Timeline representing an Airflow DAG definition file",
		"description",
		0.6,
		style.MustForceConvertSRGBHex("#89ca6a"),
		style.ColorWhite,
		style.MustForceConvertSRGBHex("#89ca6a"),
		style.ColorWhite,
		true,
		80,
		style.AlphabeticalSortPolicy(),
	)

	// TimelineTypeDAGProcessorManagerInstance is the style for the manager instance that processed the file.
	// As it contains revisions, its background is set to White.
	TimelineTypeDAGProcessorManagerInstance = style.MustRegisterTimelineType(
		"dag_processor_manager_instance",
		"Logs of the DAG Processor Manager instance",
		"terminal",
		0.6,
		style.ColorWhite,
		style.ColorBlack,
		style.MustForceConvertSRGBHex("#A51915"),
		style.ColorWhite,
		true,
		90,
		style.AlphabeticalSortPolicy(),
	)

	// Components specific timelines (Actual log/event containers: White background)
	TimelineTypeAirflowScheduler           = style.MustRegisterTimelineType("airflow_scheduler", "Logs of the Airflow Scheduler", "schedule", 0.6, style.ColorWhite, style.ColorBlack, style.MustForceConvertSRGBHex("#4285F4"), style.ColorWhite, true, 100, style.AlphabeticalSortPolicy())
	TimelineTypeAirflowWorker              = style.MustRegisterTimelineType("airflow_worker", "Logs of the Airflow Worker", "directions_run", 0.6, style.ColorWhite, style.ColorBlack, style.MustForceConvertSRGBHex("#0F9D58"), style.ColorWhite, true, 110, style.AlphabeticalSortPolicy())
	TimelineTypeAirflowDagProcessorManager = style.MustRegisterTimelineType("airflow_dag_processor_manager", "Logs of the Airflow DAG Processor Manager", "summarize", 0.6, style.ColorWhite, style.ColorBlack, style.MustForceConvertSRGBHex("#808080"), style.ColorWhite, true, 115, style.AlphabeticalSortPolicy())
	TimelineTypeAirflowTriggerer           = style.MustRegisterTimelineType("airflow_triggerer", "Logs of the Airflow Triggerer", "bolt", 0.6, style.ColorWhite, style.ColorBlack, style.MustForceConvertSRGBHex("#FFBB00"), style.ColorBlack, true, 120, style.AlphabeticalSortPolicy())
	TimelineTypeAirflowWebserver           = style.MustRegisterTimelineType("airflow_webserver", "Logs of the Airflow Webserver", "web", 0.6, style.ColorWhite, style.ColorBlack, style.MustForceConvertSRGBHex("#9470DC"), style.ColorWhite, true, 130, style.AlphabeticalSortPolicy())
	TimelineTypeAirflowComponent           = style.MustRegisterTimelineType("airflow_component", "Logs of the generic Airflow component", "extension", 0.6, style.ColorWhite, style.ColorBlack, style.MustForceConvertSRGBHex("#808080"), style.ColorWhite, true, 140, style.AlphabeticalSortPolicy())
)
