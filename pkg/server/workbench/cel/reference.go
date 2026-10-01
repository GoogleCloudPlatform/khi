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
	"bytes"
	"embed"
	"fmt"
	"slices"
	"strings"
	"text/template"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

//go:embed templates/*.md.tmpl
var referenceTemplateFS embed.FS

var referenceTemplates = template.Must(template.ParseFS(referenceTemplateFS, "templates/*.md.tmpl"))

var defaultTimelinePathKeys = []string{
	"apiversion",
	"kind",
	"namespace",
	"resource",
	"subresource",
}

var standardPathDescriptions = map[string]string{
	"apiversion":  "API version of the resource (e.g. `v1`, `apps/v1`)",
	"kind":        "Kind of the resource (e.g. `Pod`, `Node`, `Deployment`)",
	"namespace":   "Namespace of the resource (e.g. `default`, `kube-system`)",
	"resource":    "Resource type or plural name (e.g. `pods`, `nodes`)",
	"subresource": "Subresource name (e.g. `status`, `exec`)",
}

type timelinePathEntry struct {
	Key          string
	TimelineType string
	Description  string
}

type severityEntry struct {
	Constant string
	Value    int32
	Label    string
}

type logTypeEntry struct {
	Label       string
	Description string
}

type timelineReferenceData struct {
	PathEntries []timelinePathEntry
	Severities  []severityEntry
}

type logReferenceData struct {
	Severities []severityEntry
	LogTypes   []logTypeEntry
}

// GenerateTimelineReference generates Markdown reference documentation for timeline CEL queries.
func GenerateTimelineReference(styleChunk *khifilev6.TimelineStyleChunk) string {
	var pathEntries []timelinePathEntry
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
			pathEntries = append(pathEntries, timelinePathEntry{
				Key:          k,
				TimelineType: lbl,
				Description:  desc,
			})
		}
	}

	for _, k := range defaultTimelinePathKeys {
		if _, exists := seenKeys[k]; !exists {
			if desc, ok := standardPathDescriptions[k]; ok {
				seenKeys[k] = struct{}{}
				pathEntries = append(pathEntries, timelinePathEntry{
					Key:          k,
					TimelineType: "Standard",
					Description:  desc,
				})
			}
		}
	}

	slices.SortFunc(pathEntries, func(a, b timelinePathEntry) int {
		return strings.Compare(a.Key, b.Key)
	})

	data := timelineReferenceData{
		PathEntries: pathEntries,
		Severities:  buildSeverityEntries(styleChunk),
	}

	var buf bytes.Buffer
	if err := referenceTemplates.ExecuteTemplate(&buf, "cel-timeline.md.tmpl", data); err != nil {
		panic(err)
	}
	return buf.String()
}

// GenerateLogReference generates Markdown reference documentation for log CEL queries.
func GenerateLogReference(styleChunk *khifilev6.TimelineStyleChunk) string {
	rawLogTypes := slices.Clone(styleChunk.GetLogTypes())
	slices.SortFunc(rawLogTypes, func(a, b *khifilev6.LogType) int {
		return strings.Compare(a.GetLabel(), b.GetLabel())
	})

	logTypes := make([]logTypeEntry, 0, len(rawLogTypes))
	for _, lt := range rawLogTypes {
		desc := lt.GetDescription()
		if desc == "" {
			desc = lt.GetLabel()
		}
		logTypes = append(logTypes, logTypeEntry{
			Label:       lt.GetLabel(),
			Description: desc,
		})
	}

	data := logReferenceData{
		Severities: buildSeverityEntries(styleChunk),
		LogTypes:   logTypes,
	}

	var buf bytes.Buffer
	if err := referenceTemplates.ExecuteTemplate(&buf, "cel-log.md.tmpl", data); err != nil {
		panic(err)
	}
	return buf.String()
}

func buildSeverityEntries(styleChunk *khifilev6.TimelineStyleChunk) []severityEntry {
	severities := slices.Clone(styleChunk.GetSeverities())
	slices.SortFunc(severities, func(a, b *khifilev6.Severity) int {
		return int(a.GetOrder() - b.GetOrder())
	})
	entries := make([]severityEntry, 0, len(severities))
	for _, sev := range severities {
		entries = append(entries, severityEntry{
			Constant: strings.ToUpper(sev.GetLabel()),
			Value:    sev.GetOrder(),
			Label:    sev.GetLabel(),
		})
	}
	return entries
}
