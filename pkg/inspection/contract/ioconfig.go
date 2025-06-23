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

package inspectioncontract

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
)

// IOConfig holds configuration for file input/output operations.
type IOConfig struct {
	// ApplicationRoot is the project root folder
	ApplicationRoot string
	// DataDestination is the folder to save khi files
	DataDestination string
	// TemporaryFolder is the working folder for temporary files
	TemporaryFolder string
}

// NewIOConfigFromParameter creates an IOConfig from common parameters for production use.
// It resolves data destination and temporary folders from parameters or uses defaults.
func NewIOConfigFromParameter(commonParameter *parameters.CommonParameters) (*IOConfig, error) {
	dataDestinationFolder := "./data"
	if commonParameter.DataDestinationFolder != nil {
		dataDestinationFolder = *commonParameter.DataDestinationFolder
	}
	temporaryFolder := "/tmp"
	if commonParameter.TemporaryFolder != nil {
		temporaryFolder = *commonParameter.TemporaryFolder
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(dataDestinationFolder) {
		dataDestinationFolder = filepath.Join(dir, dataDestinationFolder)
	}
	if !filepath.IsAbs(temporaryFolder) {
		temporaryFolder = filepath.Join(dir, temporaryFolder)
	}
	return &IOConfig{
		ApplicationRoot: dir,
		DataDestination: dataDestinationFolder,
		TemporaryFolder: temporaryFolder,
	}, nil
}

// NewIOConfigForTest creates an IOConfig for testing by finding the project root
// using the .root marker file and setting temporary paths.
func NewIOConfigForTest() (*IOConfig, error) {
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
}
