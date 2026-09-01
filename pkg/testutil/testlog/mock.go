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

package testlog

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// MustLogFromYAML returns a log.Log instance from given YAML string.
// This method is for testing only.
func MustLogFromYAML(text string) *log.Log {
	yamlNode, err := structured.FromYAML(text)
	if err != nil {
		panic(err.Error())
	}
	reader := structured.NewNodeReader(yamlNode)
	l := log.NewLog(reader)
	if ts, err := reader.ReadTimestamp(pathTimestamp); err == nil {
		l.Timestamp = ts
	}
	return l
}

// NewEmptyLogWithID creates a new empty Log with the given ID.
func NewEmptyLogWithID(id string) *log.Log {
	l := log.NewLog(structured.NewNodeReader(structured.NewEmptyMapNode()))
	l.ID = id
	return l
}

// NewMockLog returns a *log.Log populated with the provided typed mock data structures.
func NewMockLog(mockValues ...any) *log.Log {
	mockNode := structured.NewMockNode(mockValues...)
	l := log.NewLog(structured.NewNodeReader(mockNode))
	for _, v := range mockValues {
		if t, ok := v.(time.Time); ok {
			l.Timestamp = t
		}
	}
	return l
}
