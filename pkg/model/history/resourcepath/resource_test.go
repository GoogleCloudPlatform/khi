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

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestContainer(t *testing.T) {
	testCases := []struct {
		name          string
		namespace     string
		podName       string
		containerName string
		expected      string
	}{
		{"All specified", "my-namespace", "my-pod", "my-container", "core/v1#pod#my-namespace#my-pod#my-container"},
		{"Empty namespace", "", "my-pod", "my-container", "core/v1#pod#unknown#my-pod#my-container"},
		{"Empty pod name", "my-namespace", "", "my-container", "core/v1#pod#my-namespace#unknown#my-container"},
		{"Empty container name", "my-namespace", "my-pod", "", "core/v1#pod#my-namespace#my-pod#unknown"},
		{"Two empty", "", "", "my-container", "core/v1#pod#unknown#unknown#my-container"},
		{"Two empty #2", "my-namespace", "", "", "core/v1#pod#my-namespace#unknown#unknown"},
		{"All empty", "", "", "", "core/v1#pod#unknown#unknown#unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Container(tc.namespace, tc.podName, tc.containerName)
			if result.Path != tc.expected {
				t.Errorf("got unexpected path %q, want %q", result.Path, tc.expected)
			}
			if result.ParentRelationship != enum.RelationshipContainer {
				t.Errorf("got unexpected relationship %q, want %q", result.ParentRelationship, enum.RelationshipContainer)
			}
		})
	}
}

func TestPod(t *testing.T) {
	testCases := []struct {
		name      string
		namespace string
		podName   string
		expected  string
	}{
		{"All specified", "my-namespace", "my-pod", "core/v1#pod#my-namespace#my-pod"},
		{"Empty namespace", "", "my-pod", "core/v1#pod#unknown#my-pod"},
		{"Empty pod name", "my-namespace", "", "core/v1#pod#my-namespace#unknown"},
		{"Both empty", "", "", "core/v1#pod#unknown#unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Pod(tc.namespace, tc.podName)
			if result.Path != tc.expected {
				t.Errorf("got unexpected path %q, want %q", result.Path, tc.expected)
			}
			if result.ParentRelationship != enum.RelationshipChild {
				t.Errorf("got unexpected relationship %q, want %q", result.ParentRelationship, enum.RelationshipContainer)
			}
		})
	}
}

func TestNode(t *testing.T) {
	testCases := []struct {
		name     string
		nodeName string
		expected string
	}{
		{"Node name specified", "my-node", "core/v1#node#cluster-scope#my-node"},
		{"Empty node name", "", "core/v1#node#cluster-scope#unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Node(tc.nodeName)
			if result.Path != tc.expected {
				t.Errorf("got unexpected path %q, want %q", result.Path, tc.expected)
			}
			if result.ParentRelationship != enum.RelationshipChild {
				t.Errorf("got unexpected relationship %q, want %q", result.ParentRelationship, enum.RelationshipContainer)
			}
		})
	}
}
