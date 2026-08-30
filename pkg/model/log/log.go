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

package log

import (
	"strconv"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

var logInstanceID = atomic.Int32{}

// Log represents a log handled in KHI.
// It provides direct access to its fields and basic metadata such as timestamp and instance ID.
type Log struct {
	*structured.NodeReader
	Timestamp time.Time
	ID        string
}

// NewLog returns a log instance from NodeReader instance.
func NewLog(reader *structured.NodeReader) *Log {
	return &Log{
		ID:         strconv.Itoa(int(logInstanceID.Add(1))),
		NodeReader: reader,
	}
}

// NewLogWithTimestamp returns a log instance with the given NodeReader and timestamp.
func NewLogWithTimestamp(reader *structured.NodeReader, timestamp time.Time) *Log {
	return &Log{
		ID:         strconv.Itoa(int(logInstanceID.Add(1))),
		NodeReader: reader,
		Timestamp:  timestamp,
	}
}

// NewLogFromYAMLString instantiates a new Log from the given YAML string.
func NewLogFromYAMLString(yaml string) (*Log, error) {
	node, err := structured.FromYAML(yaml)
	if err != nil {
		return nil, err
	}
	return NewLog(structured.NewNodeReader(node)), nil
}
