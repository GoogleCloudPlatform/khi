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

package csmcp

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// TaskIDPrefix is the prefix used for all in-cluster CSM CP task IDs.
const TaskIDPrefix = "googlecloudlogcsmcp.khi.google.com/"

// IstiodLogFilterTaskID is the task ID for filtering container logs to Istiod discovery logs.
var IstiodLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "istiod-log-filter")

// LogGrouperTaskID is the task ID for grouping parsed logs.
var LogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "log-grouper")

// LogToTimelineMapperTaskID is the task ID for mapping log groups to timelines.
var LogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "timeline-mapper")
