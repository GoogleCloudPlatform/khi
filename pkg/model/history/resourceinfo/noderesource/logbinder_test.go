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

package noderesource

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type testResourceBinding struct {
	uniqueIdentifier string
	resourcePath     resourcepath.ResourcePath
}

func (t *testResourceBinding) GetUniqueIdentifier() string {
	return t.uniqueIdentifier
}

func (t *testResourceBinding) GetResourcePath() resourcepath.ResourcePath {
	return t.resourcePath
}

func (t *testResourceBinding) RewriteLogSummary(summary string) string {
	return summary
}

func TestLogBinder_AddResourceBinding(t *testing.T) {
	binder := NewLogBinder()
	nodeName := "test-node"
	ra := &testResourceBinding{
		uniqueIdentifier: "foo",
		resourcePath:     resourcepath.Node(nodeName),
	}

	binder.AddResourceBinding(nodeName, ra)

	binder.nodeLogBinderMutex.RLock()
	defer binder.nodeLogBinderMutex.RUnlock()

	if _, ok := binder.nodeLogBinders[nodeName]; !ok {
		t.Errorf("expected nodeLogBinder for node %q, but not found", nodeName)
	}

	if diff := cmp.Diff(binder.nodeLogBinders[nodeName].nodeResourceBindings, []ResourceBinding{ra}, cmp.AllowUnexported(testResourceBinding{})); diff != "" {
		t.Errorf("AddResourceBinding() mismatch (-want +got):\n%s", diff)
	}
}

func TestLogBinder_GetBoundResourcesForLogBody(t *testing.T) {
	binder := NewLogBinder()
	nodeName := "test-node"
	ra1 := &testResourceBinding{
		uniqueIdentifier: "foo",
		resourcePath:     resourcepath.Node(nodeName),
	}
	binder.AddResourceBinding(nodeName, ra1)

	ra2 := &testResourceBinding{
		uniqueIdentifier: "bar",
		resourcePath:     resourcepath.Node(nodeName),
	}
	binder.AddResourceBinding(nodeName, ra2)

	testCases := []struct {
		name     string
		logBody  string
		nodeName string
		expected []ResourceBinding
	}{
		{
			name:     "match 1 resource",
			logBody:  "test log including foo",
			nodeName: nodeName,
			expected: []ResourceBinding{
				ra1,
			},
		},
		{
			name:     "match 2 resources",
			logBody:  "test log including foo and bar",
			nodeName: nodeName,
			expected: []ResourceBinding{
				ra1,
				ra2,
			},
		},
		{
			name:     "no match",
			logBody:  "test log including baz",
			nodeName: nodeName,
			expected: []ResourceBinding{},
		},
		{
			name:     "no binder for node",
			logBody:  "test log including foo",
			nodeName: "non existing node",
			expected: []ResourceBinding{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := binder.GetBoundResourcesForLogBody(tc.nodeName, tc.logBody)
			if diff := cmp.Diff(tc.expected, got, cmp.AllowUnexported(testResourceBinding{})); diff != "" {
				t.Errorf("GetAssociatedResources() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNodeLogBinder_GetBoundResourcesForLogBody(t *testing.T) {
	binder := newNodeLogBinder()

	ra1 := &testResourceBinding{
		uniqueIdentifier: "foo",
		resourcePath:     resourcepath.Node("test-node"),
	}
	binder.AddResourceBinding(ra1)

	ra2 := &testResourceBinding{
		uniqueIdentifier: "bar",
		resourcePath:     resourcepath.Node("test-node-2"),
	}
	binder.AddResourceBinding(ra2)

	testCases := []struct {
		name     string
		logBody  string
		expected []ResourceBinding
	}{
		{
			name:    "match 1 resource",
			logBody: "test log including foo",
			expected: []ResourceBinding{
				ra1,
			},
		},
		{
			name:    "match 2 resources",
			logBody: "test log including foo and bar",
			expected: []ResourceBinding{
				ra1,
				ra2,
			},
		},
		{
			name:     "no match",
			logBody:  "test log including baz",
			expected: []ResourceBinding{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := binder.GetBoundResourcesForLogBody(tc.logBody)
			if diff := cmp.Diff(tc.expected, got, cmp.AllowUnexported(testResourceBinding{})); diff != "" {
				t.Errorf("GetAssociatedResources() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNodeLogBinder_AddResourceBinding(t *testing.T) {
	binder := newNodeLogBinder()

	ra1 := &testResourceBinding{
		uniqueIdentifier: "foo",
		resourcePath:     resourcepath.Node("test-node"),
	}
	binder.AddResourceBinding(ra1)

	// Try adding the same resource binding again - it should be ignored.
	binder.AddResourceBinding(ra1)

	got := binder.nodeResourceBindings
	want := []ResourceBinding{ra1}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(testResourceBinding{})); diff != "" {
		t.Errorf("AddResourceBinding() mismatch (-want +got):\n%s", diff)
	}

	ra2 := &testResourceBinding{
		uniqueIdentifier: "bar",
		resourcePath:     resourcepath.Node("test-node-2"),
	}
	binder.AddResourceBinding(ra2)
	want = []ResourceBinding{ra1, ra2}
	got = binder.nodeResourceBindings

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(testResourceBinding{})); diff != "" {
		t.Errorf("AddResourceBinding() mismatch (-want +got):\n%s", diff)
	}

}
