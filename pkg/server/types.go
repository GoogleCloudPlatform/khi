// Copyright 2024 Google LLC
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

package server

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
)

type SerializedMetadata = map[string]any

type ServerStat struct {
	TotalMemoryAvailable int `json:"totalMemoryAvailable"`
}

// GetInspectionTypesResponse is the type of the response for /api/v2/inspection/types
type GetInspectionTypesResponse struct {
	Types []*inspection.InspectionType `json:"types"`
}

// GetInspectionTasksResponse is the type of the response for /api/v2/inspection/tasks
type GetInspectionTasksResponse struct {
	Tasks      map[string]SerializedMetadata `json:"tasks"`
	ServerStat *ServerStat                   `json:"serverStat"`
}

type PostInspectionTaskResponse struct {
	InspectionId string `json:"inspectionId"`
}

type PutInspectionTaskFeatureRequest struct {
	Features []string `json:"features"`
}

type PatchInspectionTaskFeatureRequest struct {
	Features map[string]bool `json:"features"`
}

type PutInspectionTaskFeatureResponse struct {
}

type GetInspectionTaskFeatureResponse struct {
	Features []inspection.FeatureListItem `json:"features"`
}

type PostInspectionTaskDryRunRequest = map[string]any
