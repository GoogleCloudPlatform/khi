// Copyright 2025 Google LLC
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

package generator

import (
	"regexp"
	"strings"
)

var regexToRemoveInGithubAnchorHash = regexp.MustCompile(`[^a-z0-9\s\-]+`)
var regexToHyphenInGithubAnchorHash = regexp.MustCompile(`\s+`)

// ToGithubAnchorHash convert the given text of header to the hash of anchor link.
func ToGithubAnchorHash(text string) string {
	return regexToRemoveInGithubAnchorHash.ReplaceAllString(regexToHyphenInGithubAnchorHash.ReplaceAllString(strings.ToLower(strings.TrimSpace(text)), "-"), "")
}
