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

package caik8s_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
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

func TestExtractK8sIdentity(t *testing.T) {
	testCases := []struct {
		name      string
		inputData map[string]any
		want      *k8saudit.ResourceIdentity
		wantOK    bool
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
			want: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Name:       "pod-1",
				Namespace:  "default",
			},
			wantOK: true,
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
			want: &k8saudit.ResourceIdentity{
				APIVersion: "apps/v1",
				Kind:       "deployment",
				Name:       "web",
				Namespace:  "prod",
			},
			wantOK: true,
		},
		{
			name: "falls back to asset name and asset type when manifest data is missing",
			inputData: map[string]any{
				"asset": map[string]any{
					"name":      "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/custom-ns/services/my-svc",
					"assetType": "k8s.io/Service",
				},
			},
			want: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "service",
				Name:       "my-svc",
				Namespace:  "custom-ns",
			},
			wantOK: true,
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
			want: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "node",
				Name:       "node-1",
				Namespace:  "",
			},
			wantOK: true,
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
			want: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "namespace",
				Name:       "kube-system",
				Namespace:  "",
			},
			wantOK: true,
		},
		{
			name: "returns false when kind or name is empty",
			inputData: map[string]any{
				"asset": map[string]any{},
			},
			want: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "",
				Name:       "",
				Namespace:  "",
			},
			wantOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := newTestNodeReader(t, tc.inputData)
			got, ok := extractK8sIdentity(reader)
			if ok != tc.wantOK {
				t.Errorf("extractK8sIdentity() ok = %v, want %v", ok, tc.wantOK)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("extractK8sIdentity() mismatch (-want +got):\n%s", diff)
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
			gotK8s := extractK8sResourceBody(reader)
			if (gotK8s == nil) != tc.wantNil {
				t.Errorf("extractK8sResourceBody() nil mismatch: got %v, wantNil %v", gotK8s, tc.wantNil)
			}
			gotGKE := extractGKEResourceBody(reader)
			if (gotGKE == nil) != tc.wantNil {
				t.Errorf("extractGKEResourceBody() nil mismatch: got %v, wantNil %v", gotGKE, tc.wantNil)
			}
		})
	}
}
