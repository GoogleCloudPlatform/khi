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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestTestIOConfigCanFindTheRoot(t *testing.T) {
	ioConfig, err := NewIOConfigForTest()
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

func TestNewIOConfigFromParameter(t *testing.T) {
	testCases := []struct {
		name           string
		params         *parameters.CommonParameters
		expectedErrors bool
		validateFunc   func(t *testing.T, config *IOConfig)
	}{
		{
			name: "default values",
			params: &parameters.CommonParameters{
				DataDestinationFolder: testutil.P("./data"),
				TemporaryFolder:       testutil.P("/tmp"),
			},
			expectedErrors: false,
			validateFunc: func(t *testing.T, config *IOConfig) {
				if !filepath.IsAbs(config.ApplicationRoot) {
					t.Errorf("ApplicationRoot must be absolute path")
				}
				if !filepath.IsAbs(config.DataDestination) {
					t.Errorf("DataDestination must be absolute path")
				}
				if !filepath.IsAbs(config.TemporaryFolder) {
					t.Errorf("TemporaryFolder must be absolute path")
				}
			},
		},
		{
			name: "relative data destination",
			params: &parameters.CommonParameters{
				DataDestinationFolder: testutil.P("./relative/path"),
				TemporaryFolder:       testutil.P("/tmp"),
			},
			expectedErrors: false,
			validateFunc: func(t *testing.T, config *IOConfig) {
				if !filepath.IsAbs(config.DataDestination) {
					t.Errorf("DataDestination should be converted to absolute path")
				}
			},
		},
		{
			name: "absolute paths",
			params: &parameters.CommonParameters{
				DataDestinationFolder: testutil.P("/absolute/data"),
				TemporaryFolder:       testutil.P("/absolute/tmp"),
			},
			expectedErrors: false,
			validateFunc: func(t *testing.T, config *IOConfig) {
				if config.DataDestination != "/absolute/data" {
					t.Errorf("Expected DataDestination to be /absolute/data, got %s", config.DataDestination)
				}
				if config.TemporaryFolder != "/absolute/tmp" {
					t.Errorf("Expected TemporaryFolder to be /absolute/tmp, got %s", config.TemporaryFolder)
				}
			},
		},
		{
			name: "relative temporary folder",
			params: &parameters.CommonParameters{
				DataDestinationFolder: testutil.P("/absolute/data"),
				TemporaryFolder:       testutil.P("./temp"),
			},
			expectedErrors: false,
			validateFunc: func(t *testing.T, config *IOConfig) {
				if !filepath.IsAbs(config.TemporaryFolder) {
					t.Errorf("TemporaryFolder should be converted to absolute path")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := NewIOConfigFromParameter(tc.params)

			if tc.expectedErrors && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.expectedErrors && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if err == nil && tc.validateFunc != nil {
				tc.validateFunc(t, config)
			}
		})
	}
}
