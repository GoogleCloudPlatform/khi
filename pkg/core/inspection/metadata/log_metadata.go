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
	"bytes"
	"sort"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// LogMetadata is a Metadata serializes the log data for each tasks.
type LogMetadata struct {
	logBuffers map[string]*bytes.Buffer
	lock       sync.Mutex
}

// NewLogMetadata instanciates an empty LogMetadata.
func NewLogMetadata() *LogMetadata {
	return &LogMetadata{
		logBuffers: map[string]*bytes.Buffer{},
	}
}

// SerializableLogItem is a log data for a specific task.
type SerializableLogItem struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Log  string `json:"log"`
}

// GetTaskLogBuffer creates or get the log string buffer for specified task.
func (l *LogMetadata) GetTaskLogBuffer(taskID taskid.UntypedTaskImplementationID) *bytes.Buffer {
	l.lock.Lock()
	defer l.lock.Unlock()
	if _, found := l.logBuffers[taskID.String()]; !found {
		l.logBuffers[taskID.String()] = new(bytes.Buffer)
	}
	return l.logBuffers[taskID.String()]
}

// Labels implements Metadata.
func (l *LogMetadata) Labels() *typedmap.ReadonlyTypedMap {
	return NewLabelSet(
		IncludeInRunResult(),
	)
}

// ToSerializable implements Metadata.
// It returns a slice of SerializableLogItem, sorted by task ID.
func (l *LogMetadata) ToSerializable() interface{} {
	// Get keys and sort them to ensure a stable order.
	keys := make([]string, 0, len(l.logBuffers))
	for k := range l.logBuffers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]SerializableLogItem, len(keys))
	for i, key := range keys {
		result[i] = SerializableLogItem{
			Id:   key,
			Name: key,
			Log:  l.logBuffers[key].String(),
		}
	}
	return result
}

var _ Metadata = (*LogMetadata)(nil)
