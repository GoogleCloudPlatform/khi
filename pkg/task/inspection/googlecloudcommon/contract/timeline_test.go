// Copyright 2026 Google LLC
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

package googlecloudcommon_contract

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestMustGKEClusterTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	testCases := []struct {
		name        string
		clusterName string
		wantName    string
	}{
		{
			name:        "valid cluster name",
			clusterName: "my-gke-cluster",
			wantName:    "my-gke-cluster",
		},
		{
			name:        "empty cluster name",
			clusterName: "",
			wantName:    "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MustGKEClusterTimeline(ctx, tc.clusterName)
			if got == nil {
				t.Fatal("expected timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantName, got.Name.Resolve()); diff != "" {
				t.Errorf("MustGKEClusterTimeline() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMustGKENodePoolTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	gkeClusterTimeline := MustGKEClusterTimeline(ctx, "test-gke-cluster")
	invalidParentTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "invalid",
		Type: TimelineTypeOperation,
	})

	testCases := []struct {
		name         string
		parent       *khifilev6.TimelinePath
		nodePoolName string
		wantPanic    bool
		panicMsg     string
		wantName     string
	}{
		{
			name:         "valid nodepool name",
			parent:       gkeClusterTimeline,
			nodePoolName: "default-pool",
			wantPanic:    false,
			wantName:     "default-pool",
		},
		{
			name:         "invalid parent type",
			parent:       invalidParentTimeline,
			nodePoolName: "default-pool",
			wantPanic:    true,
			panicMsg:     "parent timeline path must be GKE type",
		},
		{
			name:         "empty nodepool name",
			parent:       gkeClusterTimeline,
			nodePoolName: "",
			wantPanic:    false,
			wantName:     "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantPanic {
				defer func() {
					r := recover()
					if r == nil {
						t.Error("expected panic but did not panic")
					} else {
						errStr := fmt.Sprintf("%v", r)
						if !strings.Contains(errStr, tc.panicMsg) {
							t.Errorf("expected panic message to contain %q, got %q", tc.panicMsg, errStr)
						}
					}
				}()
			}

			got := MustGKENodePoolTimeline(ctx, tc.parent, tc.nodePoolName)
			if !tc.wantPanic {
				if got == nil {
					t.Fatal("expected timeline path to be not nil")
				}
				if diff := cmp.Diff(tc.wantName, got.Name.Resolve()); diff != "" {
					t.Errorf("MustGKENodePoolTimeline() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMustGCPOperationTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	gkeClusterTimeline := MustGKEClusterTimeline(ctx, "test-gke-cluster")

	testCases := []struct {
		name            string
		parent          *khifilev6.TimelinePath
		shortMethodName string
		operationID     string
		wantPanic       bool
		panicMsg        string
		wantName        string
	}{
		{
			name:            "valid inputs",
			parent:          gkeClusterTimeline,
			shortMethodName: "CreateMesh",
			operationID:     "op1",
			wantPanic:       false,
			wantName:        "CreateMesh-op1",
		},
		{
			name:            "nil parent",
			parent:          nil,
			shortMethodName: "CreateMesh",
			operationID:     "op1",
			wantPanic:       true,
			panicMsg:        "parent timeline path must not be nil",
		},
		{
			name:            "empty shortMethodName",
			parent:          gkeClusterTimeline,
			shortMethodName: "",
			operationID:     "op1",
			wantPanic:       false,
			wantName:        "unknown-op1",
		},
		{
			name:            "empty operationID",
			parent:          gkeClusterTimeline,
			shortMethodName: "CreateMesh",
			operationID:     "",
			wantPanic:       false,
			wantName:        "CreateMesh-unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantPanic {
				defer func() {
					r := recover()
					if r == nil {
						t.Error("expected panic but did not panic")
					} else {
						errStr := fmt.Sprintf("%v", r)
						if !strings.Contains(errStr, tc.panicMsg) {
							t.Errorf("expected panic message to contain %q, got %q", tc.panicMsg, errStr)
						}
					}
				}()
			}

			got := MustGCPOperationTimeline(ctx, tc.parent, tc.shortMethodName, tc.operationID)
			if !tc.wantPanic {
				if got == nil {
					t.Fatal("expected timeline path to be not nil")
				}
				if diff := cmp.Diff(tc.wantName, got.Name.Resolve()); diff != "" {
					t.Errorf("MustGCPOperationTimeline() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMustGCPProjectTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	testCases := []struct {
		name      string
		projectID string
		wantName  string
	}{
		{
			name:      "valid project",
			projectID: "my-project",
			wantName:  "my-project",
		},
		{
			name:      "empty project",
			projectID: "",
			wantName:  "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MustGCPProjectTimeline(ctx, tc.projectID)
			if got == nil {
				t.Fatal("expected timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantName, got.Name.Resolve()); diff != "" {
				t.Errorf("MustGCPProjectTimeline() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMustGCPResourceTypeTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	projectTimeline := MustGCPProjectTimeline(ctx, "test-project")
	invalidParentTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "invalid",
		Type: TimelineTypeOperation,
	})

	testCases := []struct {
		name         string
		parent       *khifilev6.TimelinePath
		resourceType string
		wantPanic    bool
		panicMsg     string
		wantName     string
	}{
		{
			name:         "valid type",
			parent:       projectTimeline,
			resourceType: "meshes",
			wantPanic:    false,
			wantName:     "meshes",
		},
		{
			name:         "invalid parent type",
			parent:       invalidParentTimeline,
			resourceType: "meshes",
			wantPanic:    true,
			panicMsg:     "parent timeline path must be GCPProject type",
		},
		{
			name:         "empty resource type",
			parent:       projectTimeline,
			resourceType: "",
			wantPanic:    false,
			wantName:     "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantPanic {
				defer func() {
					r := recover()
					if r == nil {
						t.Error("expected panic but did not panic")
					} else {
						errStr := fmt.Sprintf("%v", r)
						if !strings.Contains(errStr, tc.panicMsg) {
							t.Errorf("expected panic message to contain %q, got %q", tc.panicMsg, errStr)
						}
					}
				}()
			}

			got := MustGCPResourceTypeTimeline(ctx, tc.parent, tc.resourceType)
			if !tc.wantPanic {
				if got == nil {
					t.Fatal("expected timeline path to be not nil")
				}
				if diff := cmp.Diff(tc.wantName, got.Name.Resolve()); diff != "" {
					t.Errorf("MustGCPResourceTypeTimeline() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMustGCPResourceTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	projectTimeline := MustGCPProjectTimeline(ctx, "test-project")
	resourceTypeTimeline := MustGCPResourceTypeTimeline(ctx, projectTimeline, "meshes")
	invalidParentTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "invalid",
		Type: TimelineTypeOperation,
	})

	testCases := []struct {
		name         string
		parent       *khifilev6.TimelinePath
		resourceName string
		wantPanic    bool
		panicMsg     string
		wantName     string
	}{
		{
			name:         "valid resource name",
			parent:       resourceTypeTimeline,
			resourceName: "my-mesh",
			wantPanic:    false,
			wantName:     "my-mesh",
		},
		{
			name:         "invalid parent type",
			parent:       invalidParentTimeline,
			resourceName: "my-mesh",
			wantPanic:    true,
			panicMsg:     "parent timeline path must be GCPResourceType type",
		},
		{
			name:         "empty resource name",
			parent:       resourceTypeTimeline,
			resourceName: "",
			wantPanic:    false,
			wantName:     "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantPanic {
				defer func() {
					r := recover()
					if r == nil {
						t.Error("expected panic but did not panic")
					} else {
						errStr := fmt.Sprintf("%v", r)
						if !strings.Contains(errStr, tc.panicMsg) {
							t.Errorf("expected panic message to contain %q, got %q", tc.panicMsg, errStr)
						}
					}
				}()
			}

			got := MustGCPResourceTimeline(ctx, tc.parent, tc.resourceName)
			if !tc.wantPanic {
				if got == nil {
					t.Fatal("expected timeline path to be not nil")
				}
				if diff := cmp.Diff(tc.wantName, got.Name.Resolve()); diff != "" {
					t.Errorf("MustGCPResourceTimeline() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
