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

package googlecloudlogk8snode_contract

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

var (
	// RevisionStateComponentRunning represents that a node component is running successfully.
	RevisionStateComponentRunning = style.MustRegisterRevisionState(
		"Component is running",
		"check_circle",
		"The node component (e.g., kubelet, kube-proxy) is running normally.",
		style.MustForceConvertSRGBHex("#00cc66"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateComponentTerminated represents that a node component has terminated or stopped.
	RevisionStateComponentTerminated = style.MustRegisterRevisionState(
		"Component is terminated",
		"cancel",
		"The node component has terminated or stopped.",
		style.MustForceConvertSRGBHex("#ff3333"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_DELETED,
	)
)
