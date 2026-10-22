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

package md

import "strings"

// Code encloses s in markdown backticks, handling embedded backticks per CommonMark.
func Code(s string) string {
	if !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	maxRun := 0
	currentRun := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '`' {
			currentRun++
			if currentRun > maxRun {
				maxRun = currentRun
			}
		} else {
			currentRun = 0
		}
	}
	delim := strings.Repeat("`", maxRun+1)
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		return delim + " " + s + " " + delim
	}
	return delim + s + delim
}

// Cell escapes pipes and converts newlines to spaces for markdown table cells.
func Cell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// Fence wraps body in a markdown fenced code block with the given language.
func Fence(lang, body string) string {
	maxRun := 0
	currentRun := 0
	for i := 0; i < len(body); i++ {
		if body[i] == '`' {
			currentRun++
			if currentRun > maxRun {
				maxRun = currentRun
			}
		} else {
			currentRun = 0
		}
	}
	fenceLen := max(3, maxRun+1)
	fence := strings.Repeat("`", fenceLen)
	trimmedBody := strings.TrimSuffix(strings.TrimSuffix(body, "\n"), "\r")
	return fence + lang + "\n" + trimmedBody + "\n" + fence
}
