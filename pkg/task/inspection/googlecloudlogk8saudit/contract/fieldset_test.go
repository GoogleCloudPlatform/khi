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

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
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
