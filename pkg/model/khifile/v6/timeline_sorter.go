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

package khifilev6

import (
	"iter"
	"slices"
	"strings"
	"time"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

// CompareAlphabetical compares two timeline paths alphabetically, respecting any prioritized names map.
func CompareAlphabetical(a, b *TimelinePath, prioritizedNames map[string]int) int {
	aName := a.Name.Resolve()
	bName := b.Name.Resolve()

	if len(prioritizedNames) > 0 {
		aIdx, aOk := prioritizedNames[aName]
		bIdx, bOk := prioritizedNames[bName]

		if aOk && bOk {
			return aIdx - bIdx
		}
		if aOk {
			return -1
		}
		if bOk {
			return 1
		}
	}

	return strings.Compare(aName, bName)
}

// CompareChronological compares two timeline paths chronologically by finding their oldest event or revision.
func CompareChronological(a, b *TimelinePath, registry *TimelineRegistry, parentToChildren map[*TimelinePath][]*TimelinePath, maxDepth int32) int {
	aTime, aOk := FindOldestTime(a, registry, parentToChildren, 0, maxDepth)
	bTime, bOk := FindOldestTime(b, registry, parentToChildren, 0, maxDepth)

	switch {
	case aOk && bOk:
		if cmp := aTime.Compare(bTime); cmp != 0 {
			return cmp
		}
	case aOk:
		return -1
	case bOk:
		return 1
	}

	// Fallback to alphabetical if timestamps are equal or both missing.
	return strings.Compare(a.Name.Resolve(), b.Name.Resolve())
}

// CompareGroupedChronological compares two timeline paths hierarchically by splitting their names with delimiter,
// ordering common prefix groups chronologically by their oldest log timestamp.
func CompareGroupedChronological(a, b *TimelinePath, registry *TimelineRegistry, parentToChildren map[*TimelinePath][]*TimelinePath, delimiter string) int {
	return CompareGroupedChronologicalWithMap(a, b, registry, parentToChildren, delimiter, nil)
}

// CompareGroupedChronologicalWithMap compares two timeline paths hierarchically using a pre-calculated prefix timestamp map.
func CompareGroupedChronologicalWithMap(a, b *TimelinePath, registry *TimelineRegistry, parentToChildren map[*TimelinePath][]*TimelinePath, delimiter string, prefixMap map[string]time.Time) int {
	aName := a.Name.Resolve()
	bName := b.Name.Resolve()

	if aName == bName {
		return 0
	}

	if delimiter == "" {
		delimiter = "-"
	}

	aTokens := strings.Split(aName, delimiter)
	bTokens := strings.Split(bName, delimiter)

	minLen := min(len(aTokens), len(bTokens))
	k := 0
	for k < minLen && aTokens[k] == bTokens[k] {
		k++
	}

	if k == len(aTokens) && k == len(bTokens) {
		return CompareChronological(a, b, registry, parentToChildren, 0)
	}

	if k == len(aTokens) {
		return -1
	}
	if k == len(bTokens) {
		return 1
	}

	aSubPrefix := strings.Join(aTokens[:k+1], delimiter)
	bSubPrefix := strings.Join(bTokens[:k+1], delimiter)

	getPrefixTime := func(prefix string) (time.Time, bool) {
		if prefixMap != nil {
			t, ok := prefixMap[prefix]
			return t, ok
		}
		return findPrefixOldestTime(prefix, registry, parentToChildren, delimiter)
	}

	aTime, aOk := getPrefixTime(aSubPrefix)
	bTime, bOk := getPrefixTime(bSubPrefix)

	switch {
	case aOk && bOk:
		if cmp := aTime.Compare(bTime); cmp != 0 {
			return cmp
		}
	case aOk:
		return -1
	case bOk:
		return 1
	}

	return strings.Compare(aTokens[k], bTokens[k])
}

// findPrefixOldestTime finds the oldest timestamp among all descendant timelines matching the given name prefix.
// It serves as a fallback search mechanism when pre-calculated prefix maps are not available.
func findPrefixOldestTime(prefix string, registry *TimelineRegistry, parentToChildren map[*TimelinePath][]*TimelinePath, delimiter string) (time.Time, bool) {
	var oldest time.Time
	found := false

	var visit func(p *TimelinePath)
	visit = func(p *TimelinePath) {
		if p == nil || p.Name == nil {
			return
		}
		pName := p.Name.Resolve()
		// Ensures pName belongs to the hierarchical group defined by prefix.
		// To prevent false positive substring matches (e.g., prefix "aaaa-bb" matching "aaaa-bbbb"),
		// we verify that pName either matches prefix exactly or continues immediately with the delimiter.
		if strings.HasPrefix(pName, prefix) && (len(pName) == len(prefix) || strings.HasPrefix(pName[len(prefix):], delimiter)) {
			if t, ok := FindOldestTime(p, registry, parentToChildren, 0, 0); ok {
				if !found || t.Before(oldest) {
					oldest = t
					found = true
				}
			}
		}
		for _, child := range parentToChildren[p] {
			visit(child)
		}
	}

	// Starts DFS traversal from all root paths (where the parent pointer is nil).
	for _, root := range parentToChildren[nil] {
		visit(root)
	}
	return oldest, found
}

// FindOldestTime finds the oldest time in a timeline path and its descendants up to maxDepth.
func FindOldestTime(path *TimelinePath, registry *TimelineRegistry, parentToChildren map[*TimelinePath][]*TimelinePath, currentDepth int, maxDepth int32) (time.Time, bool) {
	var oldest time.Time
	found := false

	updateOldest := func(t time.Time) {
		if !found || t.Before(oldest) {
			oldest = t
			found = true
		}
	}

	if builder, exists := registry.GetBuilderIfExists(path); exists {
		builder.mu.Lock()
		revisions := slices.Clone(builder.revisions)
		events := slices.Clone(builder.events)
		builder.mu.Unlock()

		for _, rev := range revisions {
			if t := rev.GetChangedTime(); t != nil {
				updateOldest(t.AsTime())
			}
		}

		for _, ev := range events {
			if logEntry := registry.GetLog(ev.GetLogId()); logEntry != nil {
				if t := logEntry.GetTs(); t != nil {
					updateOldest(t.AsTime())
				}
			}
		}
	}

	if maxDepth <= 0 || currentDepth < int(maxDepth) {
		for _, child := range parentToChildren[path] {
			if childTime, childOk := FindOldestTime(child, registry, parentToChildren, currentDepth+1, maxDepth); childOk {
				updateOldest(childTime)
			}
		}
	}

	return oldest, found
}

// sortTimelinePaths sorts the timeline paths hierarchically using pre-order traversal,
// taking timeline type sort priority and sorting policy configurations into account.
func sortTimelinePaths(paths iter.Seq[*TimelinePath], registry *TimelineRegistry) []*TimelinePath {
	// 1. Build parent-to-children adjacency list
	parentToChildren := make(map[*TimelinePath][]*TimelinePath)
	var roots []*TimelinePath
	var allPaths []*TimelinePath

	for path := range paths {
		allPaths = append(allPaths, path)
		if path.Parent == nil {
			roots = append(roots, path)
		} else {
			parentToChildren[path.Parent] = append(parentToChildren[path.Parent], path)
		}
	}

	prioritizedNamesMap := make(map[uint32]map[string]int)
	groupedSortMap := make(map[uint32]map[string]time.Time)

	for _, path := range allPaths {
		if path == nil || path.Name == nil {
			continue
		}
		if path.Type != nil && path.Type.Id != nil {
			typeID := *path.Type.Id
			if _, ok := prioritizedNamesMap[typeID]; !ok {
				alphabeticalSortPolicy := path.Type.GetAlphabeticalPolicy()
				if alphabeticalSortPolicy != nil {
					m := make(map[string]int)
					for i, name := range alphabeticalSortPolicy.PrioritizedNames {
						m[name] = i
					}
					prioritizedNamesMap[typeID] = m
				}
			}
			if policy, ok := path.Type.SortPolicyConfig.(*pb.TimelineType_GroupedChronologicalPolicy); ok && policy.GroupedChronologicalPolicy != nil {
				if _, ok := groupedSortMap[typeID]; !ok {
					groupedSortMap[typeID] = make(map[string]time.Time)
				}
				if t, ok := FindOldestTime(path, registry, parentToChildren, 0, 0); ok {
					delimiter := policy.GroupedChronologicalPolicy.GetDelimiter()
					if delimiter == "" {
						delimiter = "-"
					}
					tokens := strings.Split(path.Name.Resolve(), delimiter)
					curMap := groupedSortMap[typeID]
					for i := range tokens {
						prefix := strings.Join(tokens[:i+1], delimiter)
						if existing, ok := curMap[prefix]; !ok || t.Before(existing) {
							curMap[prefix] = t
						}
					}
				}
			}
		}
	}

	comparePaths := func(a, b *TimelinePath) int {
		if a.Type == nil || a.Type.Id == nil {
			panic("timeline path a has nil Type or ID")
		}
		if b.Type == nil || b.Type.Id == nil {
			panic("timeline path b has nil Type or ID")
		}

		aTypeID := *a.Type.Id
		bTypeID := *b.Type.Id

		if aTypeID != bTypeID {
			aPriority := a.Type.GetSortPriority()
			bPriority := b.Type.GetSortPriority()
			if diff := int(aPriority - bPriority); diff != 0 {
				return diff
			}
			aLabel := a.Type.GetLabel()
			bLabel := b.Type.GetLabel()
			if cmpLabel := strings.Compare(aLabel, bLabel); cmpLabel != 0 {
				return cmpLabel
			}
			panic("duplicated timeline path found for same sort priority and label but different ID")
		}

		switch policy := a.Type.SortPolicyConfig.(type) {
		case *pb.TimelineType_AlphabeticalPolicy:
			if policy.AlphabeticalPolicy != nil {
				return CompareAlphabetical(a, b, prioritizedNamesMap[aTypeID])
			}
		case *pb.TimelineType_ChronologicalPolicy:
			if policy.ChronologicalPolicy != nil {
				return CompareChronological(a, b, registry, parentToChildren, policy.ChronologicalPolicy.GetChronologicalSearchDepth())
			}
		case *pb.TimelineType_GroupedChronologicalPolicy:
			if policy.GroupedChronologicalPolicy != nil {
				delimiter := policy.GroupedChronologicalPolicy.GetDelimiter()
				if delimiter == "" {
					delimiter = "-"
				}
				return CompareGroupedChronologicalWithMap(a, b, registry, parentToChildren, delimiter, groupedSortMap[aTypeID])
			}
		}
		panic("SortPolicy is not specified for timeline path")
	}

	var sortedPaths []*TimelinePath
	var traverse func(p *TimelinePath)

	traverse = func(p *TimelinePath) {
		sortedPaths = append(sortedPaths, p)
		children := parentToChildren[p]
		slices.SortFunc(children, comparePaths)
		for _, child := range children {
			traverse(child)
		}
	}

	slices.SortFunc(roots, comparePaths)
	for _, r := range roots {
		traverse(r)
	}

	return sortedPaths
}
