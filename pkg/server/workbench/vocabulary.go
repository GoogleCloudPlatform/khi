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

package workbench

import (
	"maps"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
)

// FilterVocabulary lists the values that filter CEL can match in one inspection.
// It holds only the types that appear in the inspection data, so agents can write filters without guessing.
type FilterVocabulary struct {
	// PathKeys lists the keys of path in timeline CEL, which are the lowercased timeline types.
	PathKeys []string
	// TimelineTypes lists the values of timelineType in timeline CEL.
	TimelineTypes []string
	// LogTypes lists the values of logType in log CEL.
	LogTypes []string
}

// buildFilterVocabulary collects the distinct timeline types and log types in sorted order.
func buildFilterVocabulary(timelines []*cel.TimelineData, logs []cel.LogData, styles cel.StyleResolver) FilterVocabulary {
	timelineTypes := make(map[string]struct{})
	pathKeys := make(map[string]struct{})
	for _, tl := range timelines {
		if tl.TimelineType == "" {
			continue
		}
		timelineTypes[tl.TimelineType] = struct{}{}
		pathKeys[strings.ToLower(tl.TimelineType)] = struct{}{}
	}

	logTypeIDs := make(map[uint32]struct{})
	for _, l := range logs {
		logTypeIDs[l.LogTypeID] = struct{}{}
	}
	logTypes := make(map[string]struct{})
	for id := range logTypeIDs {
		if label := styles.ResolveLogType(id); label != "" {
			logTypes[label] = struct{}{}
		}
	}

	return FilterVocabulary{
		PathKeys:      slices.Sorted(maps.Keys(pathKeys)),
		TimelineTypes: slices.Sorted(maps.Keys(timelineTypes)),
		LogTypes:      slices.Sorted(maps.Keys(logTypes)),
	}
}
