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

package googlecloudlogk8saudit_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestParseKubernetesOperation(t *testing.T) {
	testCases := []struct {
		desc                string
		resourceName        string
		methodName          string
		wantAPIVersion      string
		wantPluralKind      string
		wantNamespace       string
		wantName            string
		wantSubResourceName string
		wantVerb            *pb.Verb
	}{
		{
			desc:                "simple case",
			resourceName:        "core/v1/nodes/gke-p0-gke-basic-1-default-6400229f-n02c/status",
			methodName:          "io.k8s.core.v1.nodes.status.patch",
			wantAPIVersion:      "core/v1",
			wantPluralKind:      "nodes",
			wantNamespace:       "cluster-scope",
			wantName:            "gke-p0-gke-basic-1-default-6400229f-n02c",
			wantSubResourceName: "status",
			wantVerb:            commonlogk8saudit_contract.VerbPatch,
		},
		{
			desc:                "namespaced resource",
			resourceName:        "core/v1/namespaces/default/pods/nginx",
			methodName:          "io.k8s.core.v1.pods.create",
			wantAPIVersion:      "core/v1",
			wantPluralKind:      "pods",
			wantNamespace:       "default",
			wantName:            "nginx",
			wantSubResourceName: "",
			wantVerb:            commonlogk8saudit_contract.VerbCreate,
		},
		{
			desc:                "cluster scoped resource",
			resourceName:        "core/v1/namespaces/test-ns",
			methodName:          "io.k8s.core.v1.namespaces.delete",
			wantAPIVersion:      "core/v1",
			wantPluralKind:      "namespaces",
			wantNamespace:       "cluster-scope",
			wantName:            "test-ns",
			wantSubResourceName: "",
			wantVerb:            commonlogk8saudit_contract.VerbDelete,
		},
		{
			desc:                "namespaced resource with subresource",
			resourceName:        "apps/v1/namespaces/kube-system/deployments/coredns/scale",
			methodName:          "io.k8s.apps.v1.deployments.scale.update",
			wantAPIVersion:      "apps/v1",
			wantPluralKind:      "deployments",
			wantNamespace:       "kube-system",
			wantName:            "coredns",
			wantSubResourceName: "scale",
			wantVerb:            commonlogk8saudit_contract.VerbUpdate,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			apiVersion, pluralKind, namespace, name, subResourceName, verb := parseKubernetesOperation(tc.resourceName, tc.methodName)
			if apiVersion != tc.wantAPIVersion {
				t.Errorf("apiVersion mismatch: want %q, got %q", tc.wantAPIVersion, apiVersion)
			}
			if pluralKind != tc.wantPluralKind {
				t.Errorf("pluralKind mismatch: want %q, got %q", tc.wantPluralKind, pluralKind)
			}
			if namespace != tc.wantNamespace {
				t.Errorf("namespace mismatch: want %q, got %q", tc.wantNamespace, namespace)
			}
			if name != tc.wantName {
				t.Errorf("name mismatch: want %q, got %q", tc.wantName, name)
			}
			if subResourceName != tc.wantSubResourceName {
				t.Errorf("subResourceName mismatch: want %q, got %q", tc.wantSubResourceName, subResourceName)
			}
			if verb != tc.wantVerb {
				t.Errorf("verb mismatch: want %v, got %v", tc.wantVerb, verb)
			}
		})
	}
}

