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
)

// CommonFieldSet is an abstract FieldSet struct type to get fields commonly defined in logs.
type CommonFieldSet struct {
	// Timestamp is the timestamp of the log happens.
	Timestamp time.Time
}

// Kind implements FieldSet.
func (c *CommonFieldSet) Kind() string {
	return "common"
}

var _ FieldSet = (*CommonFieldSet)(nil)

// MainMessageFieldSet is an abstract FieldSet struct type to get the main message of its log.
// This would be read from `textPayload`, `protoPayload` or `jsonPayload` when it is read from Cloud Logging.
//
// Deprecated: Define custom FieldSet and FieldSetReader instead.
type MainMessageFieldSet struct {
	MainMessage string
}

// Kind implements FieldSet.
func (d *MainMessageFieldSet) Kind() string {
	return "main_message"
}

var _ FieldSet = (*MainMessageFieldSet)(nil)
