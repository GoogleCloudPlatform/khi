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

// RevisionStateK8sResourceExisting is the style for a resource that is existing.
var RevisionStateK8sResourceExisting = style.MustRegisterRevisionState(
	"Resource exists",
	"deployed_code",
	`The Kubernetes resource exists and is active.

**Note**: This state indicates existence in the API server; it does not guarantee that the resource is healthy or fully reconciled.`,
	style.Color{R: 0.0, G: 0.0, B: 1.0, A: 1.0},
	pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
)

// RevisionStateK8sResourceDeleting is the style for a resource that is under deletion with graceful termination period or finalizer.
var RevisionStateK8sResourceDeleting = style.MustRegisterRevisionState(
	"Resource is being deleted",
	"auto_delete",
	"The Kubernetes resource is undergoing deletion (e.g., finalizers are running or in a graceful termination phase).",
	style.Color{R: 0.8, G: 0.33333334, B: 0.0, A: 1.0},
	pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
)

// RevisionStateK8sResourceIsDeleted is the style for a resource that is deleted.
var RevisionStateK8sResourceIsDeleted = style.MustRegisterRevisionState(
	"Resource is deleted",
	"delete_forever",
	"The Kubernetes resource has been deleted.",
	style.Color{R: 0.8, G: 0.0, B: 0.0, A: 1.0},
	pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
)