func TestGCPK8sAuditLogFieldSetReader_Read(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  *commonlogk8saudit_contract.K8sAuditLogFieldSet
	}{
		{
			name: "basic fields",
			input: `{
				"operation": {
					"id": "test-op-1",
					"first": true,
					"last": false
				},
				"protoPayload": {
					"resourceName": "core/v1/namespaces/default/pods/nginx",
					"methodName": "io.k8s.core.v1.pods.create",
					"authenticationInfo": {
						"principalEmail": "admin@example.com"
					},
					"status": {
						"code": 0
					}
				}
			}`,
			want: &commonlogk8saudit_contract.K8sAuditLogFieldSet{
				OperationID:  "test-op-1",
				IsFirst:      true,
				IsLast:       false,
				RequestURI:   "core/v1/namespaces/default/pods/nginx",
				APIVersion:   "core/v1",
				PluralKind:   "pods",
				Namespace:    "default",
				ResourceName: "nginx",
				ClusterName:  "unknown",
				Verb:         commonlogk8saudit_contract.VerbCreate,
				Principal:    "admin@example.com",
				StatusCode:   0,
				IsError:      false,
			},
		},
		{
			name: "with mutating webhook annotations",
			input: `{
				"labels": {
					"mutation.webhook.admission.k8s.io/round_0_index_0": "{\"configuration\":\"my-config\",\"webhook\":\"my-webhook\",\"mutated\":true}",
					"patch.webhook.admission.k8s.io/round_0_index_0": "{\"configuration\":\"my-config\",\"webhook\":\"my-webhook\",\"patch\":[{\"op\":\"add\",\"path\":\"/metadata/annotations/my-annotation\",\"value\":\"foo\"}],\"patchType\":\"JSONPatch\"}",
					"mutation.webhook.admission.k8s.io/round_1_index_0": "{\"configuration\":\"my-config-2\",\"webhook\":\"my-webhook-2\",\"mutated\":false}",
					"failed-open.mutation.webhook.admission.k8s.io/round_2_index_0": "my-webhook-3"
				},
				"protoPayload": {
					"resourceName": "core/v1/namespaces/default/pods/nginx",
					"methodName": "io.k8s.core.v1.pods.create"
				}
			}`,
			want: &commonlogk8saudit_contract.K8sAuditLogFieldSet{
				RequestURI:   "core/v1/namespaces/default/pods/nginx",
				APIVersion:   "core/v1",
				PluralKind:   "pods",
				Namespace:    "default",
				ResourceName: "nginx",
				ClusterName:  "unknown",
				Verb:         commonlogk8saudit_contract.VerbCreate,
				MutatingWebhookResults: []*commonlogk8saudit_contract.MutatingWebhookResult{
					{
						Round:         0,
						Index:         0,
						Configuration: "my-config",
						Webhook:       "my-webhook",
						Mutated:       true,
						Patch: []commonlogk8saudit_contract.MutatingWebhookPatch{
							{
								Op:    "add",
								Path:  "/metadata/annotations/my-annotation",
								Value: "foo",
							},
						},
						FailedOpen: false,
					},
					{
						Round:         1,
						Index:         0,
						Configuration: "my-config-2",
						Webhook:       "my-webhook-2",
						Mutated:       false,
						Patch:         nil,
						FailedOpen:    false,
					},
					{
						Round:         2,
						Index:         0,
						Configuration: "",
						Webhook:       "",
						Mutated:       false,
						Patch:         nil,
						FailedOpen:    true,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := structured.FromYAML(tc.input)
			if err != nil {
				t.Fatalf("failed to parse test input: %v", err)
			}
			reader := &GCPK8sAuditLogFieldSetReader{}
			got, err := reader.Read(structured.NewNodeReader(node))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotFieldSet, ok := got.(*commonlogk8saudit_contract.K8sAuditLogFieldSet)
			if !ok {
				t.Fatalf("returned field set is not *K8sAuditLogFieldSet")
			}

			// Clear uncomparable fields
			gotFieldSet.Request = nil
			gotFieldSet.Response = nil

			if diff := cmp.Diff(tc.want, gotFieldSet, cmpopts.SortSlices(func(a, b *commonlogk8saudit_contract.MutatingWebhookResult) bool {
				if a.Round != b.Round {
					return a.Round < b.Round
				}
				return a.Index < b.Index
			}), protocmp.Transform()); diff != "" {
				t.Errorf("Read() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
