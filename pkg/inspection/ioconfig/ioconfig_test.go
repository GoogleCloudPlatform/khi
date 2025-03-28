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

package ioconfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	inspection_task_test "github.com/GoogleCloudPlatform/khi/pkg/inspection/test"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestTestIOConfigCanFindTheRoot(t *testing.T) {
	ctx := inspection_task_test.WithDefaultTestInspectionTaskContext(context.Background())

	ioConfig, _, err := inspection_task_test.RunInspectionTask(ctx, TestIOConfig, inspection_task_interface.TaskModeRun, map[string]any{})
	if err != nil {
		t.Errorf("unxepected error %v", err)
	}
	stat, err := os.Stat(ioConfig.ApplicationRoot)
	if err != nil {
		t.Errorf("unxepected error %v", err)
	}
	if !stat.IsDir() {
		t.Errorf("the result application root must be a directory")
	}
}

func TestProductionIOConfigConvertPathToAbs(t *testing.T) {
	ctx := inspection_task_test.WithDefaultTestInspectionTaskContext(context.Background())

	ioConfig, _, err := inspection_task_test.RunInspectionTask(ctx, ProductionIOConfig, inspection_task_interface.TaskModeRun, map[string]any{})
	if err != nil {
		t.Errorf("unxepected error %v", err)
	}
	if !filepath.IsAbs(ioConfig.ApplicationRoot) {
		t.Errorf("the given application folder must be abs path")
	}
	if !filepath.IsAbs(ioConfig.DataDestination) {
		t.Errorf("the given data destination folder must be abs path")
	}
	stat, err := os.Stat(ioConfig.ApplicationRoot)
	if err != nil {
		t.Errorf("unxepected error %v", err)
	}
	if !stat.IsDir() {
		t.Errorf("the result application root must be a directory")
	}
}
