// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package googlecloudlogk8scontainer_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

const TaskIDPrefix = "cloud.google.com/log/k8s-container/"

// InputContainerQueryNamespacesTaskID is the task ID for the form input that specifies which namespaces to query for container logs.
var InputContainerQueryNamespacesTaskID = taskid.NewDefaultImplementationID[*gcpqueryutil.SetFilterParseResult](TaskIDPrefix + "input/query-namespaces")

// InputContainerQueryPodNamesTaskID is the task ID for the form input that specifies which pod names to query for container logs.
var InputContainerQueryPodNamesTaskID = taskid.NewDefaultImplementationID[*gcpqueryutil.SetFilterParseResult](TaskIDPrefix + "input/query-podnames")

// GKEContainerLogQueryTaskID is the task ID for the task that queries GKE container logs from Cloud Logging.
var GKEContainerLogQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "query")

// GKEContainerParserTaskID is the task ID for the task that parses GKE container logs.
var GKEContainerParserTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "parser")
