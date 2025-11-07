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

package googlecloudlogk8scontrolplane_contract

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestK8sControlplaneComponentFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *K8sControlplaneComponentFieldSet
	}{
		{
			desc: "simple log entry",
			input: `
resource:
  labels:
    cluster_name: test-cluster
    component_name: "kube-apiserver"
`,
			want: &K8sControlplaneComponentFieldSet{
				ClusterName:   "test-cluster",
				ComponentName: "kube-apiserver",
			},
		},
		{
			desc: "without component name",
			input: `
resource:
  labels:
    foo: bar
`,
			want: &K8sControlplaneComponentFieldSet{
				ClusterName:   "unknown",
				ComponentName: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse test input YAML: %v", err)
			}
			err = l.SetFieldSetReader(&K8sControlplaneComponentFieldSetReader{})
			if err != nil {
				t.Errorf("failed to set fieldset reader: %v", err)
			}

			gotFieldSet := log.MustGetFieldSet(l, &K8sControlplaneComponentFieldSet{})
			if diff := cmp.Diff(tc.want, gotFieldSet); diff != "" {
				t.Errorf("K8sControlplaneComponentFieldSet mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sControlplaneCommonMessageFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *K8sControlplaneCommonMessageFieldSet
	}{
		{
			desc: "simple log entry",
			input: `
jsonPayload:
  message: "test message"
`,
			want: &K8sControlplaneCommonMessageFieldSet{
				Message: "test message",
			},
		},
		{
			desc:  "without message",
			input: `{}`,
			want: &K8sControlplaneCommonMessageFieldSet{
				Message: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse test input YAML: %v", err)
			}
			err = l.SetFieldSetReader(&K8sControlplaneCommonMessageFieldSetReader{})
			if err != nil {
				t.Errorf("failed to set fieldset reader: %v", err)
			}

			gotFieldSet := log.MustGetFieldSet(l, &K8sControlplaneCommonMessageFieldSet{})
			if diff := cmp.Diff(tc.want, gotFieldSet); diff != "" {
				t.Errorf("K8sControlplaneCommonMessageFieldSet mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sSchedulerComponentFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *K8sSchedulerComponentFieldSet
	}{
		{
			desc: "simple scheduler entry",
			input: `
jsonPayload:
  message: '"Attempting to schedule pod" pod="foo/bar"'
`,
			want: &K8sSchedulerComponentFieldSet{
				PodName:      "bar",
				PodNamespace: "foo",
			},
		},
		{
			desc:  "without message",
			input: `{}`,
			want: &K8sSchedulerComponentFieldSet{
				PodName:      "",
				PodNamespace: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse test input YAML: %v", err)
			}
			err = l.SetFieldSetReader(&K8sSchedulerComponentFieldSetReader{})
			if err != nil {
				t.Errorf("failed to set fieldset reader: %v", err)
			}

			gotFieldSet := log.MustGetFieldSet(l, &K8sSchedulerComponentFieldSet{})
			if diff := cmp.Diff(tc.want, gotFieldSet); diff != "" {
				t.Errorf("K8sSchedulerComponentFieldSet mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sControllerManagerComponentFieldSetReader_ReadController(t *testing.T) {
	testCases := []struct {
		desc        string
		input       string
		inputSource string
		want        string
	}{
		{
			desc:        "with logger field",
			input:       `"Finished syncing" logger="my-controller"`,
			inputSource: "namespace_controller.go",
			want:        "my-controller",
		},
		{
			desc:        "with controller field",
			input:       `"Finished syncing" controller="another-controller"`,
			inputSource: "namespace_controller.go",
			want:        "another-controller",
		},
		{
			desc:        "with sourceLocation mapping",
			input:       `"Finished syncing"`,
			inputSource: "namespace_controller.go",
			want:        "namespace-controller",
		},
		{
			desc:        "without any identifiable field",
			input:       `"Finished syncing"`,
			inputSource: "unknown.go",
			want:        "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			reader := &K8sControllerManagerComponentFieldSetReader{
				WellKnownSourceLocationToControllerMap: map[string]string{
					"namespace_controller.go": "namespace-controller",
				},
			}
			controller, err := reader.readController(tc.input, tc.inputSource)
			if err != nil {
				t.Errorf("readController() returned an unexpected error: %v", err)
			}
			if controller != tc.want {
				t.Errorf("readController() got = %q, want %q", controller, tc.want)
			}
		})
	}

}

func TestK8sControllerManagerComponentFieldSetReader_ReadResourceAssociationFromKindField(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  []resourcepath.ResourcePath
	}{
		{
			desc:  "with kind and namespaced key",
			input: `"Finished syncing" kind="Pod" key="default/my-pod"`,
			want: []resourcepath.ResourcePath{
				resourcepath.NameLayerGeneralItem("core/v1", "pod", "default", "my-pod"),
			},
		},
		{
			desc:  "with kind and cluster-scoped key",
			input: `"Finished syncing" kind="Node" key="my-node"`,
			want: []resourcepath.ResourcePath{
				resourcepath.NameLayerGeneralItem("core/v1", "node", "cluster-scope", "my-node"),
			},
		},
		{
			desc:  "with kind but malformed key",
			input: `"Finished syncing" kind="Pod" key="malformed-key"`,
			want:  nil,
		},
		{
			desc:  "with kind but no key",
			input: `"Finished syncing" kind="Pod"`,
			want:  nil,
		},
		{
			desc:  "without kind",
			input: `"Finished syncing" key="default/my-pod"`,
			want:  nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {

			reader := &K8sControllerManagerComponentFieldSetReader{
				WellKnownKindToKLogFieldPairs: []*KindToKLogFieldPairData{
					{
						APIVersion:   "core/v1",
						KindName:     "node",
						KLogField:    "node",
						IsNamespaced: false,
					},
					{
						APIVersion:   "core/v1",
						KindName:     "pod",
						KLogField:    "pod",
						IsNamespaced: true,
					},
				},
			}

			paths := reader.readResourceAssociationFromKindField(tc.input)
			if diff := cmp.Diff(tc.want, paths); diff != "" {
				t.Errorf("readResourceAssociationFromKindField() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sControllerManagerComponentFieldSetReader_ReadResourceAssociationFromControllerSpecificField(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  []resourcepath.ResourcePath
	}{
		{
			desc:  "with multiple resources",
			input: `"Finished syncing" pod="default/my-job" node="node-foo"`,
			want: []resourcepath.ResourcePath{
				resourcepath.NameLayerGeneralItem("core/v1", "pod", "default", "my-job"),
				resourcepath.NameLayerGeneralItem("core/v1", "node", "cluster-scope", "node-foo"),
			},
		},
		{
			desc:  "with single resource",
			input: `"Finished syncing" pod="default/my-job"`,
			want: []resourcepath.ResourcePath{
				resourcepath.NameLayerGeneralItem("core/v1", "pod", "default", "my-job"),
			},
		},
		{
			desc:  "without resource",
			input: `"Finished syncing"`,
			want:  nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			reader := &K8sControllerManagerComponentFieldSetReader{
				WellKnownKindToKLogFieldPairs: []*KindToKLogFieldPairData{
					{
						APIVersion:   "core/v1",
						KindName:     "node",
						KLogField:    "node",
						IsNamespaced: false,
					},
					{
						APIVersion:   "core/v1",
						KindName:     "pod",
						KLogField:    "pod",
						IsNamespaced: true,
					},
				},
			}

			paths := reader.readResourceAssociationFromControllerSpecificField(tc.input)
			if diff := cmp.Diff(tc.want, paths, cmpopts.SortSlices(func(a, b resourcepath.ResourcePath) int { return strings.Compare(a.Path, b.Path) })); diff != "" {
				t.Errorf("readResourceAssociationFromControllerSpecificField() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sControllerManagerComponentFieldSetReader_ReadResourceAssociationFromItems(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  resourcepath.ResourcePath
	}{
		{
			desc:  "valid item field - namespaced",
			input: `"Deleting item" logger="garbage-collector-controller" item="[coordination.k8s.io/v1/Lease, namespace: kube-node-lease, name: gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v, uid: 8aba20bf-0392-40c9-ae35-240b7c099523]" propagationPolicy="Background"`,
			want:  resourcepath.NameLayerGeneralItem("coordination.k8s.io/v1", "lease", "kube-node-lease", "gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v"),
		},
		{
			desc:  "valid item field - cluster-scoped",
			input: `"Deleting item" logger="garbage-collector-controller" item="[rbac.authorization.k8s.io/v1/ClusterRole, namespace: , name: admin, uid: 8aba20bf-0392-40c9-ae35-240b7c099523]" propagationPolicy="Background"`,
			want:  resourcepath.NameLayerGeneralItem("rbac.authorization.k8s.io/v1", "clusterrole", "cluster-scope", "admin"),
		},
		{
			desc:  "valid item field - in core api version",
			input: `"Deleting item" logger="garbage-collector-controller" item="[v1/Pod, namespace: kube-system, name: gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v, uid: 8aba20bf-0392-40c9-ae35-240b7c099523]" propagationPolicy="Background"`,
			want:  resourcepath.NameLayerGeneralItem("core/v1", "pod", "kube-system", "gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v"),
		},
		{
			desc:  "item field missing",
			input: `"Deleting item" logger="garbage-collector-controller" propagationPolicy="Background"`,
			want:  resourcepath.ResourcePath{},
		},
		{
			desc:  "item field malformed",
			input: `"Deleting item" logger="garbage-collector-controller" item="malformed-item" propagationPolicy="Background"`,
			want:  resourcepath.ResourcePath{},
		},
		{
			desc:  "item field malformed - no slash contained in apiVersion",
			input: `"Deleting item" logger="garbage-collector-controller" item="[Pod, namespace: kube-system, name: gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v, uid: 8aba20bf-0392-40c9-ae35-240b7c099523]" propagationPolicy="Background"`,
			want:  resourcepath.ResourcePath{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			reader := &K8sControllerManagerComponentFieldSetReader{}

			path := reader.readResourceAssociationFromItems(tc.input)

			if diff := cmp.Diff(tc.want, path); diff != "" {
				t.Errorf("readResourceAssociationFromItems() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
