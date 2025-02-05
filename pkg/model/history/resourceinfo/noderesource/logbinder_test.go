package noderesource

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/google/go-cmp/cmp"
)

type testResourceBidinAddResourceBinding struct {
	uniqueIdentifier string
	resourcePath     resourcepath.ResourcePath
}

func (t *testResourceBidinAddResourceBinding) GetUniqueIdentifier() string {
	return t.uniqueIdentifier
}

func (t *testResourceBidinAddResourceBinding) GetResourcePath() resourcepath.ResourcePath {
	return t.resourcePath
}

func (t *testResourceBidinAddResourceBinding) RewriteLogSummary(summary string) string {
	return summary
}

func TestGetAssociatedResources(t *testing.T) {
	binder := newNodeLogBinder()

	ra1 := &testResourceBidinAddResourceBinding{
		uniqueIdentifier: "foo",
		resourcePath:     resourcepath.Node("test-node"),
	}
	binder.AddResourceBinding(ra1)

	ra2 := &testResourceBidinAddResourceBinding{
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
			got := binder.GetAssociatedResources(tc.logBody)
			if diff := cmp.Diff(tc.expected, got, cmp.AllowUnexported(testResourceBidinAddResourceBinding{})); diff != "" {
				t.Errorf("GetAssociatedResources() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddResourceBinding(t *testing.T) {
	binder := newNodeLogBinder()

	ra1 := &testResourceBidinAddResourceBinding{
		uniqueIdentifier: "foo",
		resourcePath:     resourcepath.Node("test-node"),
	}
	binder.AddResourceBinding(ra1)

	// Try adding the same resource association again - it should be ignored.
	binder.AddResourceBinding(ra1)

	got := binder.nodeResourceBindings
	want := []ResourceBinding{ra1}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(testResourceBidinAddResourceBinding{})); diff != "" {
		t.Errorf("AddResourceBinding() mismatch (-want +got):\n%s", diff)
	}

	ra2 := &testResourceBidinAddResourceBinding{
		uniqueIdentifier: "bar",
		resourcePath:     resourcepath.Node("test-node-2"),
	}
	binder.AddResourceBinding(ra2)
	want = []ResourceBinding{ra1, ra2}
	got = binder.nodeResourceBindings

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(testResourceBidinAddResourceBinding{})); diff != "" {
		t.Errorf("AddResourceBinding() mismatch (-want +got):\n%s", diff)
	}

}
