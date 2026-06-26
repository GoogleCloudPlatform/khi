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
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style RevisionStates.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	// =============================================================================
	// Kubernetes Cluster States
	// =============================================================================

	RevisionStateK8sClusterProvisioning = style.MustRegisterRevisionState(
		"Cluster is being provisioned",
		"deployed_code_history",
		"The Kubernetes cluster is currently being provisioned.",
		style.MustForceConvertSRGBHex("#6666ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateK8sClusterExisting = style.MustRegisterRevisionState(
		"Cluster exists",
		"deployed_code",
		"The Kubernetes cluster exists and is active.",
		style.Color{R: 0.0, G: 0.0, B: 1.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateK8sClusterDeleting = style.MustRegisterRevisionState(
		"Cluster is being deleted",
		"auto_delete",
		"The Kubernetes cluster is undergoing deletion.",
		style.Color{R: 0.8, G: 0.33333334, B: 0.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateK8sClusterDeleted = style.MustRegisterRevisionState(
		"Cluster is deleted",
		"delete_forever",
		"The Kubernetes cluster has been deleted.",
		style.Color{R: 0.8, G: 0.0, B: 0.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateK8sClusterProvisioningLogNotFound = style.MustRegisterRevisionState(
		"Cluster is being provisioned, but starting log not found",
		"deployed_code_history",
		"The Kubernetes cluster provisioning was started, but the starting log entry was not found in the selected time range.",
		style.MustForceConvertSRGBHex("#6666ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
	RevisionStateK8sClusterExistingLogNotFound = style.MustRegisterRevisionState(
		"Cluster exists, but creation log not found",
		"deployed_code",
		"The Kubernetes cluster exists, but the creation or existence log entry was not found in the selected time range.",
		style.Color{R: 0.0, G: 0.0, B: 1.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
	RevisionStateK8sClusterDeletingLogNotFound = style.MustRegisterRevisionState(
		"Cluster is being deleted, but starting log not found",
		"auto_delete",
		"The Kubernetes cluster deletion was in progress, but the deletion starting log entry was not found in the selected time range.",
		style.Color{R: 0.8, G: 0.33333334, B: 0.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)

	// =============================================================================
	// Kubernetes Node Pool States
	// =============================================================================

	RevisionStateK8sNodepoolProvisioning = style.MustRegisterRevisionState(
		"Node pool is being provisioned",
		"deployed_code_history",
		"The Kubernetes node pool is currently being provisioned.",
		style.MustForceConvertSRGBHex("#6666ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateK8sNodepoolExisting = style.MustRegisterRevisionState(
		"Node pool exists",
		"deployed_code",
		"The Kubernetes node pool exists and is active.",
		style.Color{R: 0.0, G: 0.0, B: 1.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateK8sNodepoolDeleting = style.MustRegisterRevisionState(
		"Node pool is being deleted",
		"auto_delete",
		"The Kubernetes node pool is undergoing deletion.",
		style.Color{R: 0.8, G: 0.33333334, B: 0.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateK8sNodepoolDeleted = style.MustRegisterRevisionState(
		"Node pool is deleted",
		"delete_forever",
		"The Kubernetes node pool has been deleted.",
		style.Color{R: 0.8, G: 0.0, B: 0.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateK8sNodepoolProvisioningLogNotFound = style.MustRegisterRevisionState(
		"Node pool is being provisioned, but starting log not found",
		"deployed_code_history",
		"The Kubernetes node pool provisioning was started, but the starting log entry was not found in the selected time range.",
		style.MustForceConvertSRGBHex("#6666ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
	RevisionStateK8sNodepoolExistingLogNotFound = style.MustRegisterRevisionState(
		"Node pool exists, but creation log not found",
		"deployed_code",
		"The Kubernetes node pool exists, but the creation or existence log entry was not found in the selected time range.",
		style.Color{R: 0.0, G: 0.0, B: 1.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
	RevisionStateK8sNodepoolDeletingLogNotFound = style.MustRegisterRevisionState(
		"Node pool is being deleted, but starting log not found",
		"auto_delete",
		"The Kubernetes node pool deletion was in progress, but the deletion starting log entry was not found in the selected time range.",
		style.Color{R: 0.8, G: 0.33333334, B: 0.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)

	// =============================================================================
	// Kubernetes Generic Resource States
	// =============================================================================

	RevisionStateK8sResourceExisting = style.MustRegisterRevisionState(
		"Resource exists",
		"deployed_code",
		`The Kubernetes resource exists and is active.

**Note**: This state indicates existence in the API server; it does not guarantee that the resource is healthy or fully reconciled.`,
		style.Color{R: 0.0, G: 0.0, B: 1.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateK8sResourceExistingLogNotFound = style.MustRegisterRevisionState(
		"Resource exists, but creation log not found",
		"deployed_code",
		"The Kubernetes resource exists, but the creation or existence log entry was not found in the selected time range.",
		style.Color{R: 0.0, G: 0.0, B: 1.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
	RevisionStateK8sResourceDeleting = style.MustRegisterRevisionState(
		"Resource is being deleted",
		"auto_delete",
		"The Kubernetes resource is undergoing deletion (e.g., finalizers are running or in a graceful termination phase).",
		style.Color{R: 0.8, G: 0.33333334, B: 0.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateK8sResourceDeleted = style.MustRegisterRevisionState(
		"Resource is deleted",
		"delete_forever",
		"The Kubernetes resource has been deleted.",
		style.Color{R: 0.8, G: 0.0, B: 0.0, A: 1.0},
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)

	// =============================================================================
	// Pod Phase States
	// =============================================================================

	RevisionStatePodPhasePending = style.MustRegisterRevisionState(
		"Pod is pending",
		"hourglass_empty",
		"The pod is accepted by the Kubernetes cluster, but one or more containers are not yet running.",
		style.MustForceConvertSRGBHex("#666666"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStatePodPhaseScheduled = style.MustRegisterRevisionState(
		"Pod is scheduled",
		"schedule",
		"The pod is scheduled to run on a specific node.",
		style.MustForceConvertSRGBHex("#4444ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStatePodPhaseRunning = style.MustRegisterRevisionState(
		"Pod is running",
		"motion_play",
		"The pod is running on a node, and at least one container is currently active.",
		style.MustForceConvertSRGBHex("#004400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStatePodPhaseSucceeded = style.MustRegisterRevisionState(
		"Pod has succeeded",
		"check_circle",
		"The pod has succeeded (all containers have terminated successfully and will not be restarted).",
		style.MustForceConvertSRGBHex("#113333"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStatePodPhaseFailed = style.MustRegisterRevisionState(
		"Pod has failed",
		"error",
		"The pod has failed (all containers have terminated, and at least one container terminated in failure).",
		style.MustForceConvertSRGBHex("#331111"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStatePodPhaseUnknown = style.MustRegisterRevisionState(
		"Pod status is unavailable",
		"unknown_document",
		`The pod status is undetermined due to missing logs in the selected time range.

**Tip**: Consider expanding the query time range to capture complete pod lifecycle events.`,
		style.MustForceConvertSRGBHex("#997700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)

	// =============================================================================
	// Container States
	// =============================================================================

	RevisionStateContainerWaiting = style.MustRegisterRevisionState(
		"Container is waiting",
		"deployed_code_history",
		"The container is waiting to start (e.g., image pull or init container execution is in progress).",
		style.MustForceConvertSRGBHex("#4444ff"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateContainerRunningNonReady = style.MustRegisterRevisionState(
		"Container is not ready",
		"heart_broken",
		"The container is running but has not passed its readiness probe.",
		style.MustForceConvertSRGBHex("#EE4400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateContainerRunningReady = style.MustRegisterRevisionState(
		"Container is ready",
		"heart_check",
		"The container is running and has successfully passed its readiness probe.",
		style.MustForceConvertSRGBHex("#007700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateContainerTerminatedWithSuccess = style.MustRegisterRevisionState(
		"Container exited successfully",
		"check_circle",
		"The container has terminated successfully with an exit code of `0`.",
		style.MustForceConvertSRGBHex("#113333"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateContainerTerminatedWithError = style.MustRegisterRevisionState(
		"Container exited with error",
		"error",
		"The container has terminated in failure with a non-zero exit code.",
		style.MustForceConvertSRGBHex("#551111"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateContainerStatusNotAvailable = style.MustRegisterRevisionState(
		"Container status is unavailable",
		"unknown_document",
		`The container status is unknown or not yet reported in the log data.

**Tip**: Consider expanding the query time range to capture complete container status events.`,
		style.MustForceConvertSRGBHex("#666666"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
	RevisionStateContainerStarted = style.MustRegisterRevisionState(
		"Container is started, readiness unknown",
		"siren_question",
		`The container has started, but no readiness probe information has been recorded yet.`,
		style.MustForceConvertSRGBHex("#997700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)

	// =============================================================================
	// Endpoint States
	// =============================================================================

	RevisionStateEndpointReady = style.MustRegisterRevisionState(
		"Endpoint is ready",
		"heart_check",
		"The endpoint is active and ready to receive traffic.",
		style.MustForceConvertSRGBHex("#004400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateEndpointUnready = style.MustRegisterRevisionState(
		"Endpoint is not ready",
		"heart_broken",
		"The endpoint is not ready to receive traffic (e.g., the backing pod is unready).",
		style.MustForceConvertSRGBHex("#EE4400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateEndpointTerminating = style.MustRegisterRevisionState(
		"Endpoint is being terminated",
		"auto_delete",
		"The endpoint is in the process of being deleted or terminated.",
		style.MustForceConvertSRGBHex("#cea700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// =============================================================================
	// Condition States
	// =============================================================================

	RevisionStateConditionTrue = style.MustRegisterRevisionState(
		"Condition is True",
		"lightbulb",
		`The condition is set to **True**.

**Note**: **True** does not always indicate a healthy state (e.g., Ready=True is healthy, but DiskPressure=True is unhealthy).`,
		style.MustForceConvertSRGBHex("#004400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateConditionFalse = style.MustRegisterRevisionState(
		"Condition is False",
		"light_off",
		`The condition is set to **False**.

**Note**: **False** does not always indicate an unhealthy state (e.g., DiskPressure=False is healthy, but Ready=False is unhealthy).`,
		style.MustForceConvertSRGBHex("#EE4400"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateConditionUnknown = style.MustRegisterRevisionState(
		"Condition is Unknown",
		"siren_question",
		"The condition is `Unknown`, meaning its state cannot be determined at this moment.",
		style.MustForceConvertSRGBHex("#663366"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateConditionNotGiven = style.MustRegisterRevisionState(
		"Condition is not defined",
		"select",
		"The condition has not yet been set or reported at this point in the timeline.",
		style.MustForceConvertSRGBHex("#666666"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
	RevisionStateConditionNoAvailableInfo = style.MustRegisterRevisionState(
		"Condition info is unavailable",
		"unknown_document",
		`The condition status is undetermined due to insufficient log information in the selected range.

**Tip**: Consider expanding the query time range to capture complete condition reports.`,
		style.MustForceConvertSRGBHex("#997700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
)
