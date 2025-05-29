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

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/parser/k8s"
)

// CommonFieldSet is an abstract FieldSet struct type to get fields commonly defined in logs.
type CommonFieldSet struct {
	// Timestamp is the timestamp of the log happens.
	Timestamp time.Time
	// Severity represents the log severity.
	Severity enum.Severity
	// DisplayID is an unique identifier given from a log.
	// This is only used for showing and it may be same as the ID.
	DisplayID string
}

// Kind implements FieldSet.
func (c *CommonFieldSet) Kind() string {
	return "common"
}

var _ FieldSet = (*CommonFieldSet)(nil)

// MainMessageFieldSet is an abstract FieldSet struct type to get the main message of its log.
// This would be read from `textPayload`, `protoPayload` or `jsonPayload` when it is read from Cloud Logging.
type MainMessageFieldSet struct {
	MainMessage string
}

// Kind implements FieldSet.
func (d *MainMessageFieldSet) Kind() string {
	return "main_message"
}

var _ FieldSet = (*MainMessageFieldSet)(nil)

// KLogField parses the main message as the klog format and returns the field value.
func (m *MainMessageFieldSet) KLogField(fieldName string) (string, error) {
	return k8s.ExtractKLogField(m.MainMessage, fieldName)
}

// HasKLogField parses the main message as the klog format and returns if the field exists.
func (m *MainMessageFieldSet) HasKLogField(fieldName string) bool {
	value, err := m.KLogField(fieldName)
	return err == nil && value != ""
}

// KLogSeverity reads the severity field from KLog formatted message.
func (m *MainMessageFieldSet) KLogSeverity() enum.Severity {
	return k8s.ExractKLogSeverity(m.MainMessage)
}
