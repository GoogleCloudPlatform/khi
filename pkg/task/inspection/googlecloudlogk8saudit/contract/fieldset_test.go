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
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	"github.com/google/go-cmp/cmp"
)

func TestGCPK8sAuditLogFieldSetReader_Truncated(t *testing.T) {
	node, err := structured.FromGoValue(map[string]any{
		"labels": map[string]any{
			"audit.k8s.io/truncated": "true",
		},
		"protoPayload": map[string]any{
			"methodName":   "io.k8s.core.v1.pods.update",
			"resourceName": "core/v1/namespaces/default/pods/nginx",
		},
	}, &structured.AlphabeticalGoMapKeyOrderProvider{})
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	got, err := (&GCPK8sAuditLogFieldSetReader{}).Read(structured.NewNodeReader(node))
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	fieldSet := got.(*commonlogk8saudit_contract.K8sAuditLogFieldSet)
	if !fieldSet.Truncated {
		t.Fatalf("Truncated = false, want true")
	}
}

func TestParseKubernetesOperation(t *testing.T) {
	testCases := []struct {
		desc         string
		resourceName string
		methodName   string
		want         *model.KubernetesObjectOperation
	}{
		{
			desc:         "simple case",
			resourceName: "core/v1/nodes/gke-p0-gke-basic-1-default-6400229f-n02c/status",
			methodName:   "io.k8s.core.v1.nodes.status.patch",
			want: &model.KubernetesObjectOperation{
				APIVersion:      "core/v1",
				PluralKind:      "nodes",
				Namespace:       "cluster-scope",
				Name:            "gke-p0-gke-basic-1-default-6400229f-n02c",
				SubResourceName: "status",
				Verb:            enum.RevisionVerbPatch,
			},
		},
		{
			desc:         "namespaced resource",
			resourceName: "core/v1/namespaces/default/pods/nginx",
			methodName:   "io.k8s.core.v1.pods.create",
			want: &model.KubernetesObjectOperation{
				APIVersion:      "core/v1",
				PluralKind:      "pods",
				Namespace:       "default",
				Name:            "nginx",
				SubResourceName: "",
				Verb:            enum.RevisionVerbCreate,
			},
		},
		{
			desc:         "cluster scoped resource",
			resourceName: "core/v1/namespaces/test-ns",
			methodName:   "io.k8s.core.v1.namespaces.delete",
			want: &model.KubernetesObjectOperation{
				APIVersion:      "core/v1",
				PluralKind:      "namespaces",
				Namespace:       "cluster-scope",
				Name:            "test-ns",
				SubResourceName: "",
				Verb:            enum.RevisionVerbDelete,
			},
		},
		{
			desc:         "namespaced resource with subresource",
			resourceName: "apps/v1/namespaces/kube-system/deployments/coredns/scale",
			methodName:   "io.k8s.apps.v1.deployments.scale.update",
			want: &model.KubernetesObjectOperation{
				APIVersion:      "apps/v1",
				PluralKind:      "deployments",
				Namespace:       "kube-system",
				Name:            "coredns",
				SubResourceName: "scale",
				Verb:            enum.RevisionVerbUpdate,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := parseKubernetesOperation(tc.resourceName, tc.methodName)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseKubernetesOperation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
