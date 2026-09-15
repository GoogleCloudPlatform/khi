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

package composercluster

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

var (
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
