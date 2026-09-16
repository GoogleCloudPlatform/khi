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

package caik8s

import (
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

// ClusterResourceTaskIDPrefix is the prefix for Google Cloud CAI Kubernetes cluster resource task IDs.
const ClusterResourceTaskIDPrefix = "cloud.google.com/cai/k8s/"

// GKEResourceTaskIDPrefix is the prefix for Google Cloud CAI GKE resource task IDs.
const GKEResourceTaskIDPrefix = "cloud.google.com/cai/gke/"

// ClusterResourceTaskIDs contains the task implementation IDs for the CAI Kubernetes cluster resource pipeline.
var ClusterResourceTaskIDs = gcpcommon.NewCAITaskIDSet(ClusterResourceTaskIDPrefix)

// GKEResourceTaskIDs contains the task implementation IDs for the CAI GKE cluster and nodepool pipeline.
var GKEResourceTaskIDs = gcpcommon.NewCAITaskIDSet(GKEResourceTaskIDPrefix)
