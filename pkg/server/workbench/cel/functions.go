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

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
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
func MatchTimelineRevisionBodyField(t *TimelineData, pathKey string, patterns []string, pool *khifilev6model.InternPool) bool {
	if t == nil || len(patterns) == 0 || pool == nil {
		return false
	}

	for _, r := range t.Revisions {
		if r.ResourceBodyStructID == 0 {
			continue
		}
		s := pool.ResolveStructFromID(r.ResourceBodyStructID)
		if s == nil {
			continue
		}
		node, err := khifilev6model.FromInternedStruct(s, pool)
		if err != nil {
			continue
		}

		if matchNodeField(node, pathKey, patterns) {
			return true
		}
	}
	return false
}

// MatchLogField checks if a log body or field matches the given pattern(s).
func MatchLogField(l *LogData, pathKey string, patterns []string, pool *khifilev6model.InternPool) bool {
	if l == nil || len(patterns) == 0 || l.BodyStructID == 0 || pool == nil {
		return false
	}

	s := pool.ResolveStructFromID(l.BodyStructID)
	if s == nil {
		return false
	}
	node, err := khifilev6model.FromInternedStruct(s, pool)
	if err != nil {
		return false
	}

	return matchNodeField(node, pathKey, patterns)
}

func matchNodeField(node structured.Node, pathKey string, patterns []string) bool {
	if node == nil {
		return false
	}

	if pathKey == "*" {
		yamlBytes, err := (&structured.YAMLNodeSerializer{}).Serialize(node)
		if err != nil {
			return false
		}
		bodyYAML := string(yamlBytes)
		for _, pattern := range patterns {
			re, err := regexCompile(pattern)
			if err != nil {
				continue
			}
			if re.MatchString(bodyYAML) {
				return true
			}
		}
		return false
	}

	// Use NodeReader for path traversal without full YAML serialization
	reader := structured.NewNodeReader(node)
	targetReader, err := reader.GetReader(pathKey)
	if err != nil {
		return false
	}

	// If target is a scalar node, match its scalar string value
	if targetReader.Node.Type() == structured.ScalarNodeType {
		val, err := targetReader.Node.NodeScalarValue()
		if err != nil || val == nil {
			return false
		}
		valStr := fmt.Sprintf("%v", val)
		for _, pattern := range patterns {
			re, err := regexCompile(pattern)
			if err != nil {
				continue
			}
			if re.MatchString(valStr) {
				return true
			}
		}
	} else {
		// If target is a sub-object/array, serialize only that sub-tree to YAML
		subYAML, err := (&structured.YAMLNodeSerializer{}).Serialize(targetReader.Node)
		if err == nil {
			subStr := string(subYAML)
			for _, pattern := range patterns {
				re, err := regexCompile(pattern)
				if err != nil {
					continue
				}
				if re.MatchString(subStr) {
					return true
				}
			}
		}
	}
	return false
}
