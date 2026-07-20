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

package logutil

// ParserRule defines a matching rule for a specific context of type T.
type ParserRule[T any] struct {
	// Match returns true if the rule applies to the given context.
	Match func(ctx T) bool
	// Parser is the structured log parser to execute when Match returns true.
	Parser StructuredLogParser
}

// SelectorLogParser selects and executes a StructuredLogParser based on a context object of type T.
type SelectorLogParser[T any] struct {
	rules         []ParserRule[T]
	defaultParser StructuredLogParser
}

// NewSelectorLogParser creates a new SelectorLogParser instance.
func NewSelectorLogParser[T any](defaultParser StructuredLogParser, rules ...ParserRule[T]) *SelectorLogParser[T] {
	return &SelectorLogParser[T]{
		rules:         rules,
		defaultParser: defaultParser,
	}
}

// TryParse evaluates rules against the provided context T and executes the matching StructuredLogParser.
// If no rule matches or the matching parser returns nil, it falls back to defaultParser.
func (s *SelectorLogParser[T]) TryParse(ctx T, message string) *ParseStructuredLogResult {
	for _, rule := range s.rules {
		if rule.Match != nil && rule.Match(ctx) && rule.Parser != nil {
			if result := rule.Parser.TryParse(message); result != nil {
				return result
			}
		}
	}
	if s.defaultParser != nil {
		return s.defaultParser.TryParse(message)
	}
	return nil
}
