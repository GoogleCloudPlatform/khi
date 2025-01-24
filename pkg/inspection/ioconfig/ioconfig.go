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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var IOConfigTaskName = task.KHISystemPrefix + "inspection/ioconfig"

type IOConfig struct {
	// The project root folder
	ApplicationRoot string
	// The folder to save khi files
	DataDestination string
	// TemporaryFolder working folder
	TemporaryFolder string
}

var ProductionIOConfig = task.NewCachedProcessor(IOConfigTaskName, []string{}, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
	dataDestinationFolder := "./data"
	if parameters.Common.DataDestinationFolder != nil {
		dataDestinationFolder = *parameters.Common.DataDestinationFolder
	}
	temporaryFolder := "/tmp"
	if parameters.Common.TemporaryFolder != nil {
		temporaryFolder = *parameters.Common.TemporaryFolder
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(dataDestinationFolder) {
		dataDestinationFolder = filepath.Join(dir, dataDestinationFolder)
	}
	return &IOConfig{
		ApplicationRoot: dir,
		DataDestination: dataDestinationFolder,
		TemporaryFolder: temporaryFolder,
	}, nil
})

var TestIOConfig = task.NewCachedProcessor(IOConfigTaskName, []string{}, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".root")); err == nil {
			break
		}
		pathsSegments := strings.Split(dir, "/")
		dir = "/" + filepath.Join(pathsSegments[:len(pathsSegments)-1]...)
	}
	return &IOConfig{
		ApplicationRoot: dir + "/",
		DataDestination: "/tmp/",
		TemporaryFolder: "/tmp/",
	}, nil
})

func GetIOConfigFromTaskVariable(v *task.VariableSet) (*IOConfig, error) {
	return task.GetTypedVariableFromTaskVariable[*IOConfig](v, IOConfigTaskName, nil)
}
