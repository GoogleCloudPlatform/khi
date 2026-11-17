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

package dpv2policy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func TestMustNetworkPolicyTimeline(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())
	ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

	podPath := builder.TimelineAccumulator.GetPath(nil,
		khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
		khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion},
		khifilev6.PathSegment{Name: "pod", Type: inspectioncore.TimelineTypeKind},
		khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace},
		khifilev6.PathSegment{Name: "httpbin-0", Type: inspectioncore.TimelineTypeResource},
	)
	wantPodOutboundPath := builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{Name: "outbound", Type: TimelineTypeNetworkPolicy})
	namespacePath := builder.TimelineAccumulator.GetPath(nil,
		khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
		khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion},
		khifilev6.PathSegment{Name: "pod", Type: inspectioncore.TimelineTypeKind},
		khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace},
	)

	testCases := []struct {
		desc         string
		parent       *khifilev6.TimelinePath
		timelineName string
		want         *khifilev6.TimelinePath
		wantPanicMsg string
	}{
		{
			desc:         "timeline under a Pod resource timeline",
			parent:       podPath,
			timelineName: PodOutboundTimelineName,
			want:         wantPodOutboundPath,
		},
		{
			desc:         "parent timeline is not a resource timeline",
			parent:       namespacePath,
			timelineName: PodOutboundTimelineName,
			wantPanicMsg: "parent timeline path must be Resource type",
		},
		{
			desc:         "parent timeline is nil",
			parent:       nil,
			timelineName: NodeTimelineName,
			wantPanicMsg: "parent timeline path must be Resource type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.wantPanicMsg != "" {
				defer func() {
					r := recover()
					if r == nil {
						t.Fatalf("MustNetworkPolicyTimeline() did not panic, want panic containing %q", tc.wantPanicMsg)
					}
					if got := fmt.Sprintf("%v", r); !strings.Contains(got, tc.wantPanicMsg) {
						t.Errorf("MustNetworkPolicyTimeline() panicked with %q, want message containing %q", got, tc.wantPanicMsg)
					}
				}()
			}

			got := MustNetworkPolicyTimeline(ctx, tc.parent, tc.timelineName)
			if got != tc.want {
				t.Errorf("MustNetworkPolicyTimeline() = %v, want %v", got, tc.want)
			}
		})
	}
}
