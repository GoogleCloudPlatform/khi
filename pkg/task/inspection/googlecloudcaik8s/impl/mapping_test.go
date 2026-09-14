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

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConvertTemporalAssetToClusterResourceSnapshot(t *testing.T) {
	startTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	sampleAsset := &assetpb.Asset{
		Name:      "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/default/pods/pod-1",
		AssetType: "k8s.io/Pod",
	}

	testCases := []struct {
		name          string
		input         *assetpb.TemporalAsset
		wantStartTime time.Time
		wantErr       bool
	}{
		{
			name: "temporal asset with start time window",
			input: &assetpb.TemporalAsset{
				Window: &assetpb.TimeWindow{
					StartTime: timestamppb.New(startTime),
				},
				Asset: sampleAsset,
			},
			wantStartTime: startTime,
		},
		{
			name: "temporal asset without window",
			input: &assetpb.TemporalAsset{
				Asset: sampleAsset,
			},
			wantStartTime: time.Time{},
		},
		{
			name: "temporal asset with window but nil start time",
			input: &assetpb.TemporalAsset{
				Window: &assetpb.TimeWindow{},
				Asset:  sampleAsset,
			},
			wantStartTime: time.Time{},
		},
		{
			name:    "nil temporal asset returns error",
			input:   nil,
			wantErr: true,
		},
		{
			name: "nil asset returns error",
			input: &assetpb.TemporalAsset{
				Asset: nil,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ConvertTemporalAssetToClusterResourceSnapshot(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ConvertTemporalAssetToClusterResourceSnapshot() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if diff := cmp.Diff(tc.input, got.TemporalAsset, protocmp.Transform()); diff != "" {
				t.Errorf("TemporalAsset mismatch (-want +got):\n%s", diff)
			}
			if !got.StartTime.Equal(tc.wantStartTime) {
				t.Errorf("StartTime = %v, want %v", got.StartTime, tc.wantStartTime)
			}
		})
	}
}

func TestResolveResourceIdentity(t *testing.T) {
	testCases := []struct {
		name               string
		assetName          string
		assetType          string
		manifestAPIVersion string
		manifestKind       string
		manifestName       string
		manifestNamespace  string
		want               *commonlogk8saudit_contract.ResourceIdentity
	}{
		{
			name:               "namespaced pod with core/v1 normalization",
			assetName:          "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/default/pods/my-pod",
			assetType:          "k8s.io/Pod",
			manifestAPIVersion: "v1",
			manifestKind:       "Pod",
			manifestName:       "my-pod",
			manifestNamespace:  "default",
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Name:       "my-pod",
				Namespace:  "default",
			},
		},
		{
			name:               "deployment with custom group",
			assetName:          "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/prod/apps/deployments/my-deploy",
			assetType:          "apps/Deployment",
			manifestAPIVersion: "apps/v1",
			manifestKind:       "Deployment",
			manifestName:       "my-deploy",
			manifestNamespace:  "prod",
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "apps/v1",
				Kind:       "deployment",
				Name:       "my-deploy",
				Namespace:  "prod",
			},
		},
		{
			name:               "namespace resource forced to empty namespace",
			assetName:          "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/kube-system",
			assetType:          "k8s.io/Namespace",
			manifestAPIVersion: "v1",
			manifestKind:       "Namespace",
			manifestName:       "kube-system",
			manifestNamespace:  "kube-system",
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "namespace",
				Name:       "kube-system",
				Namespace:  "",
			},
		},
		{
			name:               "clusterrole resource cluster-scoped",
			assetName:          "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/rbac.authorization.k8s.io/clusterroles/admin",
			assetType:          "rbac.authorization.k8s.io/ClusterRole",
			manifestAPIVersion: "rbac.authorization.k8s.io/v1",
			manifestKind:       "ClusterRole",
			manifestName:       "admin",
			manifestNamespace:  "",
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "clusterrole",
				Name:       "admin",
				Namespace:  "",
			},
		},
		{
			name:               "fallback to asset name and asset type when manifest fields are missing",
			assetName:          "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/default/pods/fallback-pod",
			assetType:          "k8s.io/Pod",
			manifestAPIVersion: "",
			manifestKind:       "",
			manifestName:       "",
			manifestNamespace:  "",
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Name:       "fallback-pod",
				Namespace:  "default",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveResourceIdentity(
				tc.assetName,
				tc.assetType,
				tc.manifestAPIVersion,
				tc.manifestKind,
				tc.manifestName,
				tc.manifestNamespace,
			)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("resolveResourceIdentity() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseAssetName(t *testing.T) {
	testCases := []struct {
		name          string
		assetName     string
		wantNamespace string
		wantName      string
	}{
		{
			name:          "namespaced resource",
			assetName:     "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/default/pods/my-pod",
			wantNamespace: "default",
			wantName:      "my-pod",
		},
		{
			name:          "cluster-scoped resource",
			assetName:     "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/rbac.authorization.k8s.io/clusterroles/admin",
			wantNamespace: "",
			wantName:      "admin",
		},
		{
			name:          "namespace object itself",
			assetName:     "//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/kube-system",
			wantNamespace: "",
			wantName:      "kube-system",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotNamespace, gotName := parseAssetName(tc.assetName)
			if gotNamespace != tc.wantNamespace {
				t.Errorf("parseAssetName() namespace = %q, want %q", gotNamespace, tc.wantNamespace)
			}
			if gotName != tc.wantName {
				t.Errorf("parseAssetName() name = %q, want %q", gotName, tc.wantName)
			}
		})
	}
}

