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

func combinePatterns(patterns []string) string {
	if len(patterns) == 0 {
		return ""
	}
	if len(patterns) == 1 {
		return patterns[0]
	}
	return "(?:" + strings.Join(patterns, ")|(?:") + ")"
}

// resolveStructYAML resolves the YAML string representation for a given struct ID, using the cache if available or serializing on-demand from the intern pool.
func resolveStructYAML(structID uint32, pool khifilev6model.ReadonlyPool, structYAMLs map[uint32]string) (string, bool) {
	if structYAMLs != nil {
		if yaml, ok := structYAMLs[structID]; ok {
			return yaml, true
		}
	}
	if pool == nil {
		return "", false
	}
	s := pool.ResolveStructFromID(structID)
	if s == nil {
		return "", false
	}
	node, err := khifilev6model.FromInternedStruct(s, pool)
	if err != nil {
		return "", false
	}
	yamlBytes, err := (&structured.YAMLNodeSerializer{}).Serialize(node)
	if err != nil {
		return "", false
	}
	return string(yamlBytes), true
}

// resolveStructNode resolves the structured.Node from an interned struct ID.
func resolveStructNode(structID uint32, pool khifilev6model.ReadonlyPool) (structured.Node, bool) {
	if pool == nil {
		return nil, false
	}
	s := pool.ResolveStructFromID(structID)
	if s == nil {
		return nil, false
	}
	node, err := khifilev6model.FromInternedStruct(s, pool)
	if err != nil {
		return nil, false
	}
	return node, true
}

// MatchTimelinePath checks if any key in timeline's hierarchy path matches the given pattern(s) case-insensitively.
func MatchTimelinePath(t *TimelineData, key string, patterns []string, tlMap map[uint32]*TimelineData) bool {
	if t == nil || len(patterns) == 0 {
		return false
	}

	combinedPattern := combinePatterns(patterns)
	re, err := regexCompile(combinedPattern)
	if err != nil {
		return false
	}

	targetKeyLower := strings.ToLower(key)
	visited := make(map[uint32]struct{})
	curr := t
	for curr != nil {
		if _, seen := visited[curr.ID]; seen {
			break
		}
		visited[curr.ID] = struct{}{}

		typeKey := strings.ToLower(curr.TimelineType)
		if key == "*" || (typeKey != "" && typeKey == targetKeyLower) {
			if re.MatchString(curr.Name) {
				return true
			}
		}
		if curr.ParentID == 0 || tlMap == nil {
			break
		}
		curr = tlMap[curr.ParentID]
	}
	return false
}

// MatchTimelineRevisionBodyField checks if any revision in timeline matches the pathKey and pattern(s).
func MatchTimelineRevisionBodyField(t *TimelineData, pathKey string, patterns []string, pool khifilev6model.ReadonlyPool) bool {
	if t == nil || len(patterns) == 0 || pool == nil {
		return false
	}

	combinedPattern := combinePatterns(patterns)

	for _, r := range t.Revisions {
		if r.ResourceBodyStructID == 0 {
			continue
		}
		node, ok := resolveStructNode(r.ResourceBodyStructID, pool)
		if !ok {
			continue
		}
		if matchNodeField(node, pathKey, combinedPattern) {
			return true
		}
	}
	return false
}

// MatchLogField checks if a log body or field matches the given regular expression pattern(s).
func MatchLogField(
	l *LogData,
	pathKey string,
	patterns []string,
	pool khifilev6model.ReadonlyPool,
	trigramIndex *TrigramIndex,
	structYAMLs map[uint32]string,
) (bool, error) {
	if l == nil || len(patterns) == 0 || l.BodyStructID == 0 {
		return false, nil
	}

	combinedPattern := combinePatterns(patterns)

	// Candidate pre-filtering with TrigramIndex:
	// If trigram index exists and neither log ID nor struct ID is in candidate bitmap, prune early.
	if trigramIndex != nil {
		candidateBitmap := trigramIndex.FindCandidateLogsWithField(pathKey, combinedPattern)
		if candidateBitmap != nil {
			matchLog := l.ID != 0 && candidateBitmap.Contains(l.ID)
			matchStruct := l.BodyStructID != 0 && candidateBitmap.Contains(l.BodyStructID)
			if !matchLog && !matchStruct {
				return false, nil
			}
		}
	}

	// Wildcard search (pathKey == "*"):
	if pathKey == "*" {
		yaml, ok := resolveStructYAML(l.BodyStructID, pool, structYAMLs)
		if !ok {
			return false, nil
		}
		re, err := regexCompile(combinedPattern)
		if err != nil {
			return false, nil
		}
		return re.MatchString(yaml), nil
	}

	// Field-specific search (pathKey != "*"):
	node, ok := resolveStructNode(l.BodyStructID, pool)
	if !ok {
		return false, nil
	}
	return matchNodeField(node, pathKey, combinedPattern), nil
}

func matchNodeField(node structured.Node, pathKey string, pattern string) bool {
	if node == nil {
		return false
	}

	re, err := regexCompile(pattern)
	if err != nil {
		return false
	}

	if pathKey == "*" {
		yamlBytes, err := (&structured.YAMLNodeSerializer{}).Serialize(node)
		if err != nil {
			return false
		}
		return re.MatchString(string(yamlBytes))
	}

	reader := structured.NewNodeReader(node)
	targetReader, err := reader.GetReader(pathKey)
	if err != nil {
		return false
	}

	if targetReader.Node.Type() == structured.ScalarNodeType {
		val, err := targetReader.Node.NodeScalarValue()
		if err != nil || val == nil {
			return false
		}
		return re.MatchString(fmt.Sprintf("%v", val))
	}

	subYAML, err := (&structured.YAMLNodeSerializer{}).Serialize(targetReader.Node)
	if err != nil {
		return false
	}
	return re.MatchString(string(subYAML))
}
