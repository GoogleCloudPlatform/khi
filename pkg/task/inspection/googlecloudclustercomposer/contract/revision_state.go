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
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style RevisionStates.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	RevisionStateComposerTiScheduled = style.MustRegisterRevisionState(
		"Task instance is scheduled",
		"schedule",
		"The Airflow task instance has been scheduled and is waiting to be queued.",
		style.MustForceConvertSRGBHex("#d1b48c"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiQueued = style.MustRegisterRevisionState(
		"Task instance is queued",
		"transition_push",
		"The Airflow task instance has been queued in the executor and is waiting to run.",
		style.MustForceConvertSRGBHex("#808080"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiRunning = style.MustRegisterRevisionState(
		"Task instance is running",
		"directions_run",
		"The Airflow task instance is currently executing.",
		style.MustForceConvertSRGBHex("#00ff01"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiDeferred = style.MustRegisterRevisionState(
		"Task instance is deferred",
		"pause",
		"The Airflow task instance is deferred, waiting for a trigger to resume.",
		style.MustForceConvertSRGBHex("#9470dc"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiSuccess = style.MustRegisterRevisionState(
		"Task instance succeeded",
		"check",
		"The Airflow task instance has completed successfully.",
		style.MustForceConvertSRGBHex("#008001"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiFailed = style.MustRegisterRevisionState(
		"Task instance failed",
		"exclamation",
		"The Airflow task instance has failed during execution.",
		style.MustForceConvertSRGBHex("#fe0000"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiUpForRetry = style.MustRegisterRevisionState(
		"Task instance is up for retry",
		"camping",
		"The Airflow task instance has failed and is waiting to be retried.",
		style.MustForceConvertSRGBHex("#fed700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiRestarting = style.MustRegisterRevisionState(
		"Task instance is restarting",
		"restart_alt",
		"The Airflow task instance is being restarted.",
		style.MustForceConvertSRGBHex("#ee82ef"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiRemoved = style.MustRegisterRevisionState(
		"Task instance is removed",
		"waving_hand",
		"The Airflow task instance has been removed from the DAG run.",
		style.MustForceConvertSRGBHex("#d3d3d3"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiUpstreamFailed = style.MustRegisterRevisionState(
		"Upstream task failed",
		"falling",
		"The Airflow task instance has been skipped because one of its upstream dependencies failed.",
		style.MustForceConvertSRGBHex("#ffa11b"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiZombie = style.MustRegisterRevisionState(
		"Task instance is a zombie",
		"skull",
		"The Airflow task instance is detected as a zombie (the process died without updating the database state).",
		style.MustForceConvertSRGBHex("#4b0082"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiUpForReschedule = style.MustRegisterRevisionState(
		"Task instance is up for reschedule",
		"history",
		"The Airflow task instance is in up_for_reschedule state, waiting for the next sensor poll.",
		style.MustForceConvertSRGBHex("#808080"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiSkipped = style.MustRegisterRevisionState(
		"Task instance is skipped",
		"step_over",
		"The Airflow task instance has been skipped during execution.",
		style.MustForceConvertSRGBHex("#e60076"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	// RevisionStateComposerDagProcessorNoError indicates that DAG file processing completed without errors.
	RevisionStateComposerDagProcessorNoError = style.MustRegisterRevisionState(
		"DAG processing has no errors",
		"check",
		"The Airflow DAG processor manager processed the DAG file without errors.",
		style.MustForceConvertSRGBHex("#008001"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	// RevisionStateComposerDagProcessorHasErrors indicates that DAG file processing encountered errors.
	RevisionStateComposerDagProcessorHasErrors = style.MustRegisterRevisionState(
		"DAG processing has errors",
		"exclamation",
		"The Airflow DAG processor manager encountered errors while processing the DAG file.",
		style.MustForceConvertSRGBHex("#fe0000"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	// RevisionStateManagedAirflowEnvironmentProvisioning indicates that the Managed Airflow environment is being created.
	RevisionStateManagedAirflowEnvironmentProvisioning = style.MustRegisterRevisionState(
		"Environment is being provisioned",
		"deployed_code_history",
		"The Managed Airflow environment is currently being provisioned.",
		style.MustForceConvertSRGBHex("#6666ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	// RevisionStateManagedAirflowEnvironmentExisting indicates that the Managed Airflow environment is active and running.
	RevisionStateManagedAirflowEnvironmentExisting = style.MustRegisterRevisionState(
		"Environment exists",
		"deployed_code",
		"The Managed Airflow environment exists and is active.",
		style.Color{R: 0.0, G: 0.0, B: 1.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	// RevisionStateManagedAirflowEnvironmentDeleting indicates that the Managed Airflow environment is being deleted.
	RevisionStateManagedAirflowEnvironmentDeleting = style.MustRegisterRevisionState(
		"Environment is being deleted",
		"auto_delete",
		"The Managed Airflow environment is undergoing deletion.",
		style.Color{R: 0.8, G: 0.33333334, B: 0.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	// RevisionStateManagedAirflowEnvironmentDeleted indicates that the Managed Airflow environment has been deleted.
	RevisionStateManagedAirflowEnvironmentDeleted = style.MustRegisterRevisionState(
		"Environment is deleted",
		"delete_forever",
		"The Managed Airflow environment has been deleted.",
		style.Color{R: 0.8, G: 0.0, B: 0.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	// RevisionStateManagedAirflowEnvironmentProvisioningLogNotFound indicates that Managed Airflow environment provisioning started before the log collection window.
	RevisionStateManagedAirflowEnvironmentProvisioningLogNotFound = style.MustRegisterRevisionState(
		"Environment is being provisioned, but starting log not found",
		"deployed_code_history",
		"The Managed Airflow environment provisioning was started, but the starting log entry was not found in the selected time range.",
		style.MustForceConvertSRGBHex("#6666ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
	// RevisionStateManagedAirflowEnvironmentExistingLogNotFound indicates that Managed Airflow environment existed before the log collection window.
	RevisionStateManagedAirflowEnvironmentExistingLogNotFound = style.MustRegisterRevisionState(
		"Environment exists, but creation log not found",
		"deployed_code",
		"The Managed Airflow environment exists, but the creation or existence log entry was not found in the selected time range.",
		style.Color{R: 0.0, G: 0.0, B: 1.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
	// RevisionManagedAirflowEnvironmentDeletingLogNotFound indicates that Managed Airflow environment deletion started before the log collection window.
	RevisionManagedAirflowEnvironmentDeletingLogNotFound = style.MustRegisterRevisionState(
		"Environment is being deleted, but starting log not found",
		"auto_delete",
		"The Managed Airflow environment deletion was in progress, but the deletion starting log entry was not found in the selected time range.",
		style.Color{R: 0.8, G: 0.33333334, B: 0.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
)
