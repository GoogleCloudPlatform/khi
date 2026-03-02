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

package inspectioncore_impl

import (
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev4 "github.com/GoogleCloudPlatform/khi/pkg/generated/proto/khifile/v4"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

type registryWithStyleData interface {
	coretask.TaskRegistry
	AddSeverity(severity *khifilev4.Severity) error
	AddVerb(verb *khifilev4.Verb) error
	AddLogType(logType *khifilev4.LogType) error
	AddRevisionState(revisionState *khifilev4.RevisionState) error
	AddTimelineType(timelineType *khifilev4.TimelineType) error
}

func Register(registry registryWithStyleData) error {
	for _, severity := range inspectioncore_contract.Severities {
		if err := registry.AddSeverity(severity); err != nil {
			return err
		}
	}
	for _, verb := range inspectioncore_contract.Verbs {
		if err := registry.AddVerb(verb); err != nil {
			return err
		}
	}
	for _, logType := range inspectioncore_contract.LogTypes {
		if err := registry.AddLogType(logType); err != nil {
			return err
		}
	}
	for _, revisionState := range inspectioncore_contract.RevisionStates {
		if err := registry.AddRevisionState(revisionState); err != nil {
			return err
		}
	}
	for _, timelineType := range inspectioncore_contract.TimelineTypes {
		if err := registry.AddTimelineType(timelineType); err != nil {
			return err
		}
	}
	return coretask.RegisterTasks(registry, InspectionTimeProducer, TimeZoneShiftInputTask, SerializeTask)
}
