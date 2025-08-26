// Copyright 2025 Google LLC
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

// Package googlecloudlogk8scontrolplane_contract defines the contract for tasks related to GKE control plane component logs.
package googlecloudlogk8scontrolplane_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// K8sControlPlaneLogTaskIDPrefix is the prefix for all task IDs in this package.
const K8sControlPlaneLogTaskIDPrefix = "cloud.google.com/log/k8s-control-plane/"

// InputControlPlaneComponentNameFilterTaskID is the task ID for the form task that inputs the control plane component name filter.
var InputControlPlaneComponentNameFilterTaskID = taskid.NewDefaultImplementationID[*gcpqueryutil.SetFilterParseResult](K8sControlPlaneLogTaskIDPrefix + "input/component-names")

// GKEK8sControlPlaneComponentQueryTaskID is the task ID for the task that queries GKE control plane component logs.
var GKEK8sControlPlaneComponentQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](K8sControlPlaneLogTaskIDPrefix + "query")

// GKEK8sControlPlaneComponentParserTaskID is the task ID for the task that parses GKE control plane component logs.
var GKEK8sControlPlaneComponentParserTaskID = taskid.NewDefaultImplementationID[struct{}](K8sControlPlaneLogTaskIDPrefix + "parser")
