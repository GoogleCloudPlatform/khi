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

package googlecloudcaik8s_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	"github.com/google/go-cmp/cmp"
)

func newTestNodeReader(t *testing.T, data map[string]any) *structured.NodeReader {
	t.Helper()
	node, err := structured.FromGoValue(data, &structured.AlphabeticalGoMapKeyOrderProvider{})
	if err != nil {
		t.Fatalf("failed to create structured node from map: %v", err)
	}
	return structured.NewNodeReader(node)
}

func TestExtractResourceIdentityFromLog(t *testing.T) {
	testCases := []struct {
		name      string
		inputData map[string]any
		want      *commonlogk8saudit_contract.ResourceIdentity
	}{
		{
			name: "extracts from manifest data for namespaced pod",
			inputData: map[string]any{
				"asset": map[string]any{
					"name":      "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/default/pods/pod-1",
					"assetType": "k8s.io/Pod",
					"resource": map[string]any{
						"data": map[string]any{
							"apiVersion": "v1",
							"kind":       "Pod",
							"metadata": map[string]any{
								"name":      "pod-1",
								"namespace": "default",
							},
						},
					},
				},
			},
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Name:       "pod-1",
				Namespace:  "default",
			},
		},
		{
			name: "extracts from manifest data for apps/v1 deployment",
			inputData: map[string]any{
				"asset": map[string]any{
					"name":      "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/prod/deployments/web",
					"assetType": "apps.k8s.io/Deployment",
					"resource": map[string]any{
						"data": map[string]any{
							"apiVersion": "apps/v1",
							"kind":       "Deployment",
							"metadata": map[string]any{
								"name":      "web",
								"namespace": "prod",
							},
						},
					},
				},
			},
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "apps/v1",
				Kind:       "deployment",
				Name:       "web",
				Namespace:  "prod",
			},
		},
		{
			name: "falls back to asset name and asset type when manifest data is missing",
			inputData: map[string]any{
				"asset": map[string]any{
					"name":      "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/custom-ns/services/my-svc",
					"assetType": "k8s.io/Service",
				},
			},
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "service",
				Name:       "my-svc",
				Namespace:  "custom-ns",
			},
		},
		{
			name: "extracts cluster-scoped resource without namespace",
			inputData: map[string]any{
				"asset": map[string]any{
					"name":      "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/nodes/node-1",
					"assetType": "k8s.io/Node",
					"resource": map[string]any{
						"data": map[string]any{
							"apiVersion": "v1",
							"kind":       "Node",
							"metadata": map[string]any{
								"name": "node-1",
							},
						},
					},
				},
			},
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "node",
				Name:       "node-1",
				Namespace:  "",
			},
		},
		{
			name: "clears namespace when kind is namespace",
			inputData: map[string]any{
				"asset": map[string]any{
					"name":      "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/kube-system",
					"assetType": "k8s.io/Namespace",
					"resource": map[string]any{
						"data": map[string]any{
							"apiVersion": "v1",
							"kind":       "Namespace",
							"metadata": map[string]any{
								"name":      "kube-system",
								"namespace": "kube-system",
							},
						},
					},
				},
			},
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "namespace",
				Name:       "kube-system",
				Namespace:  "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := newTestNodeReader(t, tc.inputData)
			got := extractResourceIdentityFromLog(reader)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("extractResourceIdentityFromLog() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractTimeWindow(t *testing.T) {
	testCases := []struct {
		name          string
		inputData     map[string]any
		wantStartTime time.Time
		wantEndTime   time.Time
		wantDeleted   bool
	}{
		{
			name: "extracts valid time window and deleted false",
			inputData: map[string]any{
				"window": map[string]any{
					"startTime": "2026-01-01T10:00:00Z",
					"endTime":   "2026-01-01T11:00:00Z",
				},
				"deleted": false,
			},
			wantStartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			wantEndTime:   time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
			wantDeleted:   false,
		},
		{
			name: "extracts tombstone deleted true",
			inputData: map[string]any{
				"deleted": true,
			},
			wantStartTime: time.Time{},
			wantEndTime:   time.Time{},
			wantDeleted:   true,
		},
		{
			name:          "extracts empty map as zero times and not deleted",
			inputData:     map[string]any{},
			wantStartTime: time.Time{},
			wantEndTime:   time.Time{},
			wantDeleted:   false,
		},
		{
			name: "falls back to zero times when timestamp strings are malformed",
			inputData: map[string]any{
				"window": map[string]any{
					"startTime": "invalid-start-time",
					"endTime":   "invalid-end-time",
				},
				"deleted": false,
			},
			wantStartTime: time.Time{},
			wantEndTime:   time.Time{},
			wantDeleted:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := newTestNodeReader(t, tc.inputData)
			gotStart, gotEnd, gotDeleted := extractTimeWindow(reader)
			if !gotStart.Equal(tc.wantStartTime) {
				t.Errorf("extractTimeWindow() gotStart = %v, want %v", gotStart, tc.wantStartTime)
			}
			if !gotEnd.Equal(tc.wantEndTime) {
				t.Errorf("extractTimeWindow() gotEnd = %v, want %v", gotEnd, tc.wantEndTime)
			}
			if gotDeleted != tc.wantDeleted {
				t.Errorf("extractTimeWindow() gotDeleted = %v, want %v", gotDeleted, tc.wantDeleted)
			}
		})
	}
}

func TestExtractResourceBody(t *testing.T) {
	testCases := []struct {
		name      string
		inputData map[string]any
		wantNil   bool
	}{
		{
			name: "returns node when resource data is present",
			inputData: map[string]any{
				"asset": map[string]any{
					"resource": map[string]any{
						"data": map[string]any{
							"kind": "Pod",
						},
					},
				},
			},
			wantNil: false,
		},
		{
			name: "returns nil when resource data is absent",
			inputData: map[string]any{
				"asset": map[string]any{},
			},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := newTestNodeReader(t, tc.inputData)
			got := extractResourceBody(reader)
			if (got == nil) != tc.wantNil {
				t.Errorf("extractResourceBody() nil mismatch: got %v, wantNil %v", got, tc.wantNil)
			}
		})
	}
}
