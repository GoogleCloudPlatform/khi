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
	"fmt"
	"os"
	"path/filepath"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/generated"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
)

const (
	timelineRefPath = "plugins/khi/skills/khi-investigation/references/cel-timeline.md"
	logRefPath      = "plugins/khi/skills/khi-investigation/references/cel-log.md"
)

func main() {
	taskServer, err := coreinspection.NewServer(nil)
	if err != nil {
		panic(err)
	}
	if err := generated.RegisterAllInspectionTasks(taskServer); err != nil {
		panic(err)
	}

	styleChunk := style.GenerateChunkWithoutIconAtlas()
	timelineDoc := cel.GenerateTimelineReference(styleChunk)
	logDoc := cel.GenerateLogReference(styleChunk)

	mustWriteFile(timelineRefPath, timelineDoc)
	mustWriteFile(logRefPath, logDoc)
	fmt.Printf("Generated CEL reference documents:\n  - %s\n  - %s\n", timelineRefPath, logRefPath)
}

func mustWriteFile(filePath string, data string) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filePath, []byte(data), 0644); err != nil {
		panic(err)
	}
}
