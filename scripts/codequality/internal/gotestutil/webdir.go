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

package gotestutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveWebDir resolves the frontend web directory, returning webDir when it contains package.json or falling back to repoRoot/web.
func ResolveWebDir(webDir string) (string, error) {
	if webDir != "" {
		if _, err := os.Stat(filepath.Join(webDir, "package.json")); err == nil {
			return webDir, nil
		}
	}
	repoRoot, err := ResolveRepoRoot("")
	if err != nil {
		return "", fmt.Errorf("could not locate web/package.json for TypeScript parser: %w", err)
	}
	return filepath.Join(repoRoot, "web"), nil
}
