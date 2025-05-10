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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structurev2"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
)

type GCPCommonFieldSetReader struct{}

func (c *GCPCommonFieldSetReader) FieldSetKind() string {
	return (&log.CommonFieldSet{}).Kind()
}

func (c *GCPCommonFieldSetReader) Read(reader *structurev2.NodeReader) (*log.CommonFieldSet, error) {
	var err error
	result := &log.CommonFieldSet{}
	result.DisplayID = reader.ReadStringOrDefault("insertId", "unknown")
	result.Timestamp, err = reader.ReadTimestamp("timestamp")
	if err != nil {
		return nil, fmt.Errorf("failed to read timestmap from given log")
	}
	result.Severity = gcpSeverityToKHISeverity(reader.ReadStringOrDefault("severity", "unknown"))
	return result, nil
}

var _ log.FieldSetReader[*log.CommonFieldSet] = &GCPCommonFieldSetReader{}

// GCPMainMessageFieldSetReader read its main message from the content of log stored on Cloud Logging.
// It treats fields as its main message in the order: `textPayload` > `jsonPayload.****` (**** would be `message`, `msg`...etc)
type GCPMainMessageFieldSetReader struct{}

func (g *GCPMainMessageFieldSetReader) FieldSetKind() string {
	return (&log.MainMessageFieldSet{}).Kind()
}

func (g *GCPMainMessageFieldSetReader) Read(reader *structurev2.NodeReader) (*log.MainMessageFieldSet, error) {
	result := &log.MainMessageFieldSet{}
	textPayload, err := reader.ReadString("textPayload")
	if err == nil {
		result.MainMessage = textPayload
		return result, nil
	}

	for _, fieldName := range jsonPayloadMessageFieldNames {
		jsonPayloadMessage, err := reader.ReadString(fmt.Sprintf("jsonPayload.%s", fieldName))
		if err == nil {
			result.MainMessage = jsonPayloadMessage
			return result, nil
		}
	}
	return &log.MainMessageFieldSet{}, nil
}
