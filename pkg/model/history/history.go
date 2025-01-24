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

package history

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/binarychunk"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

// The entire inspection data.
type History struct {
	Version   string                 `json:"version"`
	Metadata  map[string]interface{} `json:"metadata"`
	Logs      []*SerializableLog     `json:"logs"`
	Timelines []*ResourceTimeline    `json:"timelines"`
	Resources []*Resource            `json:"resources"`
}

type Resource struct {
	ResourceName     string                  `json:"name"`
	Timeline         string                  `json:"timeline"`
	Relationship     enum.ParentRelationship `json:"relationship"`
	Children         []*Resource             `json:"children"`
	FullResourcePath string                  `json:"path"`
}

type ResourceTimeline struct {
	ID        string              `json:"id"`
	Revisions []*ResourceRevision `json:"revisions"`
	Events    []*ResourceEvent    `json:"events"`
}

type ResourceRevision struct {
	Log        string                       `json:"log"`
	Verb       enum.RevisionVerb            `json:"verb"`
	Requestor  *binarychunk.BinaryReference `json:"requestor"`
	Body       *binarychunk.BinaryReference `json:"body"`
	ChangeTime time.Time                    `json:"changeTime"`
	State      enum.RevisionState           `json:"state"`

	// DEPRECATED: This field is no longer used. Will be removed in near future.
	Partial bool `json:"partial"`
}

type ResourceEvent struct {
	Log string `json:"log"`
}

type SerializableLog struct {
	// Common fields assigned by log entity
	Timestamp time.Time `json:"ts"`
	// ID is an actual unique ID of this log. This field must be unique. KHI uses `insertId`-`timestamp` for GCP log.
	ID string `json:"id"`
	// Display ID is a log ID directly visible to user. This field no need to be unique. KHI uses `insertId` for GCP log.
	DisplayId string                       `json:"displayId"`
	Body      *binarychunk.BinaryReference `json:"body"`

	// These fields are managed by each parsers
	Type        enum.LogType                 `json:"type"`
	Summary     *binarychunk.BinaryReference `json:"summary"`
	Severity    enum.Severity                `json:"severity"`
	Annotations []any                        `json:"annotations"`
}

func NewHistory() *History {
	return &History{
		Version:   "5",
		Timelines: make([]*ResourceTimeline, 0),
		Logs:      make([]*SerializableLog, 0),
		Resources: make([]*Resource, 0),
	}
}

func newTimeline(tid string) *ResourceTimeline {
	return &ResourceTimeline{
		ID:        tid,
		Revisions: make([]*ResourceRevision, 0),
		Events:    make([]*ResourceEvent, 0),
	}
}
