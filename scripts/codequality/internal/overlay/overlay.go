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

package overlay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type overlayJSON struct {
	Replace map[string]string `json:"Replace"`
}

// WriteOverlayJSON marshals replace into a Go -overlay JSON file named fileName inside dir.
func WriteOverlayJSON(dir string, fileName string, replace map[string]string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create overlay directory: %w", err)
	}
	data, err := json.MarshalIndent(overlayJSON{Replace: replace}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal overlay json: %w", err)
	}
	outPath := filepath.Join(dir, fileName)
	if err := os.WriteFile(outPath, data, 0o600); err != nil {
		return "", fmt.Errorf("failed to write overlay json: %w", err)
	}
	return outPath, nil
}
