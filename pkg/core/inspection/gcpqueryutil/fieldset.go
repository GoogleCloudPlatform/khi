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

package gcpqueryutil

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

var jsonPayloadMessageFieldNames = []string{
	"MESSAGE",
	"message",
	"msg",
	"log",
}

type GCPCommonFieldSetReader struct{}

func (c *GCPCommonFieldSetReader) FieldSetKind() string {
	return (&log.CommonFieldSet{}).Kind()
}

func (c *GCPCommonFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	result := &log.CommonFieldSet{}
	result.Timestamp = reader.ReadTimestampOrDefault("timestamp", time.Time{})
	return result, nil
}

var _ log.FieldSetReader = (*GCPCommonFieldSetReader)(nil)

// GCPMainMessageFieldSetReader read its main message from the content of log stored on Cloud Logging.
// It treats fields as its main message in the order: `textPayload` > `jsonPayload.****` (**** would be `message`, `msg`...etc) > jsonPayload > labels
type GCPMainMessageFieldSetReader struct{}

func (g *GCPMainMessageFieldSetReader) FieldSetKind() string {
	return (&log.MainMessageFieldSet{}).Kind()
}

func (g *GCPMainMessageFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	result := &log.MainMessageFieldSet{}
	switch {
	case reader.Has("protoPayload"):
		return result, nil
	case reader.Has("textPayload"):
		result.MainMessage = reader.ReadStringOrDefault("textPayload", "")
	case reader.Has("jsonPayload"):
		foundMessageField := false
		for _, fieldName := range jsonPayloadMessageFieldNames {
			jsonPayloadMessage, err := reader.ReadString(fmt.Sprintf("jsonPayload.%s", fieldName))
			if err == nil {
				result.MainMessage = jsonPayloadMessage
				foundMessageField = true
				break
			}
		}
		if !foundMessageField {
			serialized, err := reader.Serialize("jsonPayload", &structured.JSONNodeSerializer{})
			if err != nil {
				return nil, err
			}
			result.MainMessage = string(serialized)
		}
	case reader.Has("labels"):
		serialized, err := reader.Serialize("labels", &structured.JSONNodeSerializer{})
		if err != nil {
			return nil, err
		}
		result.MainMessage = string(serialized)
	}

	return result, nil
}

var _ log.FieldSetReader = (*GCPMainMessageFieldSetReader)(nil)
