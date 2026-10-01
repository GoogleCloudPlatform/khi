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

package main

import (
	"os"
	"testing"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/generated"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
)

func TestGeneratedReferencesUpToDate(t *testing.T) {
	taskServer, err := coreinspection.NewServer(nil)
	if err != nil {
		t.Fatalf("failed to create inspection server: %v", err)
	}
	if err := generated.RegisterAllInspectionTasks(taskServer); err != nil {
		t.Fatalf("failed to register inspection tasks: %v", err)
	}

	styleChunk := style.GenerateChunkWithoutIconAtlas()
	expectedTimeline := cel.GenerateTimelineReference(styleChunk)
	expectedLog := cel.GenerateLogReference(styleChunk)

	testCases := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "timeline reference",
			filePath: "../../plugins/khi/skills/khi-investigation/references/cel-timeline.md",
			expected: expectedTimeline,
		},
		{
			name:     "log reference",
			filePath: "../../plugins/khi/skills/khi-investigation/references/cel-log.md",
			expected: expectedLog,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.filePath)
			if err != nil {
				t.Fatalf("failed to read %s (run 'make generate-agent-plugin-references'): %v", tc.filePath, err)
			}
			actual := string(data)
			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Errorf("%s is out of date (-expected +actual). Run 'make generate-agent-plugin-references' to update:\n%s", tc.filePath, diff)
			}
		})
	}
}
