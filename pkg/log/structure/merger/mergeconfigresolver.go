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

package merger

import (
	"fmt"
	"log/slog"
	"sync"
)

type MergeArrayStrategy string

const MergeStrategyMerge MergeArrayStrategy = "merge"
const MergeStrategyReplace MergeArrayStrategy = "replace"

var warnShownPath = sync.Map{}

type MergeConfigResolver struct {
	Parent          *MergeConfigResolver
	MergeStrategies map[string]MergeArrayStrategy
	MergeKeys       map[string]string
}

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
