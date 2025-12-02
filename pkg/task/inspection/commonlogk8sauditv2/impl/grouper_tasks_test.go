// Copyright 2025 Google LLC
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

package commonlogk8sauditv2_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type testScanTargetResourceInput struct {
	op           *model.KubernetesObjectOperation
	requestYAML  string
	responseYAML string
}

func TestScanTargetResource(t *testing.T) {
	testCases := []struct {
		desc                       string
		inputs                     []testScanTargetResourceInput
		subresourceDefaultBehavior map[string]subresourceDefaultBehavior
		want                       [][]string
	}{
		{
			desc: "simple non subresource",
			inputs: []testScanTargetResourceInput{
				{
					op: &model.KubernetesObjectOperation{
						APIVersion: "v1",
						PluralKind: "pods",
						Namespace:  "default",
						Name:       "pod1",
						Verb:       enum.RevisionVerbCreate,
					},
					requestYAML:  "",
					responseYAML: "",
				},
			},
			want: [][]string{
				{
					"v1#pod#default#pod1",
				},
			},
		},
		{
			desc: "delete collection on namespace returning pod list",
			inputs: []testScanTargetResourceInput{
				{
					op: &model.KubernetesObjectOperation{
						APIVersion: "v1",
						PluralKind: "pods",
						Namespace:  "default",
						Name:       "",
						Verb:       enum.RevisionVerbDeleteCollection,
					},
					requestYAML: "",
					responseYAML: `items:
  - metadata:
      name: pod1
  - metadata:
      name: pod2`,
				},
			},
			want: [][]string{
				{
					"v1#pod#default#pod1",
					"v1#pod#default#pod2",
				},
			},
		},
		{
			desc: "deleting all resources in a namespace",
			inputs: []testScanTargetResourceInput{
				{
					op: &model.KubernetesObjectOperation{
						APIVersion: "v1",
						PluralKind: "pods",
						Namespace:  "other",
						Name:       "pod-other",
						Verb:       enum.RevisionVerbCreate,
					},
				},
				{
					op: &model.KubernetesObjectOperation{
						APIVersion: "v1",
						PluralKind: "pods",
						Namespace:  "default",
						Name:       "pod1",
						Verb:       enum.RevisionVerbCreate,
					},
				},
				{
					op: &model.KubernetesObjectOperation{
						APIVersion: "v1",
						PluralKind: "pods",
						Namespace:  "default",
						Name:       "pod2",
						Verb:       enum.RevisionVerbCreate,
					},
				},
				{
					op: &model.KubernetesObjectOperation{
						APIVersion: "v1",
						PluralKind: "pods",
						Namespace:  "default",
						Name:       "",
						Verb:       enum.RevisionVerbDeleteCollection,
					},
					responseYAML: ``,
				},
			},
			want: [][]string{
				{"v1#pod#other#pod-other"},
				{"v1#pod#default#pod1"},
				{"v1#pod#default#pod2"},
				{
					"v1#pod#default#pod1",
					"v1#pod#default#pod2",
					"v1#pod#default#@namespace",
				},
			},
		},
		{
			desc: "subresource update returning parent resource",
			inputs: []testScanTargetResourceInput{
				{
					op: &model.KubernetesObjectOperation{
						APIVersion:      "apps/v1",
						PluralKind:      "deployments",
						Namespace:       "default",
						Name:            "deployment1",
						Verb:            enum.RevisionVerbUpdate,
						SubResourceName: "scale",
					},
					responseYAML: `apiVersion: apps/v1
kind: Deployment`,
				},
			},
			want: [][]string{
				{"apps/v1#deployment#default#deployment1"},
			},
		},
		{
			desc: "subresource update returning subresource",
			inputs: []testScanTargetResourceInput{
				{
					op: &model.KubernetesObjectOperation{
						APIVersion:      "v1",
						PluralKind:      "pods",
						Namespace:       "default",
						Name:            "pod1",
						Verb:            enum.RevisionVerbUpdate,
						SubResourceName: "binding",
					},
					responseYAML: `apiVersion: v1
kind: Binding`,
				},
			},
			want: [][]string{
				{"v1#pod#default#pod1#binding"},
			},
		},
		{
			desc: "subresource patch only includes its request",
			inputs: []testScanTargetResourceInput{
				{
					op: &model.KubernetesObjectOperation{
						APIVersion:      "v1",
						PluralKind:      "pods",
						Namespace:       "default",
						Name:            "pod1",
						Verb:            enum.RevisionVerbPatch,
						SubResourceName: "binding",
					},
					requestYAML: `apiVersion: v1
kind: Binding`,
				},
			},
			want: [][]string{
				{"v1#pod#default#pod1#binding"},
			},
		},
		{
			desc: "subresource patch only includes its request with status response",
			inputs: []testScanTargetResourceInput{
				{
					op: &model.KubernetesObjectOperation{
						APIVersion:      "v1",
						PluralKind:      "pods",
						Namespace:       "default",
						Name:            "pod1",
						Verb:            enum.RevisionVerbPatch,
						SubResourceName: "binding",
					},
					responseYAML: `apiVersion: v1
kind: Status`,
					requestYAML: `apiVersion: v1
kind: Binding`,
				},
			},
			want: [][]string{
				{"v1#pod#default#pod1#binding"},
			},
		},
		{
			desc: "cluster scoped resource",
			inputs: []testScanTargetResourceInput{
				{
					op: &model.KubernetesObjectOperation{
						APIVersion: "v1",
						PluralKind: "nodes",
						Name:       "node-1",
						Namespace:  "cluster-scope",
						Verb:       enum.RevisionVerbDelete,
					},
				},
			},
			want: [][]string{
				{"v1#node#cluster-scope#node-1"},
			},
		},
		{
			desc: "request and response are not available & behavior is overriden to Parent",
			inputs: []testScanTargetResourceInput{
				{
					op: &model.KubernetesObjectOperation{
						APIVersion:      "v1",
						PluralKind:      "pods",
						Namespace:       "default",
						Name:            "pod1",
						SubResourceName: "status",
						Verb:            enum.RevisionVerbDelete,
					},
				},
			},
			subresourceDefaultBehavior: map[string]subresourceDefaultBehavior{
				"status": Parent,
			},
			want: [][]string{
				{"v1#pod#default#pod1"},
			},
		},
		{
			desc: "request and response are not available & behavior is overriden to Subresource",
			inputs: []testScanTargetResourceInput{
				{
					op: &model.KubernetesObjectOperation{
						APIVersion:      "v1",
						PluralKind:      "pods",
						Namespace:       "default",
						Name:            "pod1",
						SubResourceName: "binding",
						Verb:            enum.RevisionVerbDelete,
					},
				},
			},
			subresourceDefaultBehavior: map[string]subresourceDefaultBehavior{
				"binding": Subresource,
			},
			want: [][]string{
				{"v1#pod#default#pod1#binding"},
			},
		},
		{
			desc: "request and response are not available & behavior is not overriden",
			inputs: []testScanTargetResourceInput{
				{
					op: &model.KubernetesObjectOperation{
						APIVersion:      "v1",
						PluralKind:      "pods",
						Namespace:       "default",
						Name:            "pod1",
						SubResourceName: "binding",
						Verb:            enum.RevisionVerbDelete,
					},
				},
			},
			want: [][]string{
				{"v1#pod#default#pod1#binding"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			logs := []*log.Log{}
			for _, input := range tc.inputs {
				var request, response *structured.NodeReader
				if input.requestYAML != "" {
					node, err := structured.FromYAML(input.requestYAML)
					if err != nil {
						t.Fatalf("failed to parse request YAML: %v", err)
					}
					request = structured.NewNodeReader(node)
				}
				if input.responseYAML != "" {
					node, err := structured.FromYAML(input.responseYAML)
					if err != nil {
						t.Fatalf("failed to parse response YAML: %v", err)
					}
					response = structured.NewNodeReader(node)
				}
				logs = append(logs, log.NewLogWithFieldSetsForTest(&commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
					K8sOperation: input.op,
					Request:      request,
					Response:     response,
				}))
			}
			var subresourceDefaultBehaviorOverrides map[string]subresourceDefaultBehavior
			if tc.subresourceDefaultBehavior != nil {
				subresourceDefaultBehaviorOverrides = tc.subresourceDefaultBehavior
			}

			targetResourceScanner := targetResourceScanner{
				resourcesByNamespaceKindAPIVersions: map[string]map[string]struct{}{},
				subresourceDefaultBehaviorOverrides: subresourceDefaultBehaviorOverrides,
			}
			got := [][]string{}
			for _, l := range logs {
				gotOperations := targetResourceScanner.scanTargetResource(l)
				gotPaths := []string{}
				for _, op := range gotOperations {
					gotPaths = append(gotPaths, op.ResourcePath())
				}
				got = append(got, gotPaths)
			}

			if diff := cmp.Diff(got, tc.want, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Errorf("mismatch (-want +got): %s", diff)
			}
		})
	}
}
