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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
)

// Log represents a log handled in KHI.
// It provides direct access to its fields and basic metadata such as timestamp and instance ID.
type Log struct {
	*structured.NodeReader
	Timestamp time.Time
	ID        uint32
}

// NewLog returns a log instance from NodeReader instance using the given ID Generator.
func NewLog(gen *id.Generator, reader *structured.NodeReader) *Log {
	return &Log{
		ID:         gen.New(id.TemporaryLog),
		NodeReader: reader,
	}
}

// NewLogWithTimestamp returns a log instance with the given NodeReader and timestamp using the given ID Generator.
func NewLogWithTimestamp(gen *id.Generator, reader *structured.NodeReader, timestamp time.Time) *Log {
	return &Log{
		ID:         gen.New(id.TemporaryLog),
		NodeReader: reader,
		Timestamp:  timestamp,
	}
}

// NewLogFromYAMLString instantiates a new Log from the given YAML string using the given ID Generator.
func NewLogFromYAMLString(gen *id.Generator, yaml string) (*Log, error) {
	node, err := structured.FromYAML(yaml)
	if err != nil {
		return nil, err
	}
	return NewLog(gen, structured.NewNodeReader(node)), nil
}
