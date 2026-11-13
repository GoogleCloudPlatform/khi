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

package structured

import (
	"fmt"
	"log/slog"
	"sync"
)

// MergeStrategyMerge merges array elements by matching on a configured merge key.
const MergeStrategyMerge MergeArrayStrategy = "merge"

// MergeStrategyReplace replaces the previous array with the patch array.
const MergeStrategyReplace MergeArrayStrategy = "replace"

var warnShownPath = sync.Map{}

// MergeArrayStrategy defines how arrays at a given field path are merged during strategic merge patching.
type MergeArrayStrategy string

// MergeConfigResolver resolves array merge strategies and merge keys by field path, falling back to an optional parent resolver.
type MergeConfigResolver struct {
	// Parent specifies an optional fallback resolver consulted when a field path is not defined locally.
	Parent *MergeConfigResolver
	// MergeStrategies maps dot-separated field paths to their array merge strategies.
	MergeStrategies map[string]MergeArrayStrategy
	// MergeKeys maps dot-separated field paths to the key field name used when merging arrays of maps.
	MergeKeys map[string]string
}

// GetMergeArrayStrategy returns the merge strategy for fieldPath, defaulting to MergeStrategyReplace if undefined.
func (r *MergeConfigResolver) GetMergeArrayStrategy(fieldPath string) MergeArrayStrategy {
	if strategy, found := r.MergeStrategies[fieldPath]; found {
		return strategy
	} else {
		if r.Parent != nil {
			return r.Parent.GetMergeArrayStrategy(fieldPath)
		}
		_, found := warnShownPath.LoadOrStore(fieldPath, struct{}{})
		if !found {
			slog.Debug(fmt.Sprintf("Merge strategy for %s is not defined. Use replace strategy.", fieldPath))
		}
		return MergeStrategyReplace
	}
}

// GetMergeKey returns the merge key field name for fieldPath, or an error if none is configured.
func (r *MergeConfigResolver) GetMergeKey(fieldPath string) (string, error) {
	if key, found := r.MergeKeys[fieldPath]; found {
		return key, nil
	} else {
		if r.Parent != nil {
			return r.Parent.GetMergeKey(fieldPath)
		}
		return "", fmt.Errorf("merge key for %s was not found", fieldPath)
	}
}
