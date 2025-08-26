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

package googlecloudcommon_contract

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// GoogleCloudCommonTaskIDPrefix is the prefix for Google Cloud common task IDs.
var GoogleCloudCommonTaskIDPrefix = "cloud.google.com/common/"

// AutocompleteLocationTaskID is the task ID for the location autocomplete.
var AutocompleteLocationTaskID taskid.TaskImplementationID[[]string] = taskid.NewDefaultImplementationID[[]string](GoogleCloudCommonTaskIDPrefix + "autocomplete-location")

// Common forms over Google Cloud related packages.

// InputProjectIdTaskID is the task ID for the Google Cloud project ID.
var InputProjectIdTaskID = taskid.NewDefaultImplementationID[string](GoogleCloudCommonTaskIDPrefix + "input-project-id")

// InputLoggingFilterResourceNameTaskID is the task ID to get log query target resource names.
var InputLoggingFilterResourceNameTaskID = taskid.NewDefaultImplementationID[*ResourceNamesInput](GoogleCloudCommonTaskIDPrefix + "input-logging-filter-resource-name")

// InputDurationTaskID is the task ID for the duration of the log query.
var InputDurationTaskID = taskid.NewDefaultImplementationID[time.Duration](GoogleCloudCommonTaskIDPrefix + "input-duration")

// InputEndTimeTaskID is the task ID for the end time of the log query.
var InputEndTimeTaskID = taskid.NewDefaultImplementationID[time.Time](GoogleCloudCommonTaskIDPrefix + "input-end-time")

// InputStartTimeTaskID is the task ID for the start time of the log query. This is computed from InputDurationTask and InputEndTimeTask.
var InputStartTimeTaskID = taskid.NewDefaultImplementationID[time.Time](GoogleCloudCommonTaskIDPrefix + "input-start-time")

// InputLocationsTaskID is the task ID for the locations of the target resource.
var InputLocationsTaskID = taskid.NewDefaultImplementationID[string](GoogleCloudCommonTaskIDPrefix + "input-location")
