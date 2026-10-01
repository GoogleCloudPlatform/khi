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

package cel

import (
	"fmt"
	"slices"
	"strings"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

var defaultTimelinePathKeys = []string{
	"apiversion",
	"kind",
	"namespace",
	"resource",
	"subresource",
}

// GenerateTimelineReference generates Markdown reference documentation for timeline CEL queries.
func GenerateTimelineReference(styleChunk *khifilev6.TimelineStyleChunk) string {
	var b strings.Builder

	b.WriteString("# Timeline CEL Reference\n\n")
	b.WriteString("Timeline CEL expressions filter which resource timelines are displayed or excluded in KHI. Expressions must evaluate to a boolean value (`bool`).\n\n")

	b.WriteString("## Variables\n\n")
	b.WriteString("The following variables are available in the timeline evaluation scope:\n\n")
	b.WriteString("- `name` (`string`): The name of the resource represented by the timeline (e.g. pod name, node name).\n")
	b.WriteString("- `timelineType` (`string`): The type of the timeline (e.g. `Pod`, `Node`, `Namespace`).\n")
	b.WriteString("- `path` (`map<string, string>`): Hierarchical resource path properties (e.g. `path[\"namespace\"]`, `path[\"kind\"]`, `path[\"resource\"]`).\n")
	b.WriteString("- `t` (`map<string, dyn>`): The root timeline object, exposing `t.name`, `t.timelineType`, and `t.path`.\n\n")

	b.WriteString("## Timeline Path Keys\n\n")
	b.WriteString("The following keys can be accessed via `path[\"<key>\"]`:\n\n")
	b.WriteString("| Key | Timeline Type | Description |\n")
	b.WriteString("| --- | --- | --- |\n")

	standardDescriptions := map[string]string{
		"apiversion":  "API version of the resource (e.g. `v1`, `apps/v1`)",
		"kind":        "Kind of the resource (e.g. `Pod`, `Node`, `Deployment`)",
		"namespace":   "Namespace of the resource (e.g. `default`, `kube-system`)",
		"resource":    "Resource type or plural name (e.g. `pods`, `nodes`)",
		"subresource": "Subresource name (e.g. `status`, `exec`)",
	}

	type pathEntry struct {
		key          string
		timelineType string
		description  string
	}
	var pathEntries []pathEntry
	seenKeys := make(map[string]struct{})

	for _, tt := range styleChunk.GetTimelineTypes() {
		lbl := tt.GetLabel()
		k := strings.ToLower(lbl)
		if k == "" || strings.HasPrefix(k, "@") {
			continue
		}
		if _, exists := seenKeys[k]; !exists {
			seenKeys[k] = struct{}{}
			desc := tt.GetDescription()
			if desc == "" {
				desc = fmt.Sprintf("Timeline of type %s", lbl)
			}
			pathEntries = append(pathEntries, pathEntry{
				key:          k,
				timelineType: lbl,
				description:  desc,
			})
		}
	}

	for _, k := range defaultTimelinePathKeys {
		if _, exists := seenKeys[k]; !exists {
			if desc, ok := standardDescriptions[k]; ok {
				seenKeys[k] = struct{}{}
				pathEntries = append(pathEntries, pathEntry{
					key:          k,
					timelineType: "Standard",
					description:  desc,
				})
			}
		}
	}

	slices.SortFunc(pathEntries, func(a, b pathEntry) int {
		return strings.Compare(a.key, b.key)
	})

	for _, pe := range pathEntries {
		b.WriteString(fmt.Sprintf("| `path[%q]` | %s | %s |\n", pe.key, pe.timelineType, pe.description))
	}
	b.WriteString("\n")

	b.WriteString("## Severity Constants\n\n")
	b.WriteString("Severities are integer constants used with `hasSeverity` and `minSeverity`:\n\n")
	b.WriteString("| Constant | Value | Description |\n")
	b.WriteString("| --- | --- | --- |\n")

	severities := slices.Clone(styleChunk.GetSeverities())
	slices.SortFunc(severities, func(a, b *khifilev6.Severity) int {
		return int(a.GetOrder() - b.GetOrder())
	})
	for _, sev := range severities {
		cName := strings.ToUpper(sev.GetLabel())
		b.WriteString(fmt.Sprintf("| `%s` | %d | %s severity |\n", cName, sev.GetOrder(), sev.GetLabel()))
	}
	b.WriteString("\n")

	b.WriteString("## Functions\n\n")
	b.WriteString("### `match` / `M`\n\n")
	b.WriteString("Matches values in the timeline path against substrings or regular expressions.\n\n")
	b.WriteString("- `match(value string) bool` / `M(value string) bool`:\n")
	b.WriteString("  Checks if any value across all timeline path keys contains `value` (case-insensitive substring or regex).\n")
	b.WriteString("- `match(values list<string>) bool` / `M(values list<string>) bool`:\n")
	b.WriteString("  Checks if any value across all timeline path keys matches any pattern in `values`.\n")
	b.WriteString("- `match(key string, value string) bool` / `M(key string, value string) bool`:\n")
	b.WriteString("  Checks if the path property `key` contains `value` (case-insensitive substring or regex).\n")
	b.WriteString("- `match(key string, values list<string>) bool` / `M(key string, values list<string>) bool`:\n")
	b.WriteString("  Checks if the path property `key` matches any pattern in `values`.\n\n")

	b.WriteString("### `revision_body` / `RB`\n\n")
	b.WriteString("Matches YAML/JSON field paths in recorded resource revision bodies.\n\n")
	b.WriteString("- `revision_body(value string) bool` / `RB(value string) bool`:\n")
	b.WriteString("  Checks if any revision body contains `value` anywhere in its contents.\n")
	b.WriteString("- `revision_body(values list<string>) bool` / `RB(values list<string>) bool`:\n")
	b.WriteString("  Checks if any revision body contains any pattern in `values`.\n")
	b.WriteString("- `revision_body(fieldPath string, value string) bool` / `RB(fieldPath string, value string) bool`:\n")
	b.WriteString("  Checks if the dot-separated field `fieldPath` in any revision body matches `value`.\n")
	b.WriteString("- `revision_body(fieldPath string, values list<string>) bool` / `RB(fieldPath string, values list<string>) bool`:\n")
	b.WriteString("  Checks if the dot-separated field `fieldPath` in any revision body matches any pattern in `values`.\n\n")

	b.WriteString("### `minSeverity`\n\n")
	b.WriteString("- `minSeverity(severity int) bool`:\n")
	b.WriteString("  Returns `true` if the timeline contains at least one event or log with a severity level equal to or greater than the given constant.\n\n")

	b.WriteString("### `hasSeverity`\n\n")
	b.WriteString("- `hasSeverity(severity int) bool`:\n")
	b.WriteString("  Returns `true` if the timeline contains at least one event or log with the exact given severity level.\n")
	b.WriteString("- `hasSeverity(severities list<int>) bool`:\n")
	b.WriteString("  Returns `true` if the timeline contains at least one event or log matching any severity level in `severities`.\n\n")

	b.WriteString("## Common Patterns\n\n")
	b.WriteString("Filter by namespace:\n\n")
	b.WriteString("```cel\npath[\"namespace\"] == \"kube-system\"\n```\n\n")
	b.WriteString("Filter by resource kind:\n\n")
	b.WriteString("```cel\npath[\"kind\"] == \"Pod\"\n```\n\n")
	b.WriteString("Filter by resource name regex:\n\n")
	b.WriteString("```cel\nmatch(\"resource\", \"^coredns-.*\")\n```\n\n")
	b.WriteString("Filter timelines with errors:\n\n")
	b.WriteString("```cel\nminSeverity(ERROR)\n```\n\n")
	b.WriteString("Filter by revision body field:\n\n")
	b.WriteString("```cel\nrevision_body(\"status.phase\", \"Failed\")\n```\n\n")
	b.WriteString("Combine conditions:\n\n")
	b.WriteString("```cel\npath[\"kind\"] == \"Pod\" && path[\"namespace\"] == \"default\" && minSeverity(WARNING)\n```\n")

	return b.String()
}

// GenerateLogReference generates Markdown reference documentation for log CEL queries.
func GenerateLogReference(styleChunk *khifilev6.TimelineStyleChunk) string {
	var b strings.Builder

	b.WriteString("# Log CEL Reference\n\n")
	b.WriteString("Log CEL expressions filter individual log entries within timelines. Logs matching the expression remain visible, while non-matching logs are hidden or dimmed. Expressions must evaluate to a boolean value (`bool`).\n\n")

	b.WriteString("## Variables\n\n")
	b.WriteString("The following variables are available in the log evaluation scope:\n\n")
	b.WriteString("- `logType` (`string`): The type identifier of the log entry (e.g. `\"k8s audit\"`, `\"container\"`).\n")
	b.WriteString("- `severity` (`int`): The integer severity level of the log entry (`INFO`, `WARNING`, `ERROR`, `FATAL`).\n")
	b.WriteString("- `l` (`map<string, dyn>`): The root log object, exposing `l.logType` and `l.severity`.\n\n")

	b.WriteString("## Severity Constants\n\n")
	b.WriteString("Severity levels can be compared using standard comparison operators (`==`, `!=`, `<`, `<=`, `>`, `>=`):\n\n")
	b.WriteString("| Constant | Value | Description |\n")
	b.WriteString("| --- | --- | --- |\n")

	severities := slices.Clone(styleChunk.GetSeverities())
	slices.SortFunc(severities, func(a, b *khifilev6.Severity) int {
		return int(a.GetOrder() - b.GetOrder())
	})
	for _, sev := range severities {
		cName := strings.ToUpper(sev.GetLabel())
		b.WriteString(fmt.Sprintf("| `%s` | %d | %s severity |\n", cName, sev.GetOrder(), sev.GetLabel()))
	}
	b.WriteString("\n")

	b.WriteString("## Registered Log Types\n\n")
	b.WriteString("The following log types are registered in KHI:\n\n")
	b.WriteString("| `Label` | Description |\n")
	b.WriteString("| --- | --- |\n")

	logTypes := slices.Clone(styleChunk.GetLogTypes())
	slices.SortFunc(logTypes, func(a, b *khifilev6.LogType) int {
		return strings.Compare(a.GetLabel(), b.GetLabel())
	})
	for _, lt := range logTypes {
		desc := lt.GetDescription()
		if desc == "" {
			desc = lt.GetLabel()
		}
		b.WriteString(fmt.Sprintf("| `%s` | %s |\n", lt.GetLabel(), desc))
	}
	b.WriteString("\n")

	b.WriteString("## Functions\n\n")
	b.WriteString("### `body` / `B`\n\n")
	b.WriteString("Matches contents within the log entry body against substrings or regular expressions.\n\n")
	b.WriteString("- `body(value string) bool` / `B(value string) bool`:\n")
	b.WriteString("  Checks if any field in the log body contains `value` (case-insensitive substring or regex).\n")
	b.WriteString("- `body(values list<string>) bool` / `B(values list<string>) bool`:\n")
	b.WriteString("  Checks if any field in the log body matches any pattern in `values`.\n")
	b.WriteString("- `body(fieldPath string, value string) bool` / `B(fieldPath string, value string) bool`:\n")
	b.WriteString("  Checks if the dot-separated field `fieldPath` in the log body matches `value`.\n")
	b.WriteString("- `body(fieldPath string, values list<string>) bool` / `B(fieldPath string, values list<string>) bool`:\n")
	b.WriteString("  Checks if the dot-separated field `fieldPath` in the log body matches any pattern in `values`.\n\n")

	b.WriteString("## Common Patterns\n\n")
	b.WriteString("Filter by minimum severity:\n\n")
	b.WriteString("```cel\nseverity >= WARNING\n```\n\n")
	b.WriteString("Filter only errors and fatal logs:\n\n")
	b.WriteString("```cel\nseverity >= ERROR\n```\n\n")
	b.WriteString("Filter by specific log type:\n\n")
	b.WriteString("```cel\nlogType == \"k8s audit\"\n```\n\n")
	b.WriteString("Search for keywords in log message:\n\n")
	b.WriteString("```cel\nbody(\"OOMKilled\")\n```\n\n")
	b.WriteString("Search within a specific JSON/structured field:\n\n")
	b.WriteString("```cel\nbody(\"protoPayload.methodName\", \"delete\")\n```\n\n")
	b.WriteString("Combine severity and content filters:\n\n")
	b.WriteString("```cel\nseverity >= ERROR && body(\"connection refused\")\n```\n")

	return b.String()
}
