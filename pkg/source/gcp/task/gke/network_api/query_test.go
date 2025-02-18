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

package network_api

import (
	"testing"

	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
)

func TestGenerateGenerateGCPNetworkAPIQueryIsValid(t *testing.T) {

	testCases := []struct {
		Name string
		NEGs []string
	}{
		{
			Name: "Valid Query",
			NEGs: []string{"neg-1", "neg-2"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			query := GenerateGCPNetworkAPIQuery(0, tc.NEGs)
			err := gcp_test.IsValidLogQuery(query[0])
			if err != nil {
				t.Errorf("Query is not valid: %v", err)
			}
		})
	}
}
