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

package server

// Codes to generate index.html dynamatically

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/server/index"
)

var IndexReplacePlaceholder = `<!--INJECT GENERATED CODE HERE FROM BACKEND-->`

// replaceLocalDevServerOnlyTag removed tags only used in the local dev environment.
func replaceLocalDevServerOnlyTag(source string) string {
	return strings.ReplaceAll(source, `<base href="/" />`, "") // The base tag must be supplied but it will be injected from backend usually. It needs to be added by default on the static index.html because KHI can't inject the tag to the static index.html served by angular dev server.
}

// replaceDynamicPartOfIndex rewrites the given original HTML string with injecting several tags to be injected dynamatically.
func replaceDynamicPartOfIndex(originalIndexHTML string) (string, error) {
	if !strings.Contains(originalIndexHTML, IndexReplacePlaceholder) {
		return "", fmt.Errorf("inject taregt string was not found")
	}

	generatedTags := index.GenerateTags()

	return strings.Replace(replaceLocalDevServerOnlyTag(originalIndexHTML), IndexReplacePlaceholder, strings.Join(generatedTags, "\n"), 1), nil
}
