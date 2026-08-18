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
	"regexp"
	"strings"
	"sync"
)

var (
	regexCache   sync.Map
	regexCompile = func(pattern string) (*regexp.Regexp, error) {
		if val, ok := regexCache.Load(pattern); ok {
			return val.(*regexp.Regexp), nil
		}
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, err
		}
		regexCache.Store(pattern, re)
		return re, nil
	}
)

// MatchTimelinePath checks if any key in timeline's path matches the given pattern(s) case-insensitively.
func MatchTimelinePath(t *TimelineData, key string, patterns []string) bool {
	if t == nil || t.Path == nil || len(patterns) == 0 {
		return false
	}

	for _, pattern := range patterns {
		re, err := regexCompile(pattern)
		if err != nil {
			continue
		}

		if key == "*" {
			for _, val := range t.Path {
				if re.MatchString(val) {
					return true
				}
			}
		} else {
			if val, ok := t.Path[strings.ToLower(key)]; ok {
				if re.MatchString(val) {
					return true
				}
			}
		}
	}
	return false
}

// MatchTimelineRevisionBodyField checks if any revision in timeline matches the pathKey and pattern(s).
func MatchTimelineRevisionBodyField(t *TimelineData, pathKey string, patterns []string) bool {
	if t == nil || len(patterns) == 0 {
		return false
	}

	for _, r := range t.Revisions {
		for _, pattern := range patterns {
			re, err := regexCompile(pattern)
			if err != nil {
				continue
			}

			if pathKey == "*" {
				if re.MatchString(r.BodyYAML) {
					return true
				}
			} else {
				val, ok := resolveMapPath(r.Body, pathKey)
				if ok && re.MatchString(fmt.Sprintf("%v", val)) {
					return true
				}
			}
		}
	}
	return false
}

// MatchLogField checks if a log body or field matches the given pattern(s).
func MatchLogField(l *LogData, pathKey string, patterns []string) bool {
	if l == nil || len(patterns) == 0 {
		return false
	}

	for _, pattern := range patterns {
		re, err := regexCompile(pattern)
		if err != nil {
			continue
		}

		if pathKey == "*" {
			if re.MatchString(l.BodyYAML) {
				return true
			}
		} else {
			val, ok := resolveMapPath(l.Body, pathKey)
			if ok && re.MatchString(fmt.Sprintf("%v", val)) {
				return true
			}
		}
	}
	return false
}

func resolveMapPath(m map[string]any, pathKey string) (any, bool) {
	if m == nil {
		return nil, false
	}
	parts := strings.Split(pathKey, ".")
	var current any = m

	for _, part := range parts {
		currMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := currMap[part]
		if !exists {
			return nil, false
		}
		current = val
	}

	if current == nil {
		return nil, false
	}
	if _, isMap := current.(map[string]any); isMap {
		return nil, false
	}
	if _, isSlice := current.([]any); isSlice {
		return nil, false
	}
	return current, true
}