func TestNormalizeAPIVersion(t *testing.T) {
	testCases := []struct {
		name       string
		apiVersion string
		assetType  string
		want       string
	}{
		{
			name:       "unprefixed core apiVersion",
			apiVersion: "v1",
			assetType:  "k8s.io/Pod",
			want:       "core/v1",
		},
		{
			name:       "already prefixed apiVersion",
			apiVersion: "apps/v1",
			assetType:  "apps/Deployment",
			want:       "apps/v1",
		},
		{
			name:       "empty apiVersion with k8s.io asset type",
			apiVersion: "",
			assetType:  "k8s.io/Pod",
			want:       "core/v1",
		},
		{
			name:       "empty apiVersion with custom group asset type",
			apiVersion: "",
			assetType:  "apps.k8s.io/Deployment",
			want:       "apps.k8s.io/v1",
		},
		{
			name:       "empty apiVersion and empty asset type",
			apiVersion: "",
			assetType:  "",
			want:       "core/v1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeAPIVersion(tc.apiVersion, tc.assetType)
			if got != tc.want {
				t.Errorf("normalizeAPIVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveManifestAPIVersion(t *testing.T) {
	testCases := []struct {
		name      string
		version   string
		assetType string
		want      string
	}{
		{
			name:      "core k8s.io group with v1 version returns v1",
			version:   "v1",
			assetType: "k8s.io/Pod",
			want:      "v1",
		},
		{
			name:      "core k8s.io group with empty version defaults to v1",
			version:   "",
			assetType: "k8s.io/Pod",
			want:      "v1",
		},
		{
			name:      "core group with empty version defaults to v1",
			version:   "",
			assetType: "core/Pod",
			want:      "v1",
		},
		{
			name:      "unprefixed asset type defaults to v1",
			version:   "",
			assetType: "Pod",
			want:      "v1",
		},
		{
			name:      "standard apps group strips k8s.io suffix",
			version:   "v1",
			assetType: "apps.k8s.io/Deployment",
			want:      "apps/v1",
		},
		{
			name:      "standard batch group strips k8s.io suffix",
			version:   "v1",
			assetType: "batch.k8s.io/Job",
			want:      "batch/v1",
		},
		{
			name:      "standard policy group strips k8s.io suffix",
			version:   "v1",
			assetType: "policy.k8s.io/PodDisruptionBudget",
			want:      "policy/v1",
		},
		{
			name:      "standard autoscaling group strips k8s.io suffix",
			version:   "v2",
			assetType: "autoscaling.k8s.io/HorizontalPodAutoscaler",
			want:      "autoscaling/v2",
		},
		{
			name:      "non-standard rbac group retains domain suffix",
			version:   "v1",
			assetType: "rbac.authorization.k8s.io/ClusterRole",
			want:      "rbac.authorization.k8s.io/v1",
		},
		{
			name:      "custom crd group with v1alpha1 version",
			version:   "v1alpha1",
			assetType: "custom.example.com/MyCRD",
			want:      "custom.example.com/v1alpha1",
		},
		{
			name:      "version already contains group prefix returns version as is",
			version:   "apps/v1",
			assetType: "apps.k8s.io/Deployment",
			want:      "apps/v1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveManifestAPIVersion(tc.version, tc.assetType)
			if got != tc.want {
				t.Errorf("resolveManifestAPIVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreManifestTypeMeta(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]any
		wantData map[string]any
	}{
		{
			name: "populates apiVersion and kind for core pod",
			input: map[string]any{
				"asset": map[string]any{
					"assetType": "k8s.io/Pod",
					"resource": map[string]any{
						"version": "v1",
						"data": map[string]any{
							"metadata": map[string]any{"name": "pod-1"},
						},
					},
				},
			},
			wantData: map[string]any{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata":   map[string]any{"name": "pod-1"},
			},
		},
		{
			name: "populates apiVersion and kind for non-core deployment",
			input: map[string]any{
				"asset": map[string]any{
					"assetType": "apps.k8s.io/Deployment",
					"resource": map[string]any{
						"version": "v1",
						"data": map[string]any{
							"metadata": map[string]any{"name": "deploy-1"},
						},
					},
				},
			},
			wantData: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata":   map[string]any{"name": "deploy-1"},
			},
		},
		{
			name: "preserves existing kind and apiVersion",
			input: map[string]any{
				"asset": map[string]any{
					"assetType": "k8s.io/Pod",
					"resource": map[string]any{
						"version": "v1",
						"data": map[string]any{
							"apiVersion": "custom/v2",
							"kind":       "CustomPod",
						},
					},
				},
			},
			wantData: map[string]any{
				"apiVersion": "custom/v2",
				"kind":       "CustomPod",
			},
		},
		{
			name: "handles missing resource gracefully without modifying non-existent data",
			input: map[string]any{
				"asset": map[string]any{
					"assetType": "k8s.io/Pod",
				},
			},
			wantData: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restoreManifestTypeMeta(tc.input)

			var gotData map[string]any
			if assetMap, ok := tc.input["asset"].(map[string]any); ok {
				if resourceMap, ok := assetMap["resource"].(map[string]any); ok {
					gotData, _ = resourceMap["data"].(map[string]any)
				}
			}

			if diff := cmp.Diff(tc.wantData, gotData); diff != "" {
				t.Errorf("data mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRestoreManifestMetadata(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]any
		wantData map[string]any
	}{
		{
			name: "decodes base64 fieldsV1 and expands into JSON object map",
			input: map[string]any{
				"asset": map[string]any{
					"resource": map[string]any{
						"data": map[string]any{
							"metadata": map[string]any{
								"name": "nginx-server",
								"managedFields": []any{
									map[string]any{
										"apiVersion": "v1",
										"fieldsType": "FieldsV1",
										"fieldsV1": map[string]any{
											"Raw": "eyJmOm1ldGFkYXRhIjp7ImY6bGFiZWxzIjp7fX19", // {"f:metadata":{"f:labels":{}}}
										},
										"manager":   "kubectl",
										"operation": "Update",
										"time":      "2026-09-10T05:17:28Z",
									},
								},
							},
						},
					},
				},
			},
			wantData: map[string]any{
				"metadata": map[string]any{
					"name": "nginx-server",
					"managedFields": []any{
						map[string]any{
							"apiVersion": "v1",
							"fieldsType": "FieldsV1",
							"fieldsV1": map[string]any{
								"f:metadata": map[string]any{
									"f:labels": map[string]any{},
								},
							},
							"manager":   "kubectl",
							"operation": "Update",
							"time":      "2026-09-10T05:17:28Z",
						},
					},
				},
			},
		},
		{
			name: "removes empty default fields in metadata and empty subresource",
			input: map[string]any{
				"asset": map[string]any{
					"resource": map[string]any{
						"data": map[string]any{
							"metadata": map[string]any{
								"name":         "nginx-server",
								"clusterName":  "",
								"selfLink":     "",
								"generateName": "",
								"namespace":    "",
								"managedFields": []any{
									map[string]any{
										"manager":     "controller",
										"subresource": "",
									},
									map[string]any{
										"manager":     "kubelet",
										"subresource": "status",
									},
								},
							},
						},
					},
				},
			},
			wantData: map[string]any{
				"metadata": map[string]any{
					"name": "nginx-server",
					"managedFields": []any{
						map[string]any{
							"manager": "controller",
						},
						map[string]any{
							"manager":     "kubelet",
							"subresource": "status",
						},
					},
				},
			},
		},
		{
			name: "preserves non-empty generateName and namespace",
			input: map[string]any{
				"asset": map[string]any{
					"resource": map[string]any{
						"data": map[string]any{
							"metadata": map[string]any{
								"name":         "pod-12345",
								"generateName": "pod-",
								"namespace":    "kube-system",
								"clusterName":  "",
							},
						},
					},
				},
			},
			wantData: map[string]any{
				"metadata": map[string]any{
					"name":         "pod-12345",
					"generateName": "pod-",
					"namespace":    "kube-system",
				},
			},
		},
		{
			name: "handles invalid base64 or non-JSON gracefully without panic",
			input: map[string]any{
				"asset": map[string]any{
					"resource": map[string]any{
						"data": map[string]any{
							"metadata": map[string]any{
								"name": "broken-asset",
								"managedFields": []any{
									map[string]any{
										"fieldsV1": map[string]any{
											"Raw": "not-valid-base64!@#$",
										},
									},
									map[string]any{
										"fieldsV1": map[string]any{
											"Raw": "bm90LWpzb24=", // base64("not-json")
										},
									},
								},
							},
						},
					},
				},
			},
			wantData: map[string]any{
				"metadata": map[string]any{
					"name": "broken-asset",
					"managedFields": []any{
						map[string]any{
							"fieldsV1": map[string]any{
								"Raw": "not-valid-base64!@#$",
							},
						},
						map[string]any{
							"fieldsV1": map[string]any{
								"Raw": "bm90LWpzb24=",
							},
						},
					},
				},
			},
		},
		{
			name: "handles missing metadata gracefully",
			input: map[string]any{
				"asset": map[string]any{
					"resource": map[string]any{
						"data": map[string]any{},
					},
				},
			},
			wantData: map[string]any{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restoreManifestMetadata(tc.input)

			var gotData map[string]any
			if assetMap, ok := tc.input["asset"].(map[string]any); ok {
				if resourceMap, ok := assetMap["resource"].(map[string]any); ok {
					gotData, _ = resourceMap["data"].(map[string]any)
				}
			}

			if diff := cmp.Diff(tc.wantData, gotData); diff != "" {
				t.Errorf("data mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRestoreManagedFieldsEntry(t *testing.T) {
	testCases := []struct {
		name      string
		input     map[string]any
		wantEntry map[string]any
	}{
		{
			name: "decodes base64 fieldsV1 and strips empty subresource",
			input: map[string]any{
				"manager":     "kubectl",
				"operation":   "Update",
				"subresource": "",
				"fieldsV1": map[string]any{
					"Raw": "eyJmOmRhdGEiOnt9fQ==", // {"f:data":{}}
				},
			},
			wantEntry: map[string]any{
				"manager":   "kubectl",
				"operation": "Update",
				"fieldsV1": map[string]any{
					"f:data": map[string]any{},
				},
			},
		},
		{
			name: "preserves non-empty subresource",
			input: map[string]any{
				"manager":     "kubelet",
				"subresource": "status",
			},
			wantEntry: map[string]any{
				"manager":     "kubelet",
				"subresource": "status",
			},
		},
		{
			name: "handles invalid base64 gracefully",
			input: map[string]any{
				"fieldsV1": map[string]any{
					"Raw": "not-valid-base64!@#$",
				},
			},
			wantEntry: map[string]any{
				"fieldsV1": map[string]any{
					"Raw": "not-valid-base64!@#$",
				},
			},
		},
		{
			name: "handles valid base64 with non-JSON payload gracefully",
			input: map[string]any{
				"fieldsV1": map[string]any{
					"Raw": "bm90LWpzb24=", // base64("not-json")
				},
			},
			wantEntry: map[string]any{
				"fieldsV1": map[string]any{
					"Raw": "bm90LWpzb24=",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restoreManagedFieldsEntry(tc.input)
			if diff := cmp.Diff(tc.wantEntry, tc.input); diff != "" {
				t.Errorf("entry mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
