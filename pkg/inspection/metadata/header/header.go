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

package header

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var HeaderMetadataKey = "header"

// Header is a metadata type shown for users in the inspection list page.
type Header struct {
	InspectionType         string `json:"inspectionType"`
	InspectionTypeIconPath string `json:"inspectionTypeIconPath"`
	StartTimeUnixSeconds   int64  `json:"startTimeUnixSeconds"`
	EndTimeUnixSeconds     int64  `json:"endTimeUnixSeconds"`
	InspectTimeUnixSeconds int64  `json:"inspectTimeUnixSeconds"`
	// KHI frontend uses this metadata value for the default value of khi file name on download.
	SuggestedFileName string `json:"suggestedFilename"`
	FileSize          int    `json:"fileSize,omitempty"`
}

var _ metadata.Metadata = (*Header)(nil)

// Labels implements Metadata.
func (*Header) Labels() *task.LabelSet {
	return task.NewLabelSet(metadata.IncludeInRunResult(), metadata.IncludeInDryRunResult(), metadata.IncludeInTaskList(), metadata.IncludeInResultBinary())
}

func (h *Header) ToSerializable() interface{} {
	return h
}

func (h *Header) SetStartTime(startTime time.Time) {
	h.StartTimeUnixSeconds = startTime.Unix()
}

func (h *Header) SetEndTime(endTime time.Time) {
	h.EndTimeUnixSeconds = endTime.Unix()
}

func (h *Header) SetInspectionTime(inspectionTime time.Time) {
	h.InspectTimeUnixSeconds = inspectionTime.Unix()
}

type HeaderMetadataFactory struct {
	DefaultHeader Header
}

// Instanciate implements metadata.MetadataFactory.
func (h *HeaderMetadataFactory) Instanciate() metadata.Metadata {
	return &h.DefaultHeader
}

var _ metadata.MetadataFactory = (*HeaderMetadataFactory)(nil)
