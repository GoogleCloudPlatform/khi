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

package resourceinfo

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
)

func TestGetNodeResourceIdTypeFromID(t *testing.T) {
	info := NewClusterResourceInfo()
	info.PodSandboxIDs.TouchResourceLease("pod-id-A", time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC), &resourcelease.K8sResourceLeaseHolder{})
	info.ContainerIDs.TouchResourceLease("container-id-A", time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC), &resourcelease.ContainerLeaseHolder{})
	testCases := []struct {
		name      string
		query     string
		queryTime time.Time
		want      NodeResourceIDType
	}{
		{
			name:      "with a valid pod id",
			query:     "pod-id-A",
			queryTime: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			want:      NodeResourceIDTypePodSandbox,
		}, {
			name:      "with a valid container id",
			query:     "container-id-A",
			queryTime: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			want:      NodeResourceIDTypeContainer,
		},
		{
			name:      "with an invalid id",
			query:     "invalid-id",
			queryTime: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			want:      NodeResourceIDTypeUnknown,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := info.GetNodeResourceIDTypeFromID(tc.query, tc.queryTime)

			if got != tc.want {
				t.Errorf("GetNodeResourceIDTypeFromID(%s,%s)=%v, want %v", tc.query, tc.queryTime, got, tc.want)
			}
		})
	}
}
