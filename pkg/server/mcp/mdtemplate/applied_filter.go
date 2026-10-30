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

package mdtemplate

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/md"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

// yamlBlock formats s as a YAML literal block scalar indented by two spaces so that query text such as `!has(x)` or `a: b` stays verbatim except for trailing newlines.
// It normalizes CRLF and CR line endings to LF, returns an empty quoted string for an empty s, and adds an explicit indentation indicator when the first non-empty line starts with spaces or tabs because YAML would otherwise detect the indentation from them.
func yamlBlock(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if s == "" {
		return `""`
	}
	header := "|-"
	if body := strings.TrimLeft(s, "\n"); strings.HasPrefix(body, " ") || strings.HasPrefix(body, "\t") {
		header = "|2-"
	}
	return header + "\n  " + strings.ReplaceAll(s, "\n", "\n  ")
}

// FormatAppliedFilter formats an AppliedFilter struct into a markdown explanation with a YAML block.
func FormatAppliedFilter(f workbench.AppliedFilter) string {
	startTime := `""`
	if !f.StartTime.IsZero() {
		startTime = FormatTime(f.StartTime)
	}
	endTime := `""`
	if !f.EndTime.IsZero() {
		endTime = FormatTime(f.EndTime)
	}

	var yamlBody strings.Builder
	fmt.Fprintf(&yamlBody, "timelineQuery: %s\n", yamlBlock(f.TimelineQuery))
	fmt.Fprintf(&yamlBody, "timelineExclusionQuery: %s\n", yamlBlock(f.TimelineExclusionQuery))
	fmt.Fprintf(&yamlBody, "logQuery: %s\n", yamlBlock(f.LogQuery))
	fmt.Fprintf(&yamlBody, "startTime: %s\n", startTime)
	fmt.Fprintf(&yamlBody, "endTime: %s\n", endTime)
	fmt.Fprintf(&yamlBody, "excludeTimelinesWithoutLogs: %t", f.ExcludeTimelinesWithoutLogs)

	var sb strings.Builder
	sb.WriteString("## Applied filter\n")
	sb.WriteString("Enter these values in the KHI Web UI filter to see the same data.\n\n")
	sb.WriteString(md.Fence("yaml", yamlBody.String()))
	return sb.String()
}
