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

// package googlecloudlognetworkapiaudit_contract defines the task IDs for the googlecloudlognetworkapiaudit inspection tasks.
package googlecloudlognetworkapiaudit_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// NetworkAPILogTaskIDPrefix is the prefix for all task IDs in this package.
var NetworkAPILogTaskIDPrefix = "cloud.google.com/log/network-api/"

// NetworkAPIQueryTaskID is the task id for the task that queries network API logs from Cloud Logging.
var NetworkAPIQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](NetworkAPILogTaskIDPrefix + "query")

// NetworkAPIParserTaskID is the task id for the task that parses network API logs.
var NetworkAPIParserTaskID = taskid.NewDefaultImplementationID[struct{}](NetworkAPILogTaskIDPrefix + "parser")
