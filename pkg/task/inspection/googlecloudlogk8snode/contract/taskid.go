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

// Package googlecloudlogk8snode_contract defines the contract for the googlecloudlogk8snode task.
package googlecloudlogk8snode_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

const (
	// TaskIDPrefix is the prefix for all task IDs in this package.
	TaskIDPrefix = "cloud.google.com/log/k8s-node/"
)

// GKENodeLogQueryTaskID is the task id for the task that queries GKE node logs from Cloud Logging.
var GKENodeLogQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "query")

// GKENodeLogParseTaskID is the task id for the task that parses GKE node logs.
var GKENodeLogParseTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "parser")
