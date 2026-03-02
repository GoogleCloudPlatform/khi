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

package coreinspection

import (
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev4 "github.com/GoogleCloudPlatform/khi/pkg/generated/proto/khifile/v4"
)

// InspectionTaskRegistry is interface for task packages to register its tasks and inspection types.
type InspectionTaskRegistry interface {
	coretask.TaskRegistry

	// AddInspectionType adds a new definition of InspectionType.
	AddInspectionType(newInspectionType InspectionType) error

	// AddSeverity registers a Severity. The ID will be automatically assigned. Returns an error if the ID is already set.
	AddSeverity(severity *khifilev4.Severity) error

	// AddVerb registers a Verb. The ID will be automatically assigned. Returns an error if the ID is already set.
	AddVerb(verb *khifilev4.Verb) error

	// AddLogType registers a LogType. The ID will be automatically assigned. Returns an error if the ID is already set.
	AddLogType(logType *khifilev4.LogType) error

	// AddRevisionState registers a RevisionState. The ID will be automatically assigned. Returns an error if the ID is already set.
	AddRevisionState(revisionState *khifilev4.RevisionState) error

	// AddTimelineType registers a TimelineType. The ID will be automatically assigned. Returns an error if the ID is already set.
	AddTimelineType(timelineType *khifilev4.TimelineType) error
}
