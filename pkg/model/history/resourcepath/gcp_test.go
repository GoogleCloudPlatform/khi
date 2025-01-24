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

package resourcepath

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

func TestNetworkEndpointGroup(t *testing.T) {
	testCases := []struct {
		name         string
		negNamespace string
		negName      string
		expected     string
	}{
		{"NEG name specified", "my-namespace", "my-neg", "networking.gke.io/v1beta1#servicenetworkendpointgroup#my-namespace#my-neg"},
		{"Empty NEG namespace", "", "my-neg", "networking.gke.io/v1beta1#servicenetworkendpointgroup#unknown#my-neg"},
		{"Empty NEG name", "my-namespace", "", "networking.gke.io/v1beta1#servicenetworkendpointgroup#my-namespace#unknown"},
		{"Both empty", "", "", "networking.gke.io/v1beta1#servicenetworkendpointgroup#unknown#unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NetworkEndpointGroup(tc.negNamespace, tc.negName)
			if result.Path != tc.expected {
				t.Errorf("NetworkEndpointGroup(%s,%s).Path=%q, want %q", tc.negNamespace, tc.negName, result, tc.expected)
			}
			if result.ParentRelationship != enum.RelationshipChild {
				t.Errorf("NetworkEndpointGroup(%s,%s).ParentRelationshiop=%q, want %q", tc.negNamespace, tc.negName, result.ParentRelationship, enum.RelationshipChild)
			}
		})
	}
}
