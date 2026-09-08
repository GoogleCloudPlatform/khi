// Copyright 2026 Google LLC
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

package commonlogk8saudit_contract

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
)

// EventMessageUIDStarterRunes are delimiters that may precede a Kubernetes resource UID in event messages.
var EventMessageUIDStarterRunes = []rune{'"', ' ', '=', ':', '/', '(', '[', '\'', '#'}

// FormatEventSummary constructs an event log summary and replaces any detected resource UIDs with their readable tags.
func FormatEventSummary(reason, message string, uidFinder patternfinder.PatternFinder[*ResourceIdentity]) string {
	if message == "" {
		return fmt.Sprintf("【%s】", reason)
	}
	matches := patternfinder.FindAllWithStarterRunes(message, uidFinder, true, EventMessageUIDStarterRunes...)
	if len(matches) == 0 {
		return fmt.Sprintf("【%s】%s", reason, message)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【%s】", reason))
	lastIndex := 0
	for _, match := range matches {
		sb.WriteString(message[lastIndex:match.Start])
		sb.WriteString(match.Value.SummaryTag())
		lastIndex = match.End
	}
	sb.WriteString(message[lastIndex:])
	return sb.String()
}
