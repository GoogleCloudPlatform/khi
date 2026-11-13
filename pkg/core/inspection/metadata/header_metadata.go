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

package inspectionmetadata

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

var _ Metadata = (*HeaderMetadata)(nil)

// HeaderMetadata is a metadata type shown for users in the inspection list page.
type HeaderMetadata struct {
	// InspectionType is the identifier of the inspection type.
	InspectionType string `json:"inspectionType"`
	// InspectionName is the human-readable name of the inspection.
	InspectionName string `json:"inspectionName"`
	// InspectionTypeIconPath is the asset path to the icon representing the inspection type.
	InspectionTypeIconPath string `json:"inspectionTypeIconPath"`
	// StartTimeUnixSeconds is the inspection range start time in Unix seconds.
	StartTimeUnixSeconds int64 `json:"startTimeUnixSeconds"`
	// EndTimeUnixSeconds is the inspection range end time in Unix seconds.
	EndTimeUnixSeconds int64 `json:"endTimeUnixSeconds"`
	// InspectTimeUnixSeconds is the timestamp when the inspection was executed, in Unix seconds.
	InspectTimeUnixSeconds int64 `json:"inspectTimeUnixSeconds"`
	// SuggestedFileName is the default KHI file name suggested to the frontend on download.
	SuggestedFileName string `json:"suggestedFilename"`
	// FileSize is the serialized size of the inspection result file in bytes.
	FileSize int `json:"fileSize,omitempty"`
}

// Labels implements Metadata.
func (*HeaderMetadata) Labels() *typedmap.ReadonlyTypedMap {
	return NewLabelSet(IncludeInRunResult(), IncludeInTaskList(), IncludeInResultBinary())
}

// ToSerializable implements Metadata.
func (h *HeaderMetadata) ToSerializable() interface{} {
	return h
}

// SetStartTime records the inspection range start time.
func (h *HeaderMetadata) SetStartTime(startTime time.Time) {
	h.StartTimeUnixSeconds = startTime.Unix()
}

// SetEndTime records the inspection range end time.
func (h *HeaderMetadata) SetEndTime(endTime time.Time) {
	h.EndTimeUnixSeconds = endTime.Unix()
}

// SetInspectionTime records the time when the inspection ran.
func (h *HeaderMetadata) SetInspectionTime(inspectionTime time.Time) {
	h.InspectTimeUnixSeconds = inspectionTime.Unix()
}
