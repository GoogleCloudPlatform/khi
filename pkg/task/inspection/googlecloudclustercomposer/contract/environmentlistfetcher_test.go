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

package googlecloudclustercomposer_contract

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/composer/v1"
)

func TestApiEnvironmentToClusterName(t *testing.T) {
	tests := []struct {
		name     string
		resp     *composer.Environment
		expected string
	}{
		{
			name: "Normal case with multiple environments",
			resp: &composer.Environment{
				Name: "projects/foo/locations/us-central1/environments/env1",
			},
			expected: "env1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := apiEnvironmentToClusterName(tt.resp)
			if diff := cmp.Diff(tt.expected, actual); diff != "" {
				t.Errorf("apiResponseToClusterNameList() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
