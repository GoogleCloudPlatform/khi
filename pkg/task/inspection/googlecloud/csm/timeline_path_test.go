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

package csm

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestMustCSMControlPlaneTimeline(t *testing.T) {
	testCases := []struct {
		name     string
		wantPath func(builder *khifilev6.Builder) *khifilev6.TimelinePath
	}{
		{
			name: "returns Google Managed control plane timeline",
			wantPath: func(builder *khifilev6.Builder) *khifilev6.TimelinePath {
				return builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "Google Managed", Type: gcpcommon.TimelineTypeGCPProject},
					khifilev6.PathSegment{Name: "Control Plane", Type: gcpcommon.TimelineTypeGCPResourceType},
					khifilev6.PathSegment{Name: "istio_control_plane", Type: gcpcommon.TimelineTypeGCPResource},
				)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewTestBuilder(id.NewGenerator())
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

			got := MustCSMControlPlaneTimeline(ctx)
			want := tc.wantPath(builder)
			if diff := cmp.Diff(want, got, protocmp.Transform(), cmpopts.EquateComparable(khifilev6.InternStringRef{})); diff != "" {
				t.Errorf("MustCSMControlPlaneTimeline() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
