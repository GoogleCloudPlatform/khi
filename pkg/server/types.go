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

import coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"

type SerializedMetadata = map[string]any

type ServerStat struct {
	TotalMemoryAvailable int `json:"totalMemoryAvailable"`
}

// GetInspectionTypesResponse is the type of the response for /api/v3/inspection/types
type GetInspectionTypesResponse struct {
	Types []*coreinspection.InspectionType `json:"types"`
}

// GetInspectionsResponse is the type of the response for /api/v3/inspection
type GetInspectionsResponse struct {
	Inspections map[string]SerializedMetadata `json:"inspections"`
	ServerStat  *ServerStat                   `json:"serverStat"`
}

type PostInspectionResponse struct {
	InspectionID string `json:"inspectionID"`
}

type PutInspectionFeatureRequest struct {
	Features []string `json:"features"`
}

type PatchInspectionFeatureRequest struct {
	Features map[string]bool `json:"features"`
}

type PutInspectionFeatureResponse struct {
}

type GetInspectionFeatureResponse struct {
	Features []coreinspection.FeatureListItem `json:"features"`
}

type PostInspectionDryRunRequest = map[string]any
