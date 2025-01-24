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

func TestControlplaneComponent(t *testing.T) {
	testCases := []struct {
		name        string
		clusterName string
		component   string
		expected    string
	}{
		{"Component name specified", "cluster-name", "kube-apiserver", "@Cluster#controlplane#cluster-scope#cluster-name#kube-apiserver"},
		{"Empty component name", "cluster-name", "", "@Cluster#controlplane#cluster-scope#cluster-name#unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ControlplaneComponent(tc.clusterName, tc.component)
			if result.Path != tc.expected {
				t.Errorf("ControlplneComponent(%s,%s).Path=%q, want %q", tc.clusterName, tc.component, result, tc.expected)
			}
			if result.ParentRelationship != enum.RelationshipControlPlaneComponent {
				t.Errorf("ControlplaneComponent(%s,%s).ParentRelationshiop=%q, want %q", tc.clusterName, tc.component, result.ParentRelationship, enum.RelationshipControlPlaneComponent)
			}
		})
	}
}
