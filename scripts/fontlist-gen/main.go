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

package main

import (
	"encoding/json"
	"os"
	"sort"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/generated"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

const usedIconFilesLocation = "./scripts/msdf-generator/zzz_generated_used_icons.json"

type usedIconSetting struct {
	Icons []string `json:"icons"`
}

func main() {
	// Initialize the task server to run all the inspection task registrations
	// which in turn register the custom revision states and their icons.
	taskServer, err := coreinspection.NewServer(nil)
	if err != nil {
		panic(err)
	}
	err = generated.RegisterAllInspectionTasks(taskServer)
	if err != nil {
		panic(err)
	}

	// Generate icons.json storing all the icons used in revision states to generate the icon font atlas.
	var icons = map[string]struct{}{
		// TODO: Remove fofllowing icons. These are registered temporary for the mock data.
		"step_over": {},
		"skull":     {},
	}
	chunk := style.GenerateChunkWithoutIconAtlas()
	for _, revisionState := range chunk.RevisionStates {
		icon := revisionState.GetIcon()
		if icon != "" {
			icons[icon] = struct{}{}
		}
	}

	iconSetting := usedIconSetting{
		Icons: []string{},
	}
	for icon := range icons {
		iconSetting.Icons = append(iconSetting.Icons, icon)
	}
	sort.Strings(iconSetting.Icons)
	iconsJson, err := json.Marshal(iconSetting)
	if err != nil {
		panic(err)
	}
	mustWriteFile(usedIconFilesLocation, string(iconsJson))
}

// mustWriteFile writes the given data to a file, panicking if an error occurs.
func mustWriteFile(filePath string, data string) {
	if err := os.WriteFile(filePath, []byte(data), 0644); err != nil {
		panic(err)
	}
}
