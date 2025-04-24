// Copyright 2025 Google LLC
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

package api

import (
	"strings"
	"testing"
)

func TestValidateResourceNameOnLogEntriesList(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid two-segment project resource",
			resourceName: "projects/my-project",
			wantErr:      false,
		},
		{
			name:         "valid two-segment organization resource",
			resourceName: "organizations/123456789",
			wantErr:      false,
		},
		{
			name:         "valid two-segment folder resource",
			resourceName: "folders/my-folder",
			wantErr:      false,
		},
		{
			name:         "valid two-segment billing account resource",
			resourceName: "billingAccounts/BA12345",
			wantErr:      false,
		},
		{
			name:         "valid eight-segment log view resource",
			resourceName: "projects/my-project/locations/global/buckets/my-bucket/views/my-view",
			wantErr:      false,
		},
		{
			name:         "valid organization log view resource",
			resourceName: "organizations/123456789/locations/us-central1/buckets/primary-bucket/views/debug-view",
			wantErr:      false,
		},
		{
			name:         "invalid resource name root",
			resourceName: "invalid/my-project",
			wantErr:      true,
			errContains:  "resource name must begin with one of the following",
		},
		{
			name:         "invalid segment count",
			resourceName: "projects/my-project/extra",
			wantErr:      true,
			errContains:  "resource name must have 2 or 8 segments",
		},
		{
			name:         "invalid log view - wrong locations segment",
			resourceName: "projects/my-project/wrongword/global/buckets/my-bucket/views/my-view",
			wantErr:      true,
			errContains:  "locations",
		},
		{
			name:         "invalid log view - wrong buckets segment",
			resourceName: "projects/my-project/locations/global/wrongword/my-bucket/views/my-view",
			wantErr:      true,
			errContains:  "buckets",
		},
		{
			name:         "invalid log view - wrong views segment",
			resourceName: "projects/my-project/locations/global/buckets/my-bucket/wrongword/my-view",
			wantErr:      true,
			errContains:  "views",
		},
		{
			name:         "empty resource name",
			resourceName: "",
			wantErr:      true,
			errContains:  "resource name must begin with one of the following",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResourceNameOnLogEntriesList(tt.resourceName)

			// Check if error occurred
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResourceNameOnLogEntriesList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we want an error, check that it contains the expected text
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateResourceNameOnLogEntriesList() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}
