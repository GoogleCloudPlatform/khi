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
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structurev2"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
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

func (c *GCPCommonFieldSetReader) Read(reader *structurev2.NodeReader) (log.FieldSet, error) {
	result := &log.CommonFieldSet{}
	result.DisplayID = reader.ReadStringOrDefault("insertId", "unknown")
	result.Timestamp = reader.ReadTimestampOrDefault("timestamp", time.Time{})
	result.Severity = gcpSeverityToKHISeverity(reader.ReadStringOrDefault("severity", "unknown"))
	return result, nil
}

var _ log.FieldSetReader = (*GCPCommonFieldSetReader)(nil)

// GCPMainMessageFieldSetReader read its main message from the content of log stored on Cloud Logging.
// It treats fields as its main message in the order: `textPayload` > `jsonPayload.****` (**** would be `message`, `msg`...etc)
type GCPMainMessageFieldSetReader struct{}

func (g *GCPMainMessageFieldSetReader) FieldSetKind() string {
	return (&log.MainMessageFieldSet{}).Kind()
}

func (g *GCPMainMessageFieldSetReader) Read(reader *structurev2.NodeReader) (log.FieldSet, error) {
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

var _ log.FieldSetReader = (*GCPMainMessageFieldSetReader)(nil)

// gcpSeverityToKHISeverity convert the `severity` field in Cloud Logging log to the enum.Severity used in KHI.
func gcpSeverityToKHISeverity(severity string) enum.Severity {
	// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#logseverity
	severity = strings.ToUpper(severity)
	switch severity {
	case "DEFAULT":
		return enum.SeverityInfo
	case "DEBUG":
		return enum.SeverityInfo
	case "INFO":
		return enum.SeverityInfo
	case "NOTICE":
		return enum.SeverityInfo
	case "WARNING":
		return enum.SeverityWarning
	case "ERROR":
		return enum.SeverityError
	case "CRITICAL":
		return enum.SeverityFatal
	case "ALERT":
		return enum.SeverityFatal
	case "EMERGENCY":
		return enum.SeverityFatal
	default:
		return enum.SeverityUnknown
	}
}
