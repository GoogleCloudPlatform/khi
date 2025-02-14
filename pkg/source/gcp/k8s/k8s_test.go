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

package k8s

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestParseKubernetesOperation(t *testing.T) {
	testCases := []struct {
		ResourceName  string
		MethodName    string
		ExpectedK8sOp *model.KubernetesObjectOperation
	}{
		{
			ResourceName: "io.k8s.core/v1/namespaces/foo/pods/bar/status",
			MethodName:   "io.k8s.core.v1.pods.status.update",
			ExpectedK8sOp: &model.KubernetesObjectOperation{
				APIVersion:      "io.k8s.core/v1",
				Namespace:       "foo",
				Name:            "bar",
				PluralKind:      "pods",
				SubResourceName: "status",
				Verb:            enum.RevisionVerbUpdate,
			},
		},
		{
			ResourceName: "io.k8s.core/v1/nodes/foo",
			MethodName:   "io.k8s.core.v1.nodes.delete",
			ExpectedK8sOp: &model.KubernetesObjectOperation{
				APIVersion:      "io.k8s.core/v1",
				Namespace:       "Cluster-Scope",
				Name:            "foo",
				PluralKind:      "nodes",
				SubResourceName: "",
				Verb:            enum.RevisionVerbDelete,
			},
		}, {
			ResourceName: "apps/v1/namespaces/knative-serving/deployments",
			MethodName:   "io.k8s.apps.v1.deployments.deletecollection",
			ExpectedK8sOp: &model.KubernetesObjectOperation{
				APIVersion:      "apps/v1",
				Namespace:       "knative-serving",
				Name:            "",
				PluralKind:      "deployments",
				SubResourceName: "",
				Verb:            enum.RevisionVerbDeleteCollection,
			},
		},
		{
			ResourceName: "core/v1/namespaces/001-jobs/finalize",
			MethodName:   "io.k8s.core.v1.namespaces.finalize.update",
			ExpectedK8sOp: &model.KubernetesObjectOperation{
				APIVersion:      "core/v1",
				Namespace:       "Cluster-Scope",
				Name:            "001-jobs",
				PluralKind:      "namespaces",
				SubResourceName: "finalize",
				Verb:            enum.RevisionVerbUpdate,
			},
		},
		{
			ResourceName: "core/v1/namespaces/001-jobs",
			MethodName:   "io.k8s.core.v1.namespaces.create",
			ExpectedK8sOp: &model.KubernetesObjectOperation{
				APIVersion:      "core/v1",
				Namespace:       "Cluster-Scope",
				Name:            "001-jobs",
				PluralKind:      "namespaces",
				SubResourceName: "",
				Verb:            enum.RevisionVerbCreate,
			},
		},
		{
			ResourceName: "core/v1/namespaces/003-disks/pods",
			MethodName:   "io.k8s.core.v1.pods.create",
			ExpectedK8sOp: &model.KubernetesObjectOperation{
				APIVersion:      "core/v1",
				Namespace:       "003-disks",
				Name:            "unknown",
				PluralKind:      "pods",
				SubResourceName: "",
				Verb:            enum.RevisionVerbCreate,
			},
		},
		{
			ResourceName: "core/v1/namespaces/003-disks/pods",
			MethodName:   "io.k8s.core.v1.pods.patch",
			ExpectedK8sOp: &model.KubernetesObjectOperation{
				APIVersion:      "core/v1",
				Namespace:       "003-disks",
				Name:            "unknown",
				PluralKind:      "pods",
				SubResourceName: "",
				Verb:            enum.RevisionVerbPatch,
			},
		},
		{
			ResourceName: "core/v1/namespaces/003-disks/pods",
			MethodName:   "io.k8s.core.v1.pods.watch",
			ExpectedK8sOp: &model.KubernetesObjectOperation{
				APIVersion:      "core/v1",
				Namespace:       "003-disks",
				Name:            "unknown",
				PluralKind:      "pods",
				SubResourceName: "",
				Verb:            enum.RevisionVerbUnknown,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%s-%s", testCase.ResourceName, testCase.MethodName), func(t *testing.T) {
			res := ParseKubernetesOperation(testCase.ResourceName, testCase.MethodName)
			if diff := cmp.Diff(res, testCase.ExpectedK8sOp); diff != "" {
				t.Errorf("result operation is not matching with the expected operation\n%s", diff)
			}
		})
	}
}

func TestConvertToResourcePath(t *testing.T) {
	res := ParseKubernetesOperation("io.k8s.core/v1/namespaces/foo/pods/bar/status", "io.k8s.core.v1.pods.status.update")
	if res.CovertToResourcePath() != "io.k8s.core/v1#pod#foo#bar#status" {
		t.Errorf("Expected resource path mismatch, got %q want 'io.k8s.core/v1#pod#foo#bar#status'", res.CovertToResourcePath())
	}

	res = ParseKubernetesOperation("io.k8s.core/v1/namespaces/foo/pods/bar", "io.k8s.core.v1.pods.update")
	if res.CovertToResourcePath() != "io.k8s.core/v1#pod#foo#bar" {
		t.Errorf("EExpected resource path mismatch, got %q want 'io.k8s.core/v1#pod#foo#bar'", res.CovertToResourcePath())
	}
}
