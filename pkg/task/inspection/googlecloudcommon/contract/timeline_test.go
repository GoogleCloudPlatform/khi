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
		wantPanic   bool
		panicMsg    string
	}{
		{
			name:        "valid cluster name",
			clusterName: "my-gke-cluster",
			wantPanic:   false,
		},
		{
			name:        "empty cluster name",
			clusterName: "",
			wantPanic:   true,
			panicMsg:    "cluster name must not be empty",
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

			got := MustGKEClusterTimeline(ctx, tc.clusterName)
			if !tc.wantPanic {
				if got == nil {
					t.Fatal("expected timeline path to be not nil")
				}
				if diff := cmp.Diff(tc.clusterName, got.Name.Resolve()); diff != "" {
					t.Errorf("MustGKEClusterTimeline() mismatch (-want +got):\n%s", diff)
				}
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
	}{
		{
			name:         "valid nodepool name",
			parent:       gkeClusterTimeline,
			nodePoolName: "default-pool",
			wantPanic:    false,
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
			wantPanic:    true,
			panicMsg:     "node pool name must not be empty",
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
				if diff := cmp.Diff(tc.nodePoolName, got.Name.Resolve()); diff != "" {
					t.Errorf("MustGKENodePoolTimeline() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
