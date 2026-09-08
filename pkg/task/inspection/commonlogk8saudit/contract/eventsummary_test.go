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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/google/go-cmp/cmp"
)

func TestFormatEventSummary(t *testing.T) {
	testCases := []struct {
		desc                    string
		reason                  string
		message                 string
		resourceIdentitiesByUID map[string]*ResourceIdentity
		want                    string
	}{
		{
			desc:    "empty message returns bracketed reason",
			reason:  "Scheduled",
			message: "",
			want:    "【Scheduled】",
		},
		{
			desc:    "message without UIDs returns reason and message",
			reason:  "Scheduled",
			message: "Successfully assigned default/my-pod to node-1",
			want:    "【Scheduled】Successfully assigned default/my-pod to node-1",
		},
		{
			desc:    "message with single UID replaces UID with summary tag",
			reason:  "Scheduled",
			message: "Assigned pod (UID: 11112222-3333-4444-5555-666677778888) to node",
			resourceIdentitiesByUID: map[string]*ResourceIdentity{
				"11112222-3333-4444-5555-666677778888": {
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "my-pod",
				},
			},
			want: "【Scheduled】Assigned pod (UID: 【my-pod (Namespace: default, APIVersion: core/v1, Kind: pod)】) to node",
		},
		{
			desc:    "message with multiple distinct UIDs replaces all UIDs",
			reason:  "SyncLoop",
			message: "Processing pod: uid-1 and service: uid-2",
			resourceIdentitiesByUID: map[string]*ResourceIdentity{
				"uid-1": {
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "my-pod",
				},
				"uid-2": {
					APIVersion: "core/v1",
					Kind:       "service",
					Namespace:  "default",
					Name:       "my-service",
				},
			},
			want: "【SyncLoop】Processing pod: 【my-pod (Namespace: default, APIVersion: core/v1, Kind: pod)】 and service: 【my-service (Namespace: default, APIVersion: core/v1, Kind: service)】",
		},
		{
			desc:    "message with duplicate UID occurrences replaces all instances",
			reason:  "SyncLoop",
			message: "Processing pod: uid-1 and retry for uid-1",
			resourceIdentitiesByUID: map[string]*ResourceIdentity{
				"uid-1": {
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "my-pod",
				},
			},
			want: "【SyncLoop】Processing pod: 【my-pod (Namespace: default, APIVersion: core/v1, Kind: pod)】 and retry for 【my-pod (Namespace: default, APIVersion: core/v1, Kind: pod)】",
		},
		{
			desc:    "message with cluster-scoped resource UID formats tag without namespace",
			reason:  "NodeReady",
			message: "Node uid-node is ready",
			resourceIdentitiesByUID: map[string]*ResourceIdentity{
				"uid-node": {
					APIVersion: "core/v1",
					Kind:       "node",
					Namespace:  "",
					Name:       "node-1",
				},
			},
			want: "【NodeReady】Node 【node-1 (APIVersion: core/v1, Kind: node)】 is ready",
		},
		{
			desc:    "message with unindexed UID leaves text unchanged",
			reason:  "FailedScheduling",
			message: "Pod with unknown UID 00000000-0000-0000-0000-000000000000 cannot be scheduled",
			resourceIdentitiesByUID: map[string]*ResourceIdentity{
				"other-uid": {
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "other-pod",
				},
			},
			want: "【FailedScheduling】Pod with unknown UID 00000000-0000-0000-0000-000000000000 cannot be scheduled",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			finder := patternfinder.NewNaivePatternFinder[*ResourceIdentity]()
			for k, v := range tc.resourceIdentitiesByUID {
				_ = finder.AddPattern(k, v)
			}
			got := FormatEventSummary(tc.reason, tc.message, finder)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FormatEventSummary() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
