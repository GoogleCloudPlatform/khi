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

package commonlogk8saudit_contract

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestMustK8sClusterTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	testCases := []struct {
		name        string
		clusterName string
		wantName    string
	}{
		{
			name:        "valid cluster name",
			clusterName: "my-cluster",
			wantName:    "my-cluster",
		},
		{
			name:        "empty cluster name",
			clusterName: "",
			wantName:    "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MustK8sClusterTimeline(ctx, tc.clusterName)
			if got == nil {
				t.Fatal("expected timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantName, got.Name.Resolve()); diff != "" {
				t.Errorf("MustK8sClusterTimeline() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMustK8sAPIVersionTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	clusterTimeline := MustK8sClusterTimeline(ctx, "test-cluster")
	invalidParentTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "invalid",
		Type: inspectioncore_contract.TimelineTypeKind,
	})

	testCases := []struct {
		name       string
		parent     *khifilev6.TimelinePath
		apiVersion string
		wantPanic  bool
		panicMsg   string
	}{
		{
			name:       "valid apiVersion",
			parent:     clusterTimeline,
			apiVersion: "apps/v1",
			wantPanic:  false,
		},
		{
			name:       "invalid parent type",
			parent:     invalidParentTimeline,
			apiVersion: "apps/v1",
			wantPanic:  true,
			panicMsg:   "parent timeline path must be K8sCluster type",
		},
		{
			name:       "invalid apiVersion without slash",
			parent:     clusterTimeline,
			apiVersion: "v1",
			wantPanic:  true,
			panicMsg:   "Missing core/?",
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

			got := MustK8sAPIVersionTimeline(ctx, tc.parent, tc.apiVersion)
			if !tc.wantPanic {
				if got == nil {
					t.Fatal("expected timeline path to be not nil")
				}
				if diff := cmp.Diff(tc.apiVersion, got.Name.Resolve()); diff != "" {
					t.Errorf("MustK8sAPIVersionTimeline() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMustK8sKindTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	clusterTimeline := MustK8sClusterTimeline(ctx, "test-cluster")
	apiVersionPath := MustK8sAPIVersionTimeline(ctx, clusterTimeline, "apps/v1")
	invalidParentPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "invalid",
		Type: inspectioncore_contract.TimelineTypeKind,
	})

	testCases := []struct {
		name      string
		parent    *khifilev6.TimelinePath
		kind      string
		wantPanic bool
		panicMsg  string
	}{
		{
			name:      "valid kind",
			parent:    apiVersionPath,
			kind:      "deployment",
			wantPanic: false,
		},
		{
			name:      "invalid parent type",
			parent:    invalidParentPath,
			kind:      "pod",
			wantPanic: true,
			panicMsg:  "parent timeline path must be APIVersion type",
		},
		{
			name:      "invalid kind with uppercase letters",
			parent:    apiVersionPath,
			kind:      "Pod",
			wantPanic: true,
			panicMsg:  "Kind must be all lowercase",
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

			got := MustK8sKindTimeline(ctx, tc.parent, tc.kind)
			if !tc.wantPanic {
				if got == nil {
					t.Fatal("expected timeline path to be not nil")
				}
				if diff := cmp.Diff(tc.kind, got.Name.Resolve()); diff != "" {
					t.Errorf("MustK8sKindTimeline() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMustK8sNamespaceTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	clusterTimeline := MustK8sClusterTimeline(ctx, "test-cluster")
	apiVersionPath := MustK8sAPIVersionTimeline(ctx, clusterTimeline, "apps/v1")
	kindPath := MustK8sKindTimeline(ctx, apiVersionPath, "deployment")
	invalidParentPath := apiVersionPath

	testCases := []struct {
		name      string
		parent    *khifilev6.TimelinePath
		namespace string
		wantPanic bool
		panicMsg  string
	}{
		{
			name:      "valid namespace",
			parent:    kindPath,
			namespace: "default",
			wantPanic: false,
		},
		{
			name:      "invalid parent type",
			parent:    invalidParentPath,
			namespace: "default",
			wantPanic: true,
			panicMsg:  "parent timeline path must be Kind type",
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

			got := MustK8sNamespaceTimeline(ctx, tc.parent, tc.namespace)
			if !tc.wantPanic {
				if got == nil {
					t.Fatal("expected timeline path to be not nil")
				}
				if diff := cmp.Diff(tc.namespace, got.Name.Resolve()); diff != "" {
					t.Errorf("MustK8sNamespaceTimeline() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMustK8sNamespacedResourceTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	clusterTimeline := MustK8sClusterTimeline(ctx, "test-cluster")
	apiVersionPath := MustK8sAPIVersionTimeline(ctx, clusterTimeline, "apps/v1")
	kindPath := MustK8sKindTimeline(ctx, apiVersionPath, "deployment")
	namespacePath := MustK8sNamespaceTimeline(ctx, kindPath, "default")
	invalidParentPath := kindPath

	testCases := []struct {
		name         string
		parent       *khifilev6.TimelinePath
		resourceName string
		wantPanic    bool
		panicMsg     string
	}{
		{
			name:         "valid resource name",
			parent:       namespacePath,
			resourceName: "my-deployment",
			wantPanic:    false,
		},
		{
			name:         "invalid parent type",
			parent:       invalidParentPath,
			resourceName: "my-deployment",
			wantPanic:    true,
			panicMsg:     "parent timeline path must be Namespace type",
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

			got := MustK8sNamespacedResourceTimeline(ctx, tc.parent, tc.resourceName)
			if !tc.wantPanic {
				if got == nil {
					t.Fatal("expected timeline path to be not nil")
				}
				if diff := cmp.Diff(tc.resourceName, got.Name.Resolve()); diff != "" {
					t.Errorf("MustK8sNamespacedResourceTimeline() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMustK8sClusterScopeResourceTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	clusterTimeline := MustK8sClusterTimeline(ctx, "test-cluster")
	corePath := MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindPath := MustK8sKindTimeline(ctx, corePath, "node")
	invalidParentPath := MustK8sNamespaceTimeline(ctx, kindPath, "default")

	testCases := []struct {
		name         string
		parent       *khifilev6.TimelinePath
		resourceName string
		wantPanic    bool
		panicMsg     string
	}{
		{
			name:         "valid resource name",
			parent:       kindPath,
			resourceName: "my-node",
			wantPanic:    false,
		},
		{
			name:         "invalid parent type",
			parent:       invalidParentPath,
			resourceName: "my-node",
			wantPanic:    true,
			panicMsg:     "parent timeline path must be Kind type",
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

			got := MustK8sClusterScopeResourceTimeline(ctx, tc.parent, tc.resourceName)
			if !tc.wantPanic {
				if got == nil {
					t.Fatal("expected timeline path to be not nil")
				}
				if diff := cmp.Diff(tc.resourceName, got.Name.Resolve()); diff != "" {
					t.Errorf("MustK8sClusterScopeResourceTimeline() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMustK8sSubresourceTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	clusterTimeline := MustK8sClusterTimeline(ctx, "test-cluster")
	apiVersionPath := MustK8sAPIVersionTimeline(ctx, clusterTimeline, "apps/v1")
	kindPath := MustK8sKindTimeline(ctx, apiVersionPath, "deployment")
	namespacePath := MustK8sNamespaceTimeline(ctx, kindPath, "default")
	resourcePath := MustK8sNamespacedResourceTimeline(ctx, namespacePath, "my-deployment")
	invalidParentPath := namespacePath

	testCases := []struct {
		name            string
		parent          *khifilev6.TimelinePath
		subresourceName string
		wantPanic       bool
		panicMsg        string
	}{
		{
			name:            "valid subresource",
			parent:          resourcePath,
			subresourceName: "status",
			wantPanic:       false,
		},
		{
			name:            "invalid parent type",
			parent:          invalidParentPath,
			subresourceName: "status",
			wantPanic:       true,
			panicMsg:        "parent timeline path must be Resource type",
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

			got := MustK8sSubresourceTimeline(ctx, tc.parent, tc.subresourceName)
			if !tc.wantPanic {
				if got == nil {
					t.Fatal("expected timeline path to be not nil")
				}
				if diff := cmp.Diff(tc.subresourceName, got.Name.Resolve()); diff != "" {
					t.Errorf("MustK8sSubresourceTimeline() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
