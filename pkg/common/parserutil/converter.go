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

package parserutil

import (
	"strconv"
	"strings"
)

// ANSIBeginSequences is a list of prefixes of ANSI escape sequences.
var ANSIBeginSequences []string = []string{
	"\\x1b[",
	"\\033[",
	"\\u001B[",
}

// SpecialSequenceConverter converts specific sequences from string to the other or removes them. (e.g ASCII escape characters, ANSI color characters)
type SpecialSequenceConverter interface {
	// Convert receives the original string and returns the converted string.
	Convert(s string) string
}

// ConvertSpecialSequences returns the stripped string with applying provided strippers to the original string.
func ConvertSpecialSequences(original string, converter ...SpecialSequenceConverter) string {
	for _, s := range converter {
		original = s.Convert(original)
	}
	return original
}

// SequenceConverter replaces specific sequences to a string.
type SequenceConverter struct {
	// To is the replace result of any items in the From field.
	To string
	// From is the list of string to be replaced with the To field.
	From []string
}

func (c *SequenceConverter) Convert(s string) string {
	for _, target := range c.From {
		s = strings.ReplaceAll(s, target, c.To)
	}
	return s
}

var _ SpecialSequenceConverter = (*SequenceConverter)(nil)

// ANSIEscapeSequenceStripper removes ANSI escape sequences.
type ANSIEscapeSequenceStripper struct {
}

// Convert implements SpecialSequenceStripper.
func (a *ANSIEscapeSequenceStripper) Convert(s string) string {
	builder := strings.Builder{}
	for i := 0; i < len(s); i++ {
		ansiFound := false
		nextFound := len(s)
		for _, beginSequence := range ANSIBeginSequences {
			nextFoundForSequence := strings.Index(s[i:], beginSequence)
			if nextFoundForSequence != -1 && nextFoundForSequence < nextFound {
				nextFound = nextFoundForSequence
			}
		}
		if nextFound != len(s) {
			ansiFound = true
			foundSuffix := false
			builder.WriteString(s[i : i+nextFound])
			i += nextFound
			for j := i; j < len(s); j++ {
				if s[j] == 'm' {
					i = j
					foundSuffix = true
					break
				}
			}
			if !foundSuffix { // This ANSI sequence is not complete. Write it as is.
				builder.WriteString(s[i:])
				break
			}
		}
		if !ansiFound {
			builder.WriteString(s[i:])
			break
		}
	}
	return builder.String()
}

var _ SpecialSequenceConverter = (*ANSIEscapeSequenceStripper)(nil)

// UnicodeUnquoteConverter replaces escaped unicode characters like `\\xe2\\x80\\xa6` into the corresponded unicode string.
type UnicodeUnquoteConverter struct{}

// Convert implements SpecialSequenceConverter.
func (u *UnicodeUnquoteConverter) Convert(s string) string {
	converted, err := strconv.Unquote(`"` + s + `"`)
	if err != nil {
		return s
	}
	return converted
}

var _ SpecialSequenceConverter = (*UnicodeUnquoteConverter)(nil)
