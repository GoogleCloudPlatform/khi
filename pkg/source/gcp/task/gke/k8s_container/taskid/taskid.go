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

package gke_k8s_container_taskid

import (
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var InputContainerQueryNamespacesTaskID = taskid.NewDefaultImplementationID[*queryutil.SetFilterParseResult](gcp_task.GCPPrefix + "input/container-query-namespaces")
var InputContainerQueryPodNamesTaskID = taskid.NewDefaultImplementationID[*queryutil.SetFilterParseResult](gcp_task.GCPPrefix + "input/container-query-podnames")
var GKEContainerLogQueryTaskID = taskid.NewDefaultImplementationID[[]*log.LogEntity](query.GKEQueryPrefix + "k8s-container")
var GKEContainerParserTaskID = taskid.NewDefaultImplementationID[any](gcp_task.GCPPrefix + "feature/container-parser")
