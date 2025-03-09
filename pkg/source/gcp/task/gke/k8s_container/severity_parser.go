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

package k8s_container

import (
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

var MainMessageSeverityParsers = []MainMessageSeverityParser{
	&MetricsContainerLogSeverityParser{},
}

// MainMessageSeverityParser is used for MainMessage extracted from a log entry.
// These parsers are only used when the structured log itself didn't contain severity info and KHI need to read severtiy from the content of log.
type MainMessageSeverityParser interface {
	TryParse(message string) enum.Severity
}

func ParseSeverity(message string) enum.Severity {
	for _, parser := range MainMessageSeverityParsers {
		severity := parser.TryParse(message)
		if severity != enum.SeverityUnknown {
			return severity
		}
	}
	return enum.SeverityUnknown
}

type MetricsContainerLogSeverityParser struct{}

// TryParse implements MainMessageSeverityParser.
func (m *MetricsContainerLogSeverityParser) TryParse(message string) enum.Severity {
	fragments := strings.Split(message, "\t")
	if len(fragments) < 2 {
		return enum.SeverityUnknown
	}
	_, err := time.Parse(time.RFC3339, fragments[0])
	if err != nil {
		return enum.SeverityUnknown
	}
	severityStr := fragments[1]
	switch severityStr {
	case "info":
		return enum.SeverityInfo
	case "warn":
		return enum.SeverityWarning
	case "error":
		return enum.SeverityError
	default:
		return enum.SeverityUnknown
	}
}

var _ (MainMessageSeverityParser) = (*MetricsContainerLogSeverityParser)(nil)
