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

package multicloud_api_taskid

import (
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var MultiCloudAPIQueryTaskID = taskid.NewDefaultImplementationID[[]*log.LogEntity](query.GKEQueryPrefix + "multicloud-api")
var MultiCloudAPIParserTaskID = taskid.NewDefaultImplementationID[any](gcp_task.GCPPrefix + "feature/multicloud-audit-parser")
