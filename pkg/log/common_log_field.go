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

package log

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

// CommonLogFieldExtractor extracts information being available for all log entries(e.g timestamp)..etc
// These fields can be used outside of parsers specific to log types.
type CommonLogFieldExtractor interface {
	// Get unique ID from a log. Unique but shorter id is preferable.
	ID(l *LogEntity) string
	// Get the timestamp from a log. This is used for sorting.
	Timestamp(l *LogEntity) time.Time
	// Extract the main content of the structured logging.
	// textPayload or jsonPayload.MESSAGE in GCP
	MainMessage(l *LogEntity) (string, error)
	// Severity of this log.
	Severity(l *LogEntity) (enum.Severity, error)
	// ID visible to user. This value isn't necessary to be unique.
	DisplayID(l *LogEntity) string
	// Entire log body represented as a string.
	LogBody(l *LogEntity) string
}
