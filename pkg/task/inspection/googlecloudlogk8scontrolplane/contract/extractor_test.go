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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestExtractK8sControlplaneComponent(t *testing.T) {
	testCases := []struct {
		desc      string
		inputYAML string
		mock      *K8sControlplaneComponentFieldSet
		want      K8sControlplaneComponentFieldSet
	}{
		{
			desc: "simple log entry",
			inputYAML: `
resource:
  labels:
    cluster_name: test-cluster
    component_name: "kube-apiserver"
`,
			want: K8sControlplaneComponentFieldSet{
				ProjectID:     "unknown",
				ClusterName:   "test-cluster",
				ComponentName: "kube-apiserver",
			},
		},
		{
			desc: "without component name",
			inputYAML: `
resource:
  labels:
    foo: bar
`,
			want: K8sControlplaneComponentFieldSet{
				ProjectID:     "unknown",
				ClusterName:   "unknown",
				ComponentName: "",
			},
		},
		{
			desc: "from mock",
			mock: &K8sControlplaneComponentFieldSet{
				ProjectID:     "mock-project",
				ClusterName:   "mock-cluster",
				ComponentName: "mock-component",
			},
			want: K8sControlplaneComponentFieldSet{
				ProjectID:     "mock-project",
				ClusterName:   "mock-cluster",
				ComponentName: "mock-component",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var l *log.Log
			var err error
			if tc.mock != nil {
				l = testlog.NewMockLog(*tc.mock)
			} else {
				l, err = log.NewLogFromYAMLString(tc.inputYAML)
				if err != nil {
					t.Fatalf("failed to parse test input YAML: %v", err)
				}
			}

			got, err := ExtractK8sControlplaneComponent(l.NodeReader)
			if err != nil {
				t.Fatalf("ExtractK8sControlplaneComponent() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractK8sControlplaneComponent() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractK8sControlplaneCommonMessage(t *testing.T) {
	testCases := []struct {
		desc      string
		inputYAML string
		mock      *K8sControlplaneCommonMessageFieldSet
		want      string
	}{
		{
			desc: "simple log entry",
			inputYAML: `
jsonPayload:
  message: "test message"
`,
			want: "test message",
		},
		{
			desc:      "without message",
			inputYAML: `{}`,
			want:      "",
		},
		{
			desc: "from mock",
			mock: &K8sControlplaneCommonMessageFieldSet{
				Message: "mock message",
			},
			want: "mock message",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var l *log.Log
			var err error
			if tc.mock != nil {
				l = testlog.NewMockLog(*tc.mock)
			} else {
				l, err = log.NewLogFromYAMLString(tc.inputYAML)
				if err != nil {
					t.Fatalf("failed to parse test input YAML: %v", err)
				}
			}

			got, err := ExtractK8sControlplaneCommonMessage(l.NodeReader)
			if err != nil {
				t.Fatalf("ExtractK8sControlplaneCommonMessage() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractK8sControlplaneCommonMessage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractK8sSchedulerComponent(t *testing.T) {
	testCases := []struct {
		desc      string
		inputYAML string
		mock      *K8sSchedulerComponentFieldSet
		want      K8sSchedulerComponentFieldSet
	}{
		{
			desc: "simple scheduler entry",
			inputYAML: `
jsonPayload:
  message: '"Attempting to schedule pod" pod="foo/bar"'
`,
			want: K8sSchedulerComponentFieldSet{
				PodName:      "bar",
				PodNamespace: "foo",
			},
		},
		{
			desc:      "without message",
			inputYAML: `{}`,
			want: K8sSchedulerComponentFieldSet{
				PodName:      "",
				PodNamespace: "",
			},
		},
		{
			desc: "from mock",
			mock: &K8sSchedulerComponentFieldSet{
				PodName:      "mock-pod",
				PodNamespace: "mock-ns",
			},
			want: K8sSchedulerComponentFieldSet{
				PodName:      "mock-pod",
				PodNamespace: "mock-ns",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var l *log.Log
			var err error
			if tc.mock != nil {
				l = testlog.NewMockLog(*tc.mock)
			} else {
				l, err = log.NewLogFromYAMLString(tc.inputYAML)
				if err != nil {
					t.Fatalf("failed to parse test input YAML: %v", err)
				}
			}

			got, err := ExtractK8sSchedulerComponent(l.NodeReader, nil)
			if err != nil {
				t.Fatalf("ExtractK8sSchedulerComponent() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractK8sSchedulerComponent() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sControllerManagerComponentExtractor_ReadController(t *testing.T) {
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
			extractor := &K8sControllerManagerComponentExtractor{
				WellKnownSourceLocationToControllerMap: map[string]string{
					"namespace_controller.go": "namespace-controller",
				},
			}
			klogParser := logutil.NewKLogTextParser(false)
			controller, err := extractor.ReadController(klogParser.TryParse(tc.input), tc.inputSource)
			if err != nil {
				t.Errorf("ReadController() returned an unexpected error: %v", err)
			}
			if controller != tc.want {
				t.Errorf("ReadController() got = %q, want %q", controller, tc.want)
			}
		})
	}
}

func TestK8sControllerManagerComponentExtractor_ReadResourceAssociationFromKindField(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  []*commonlogk8saudit_contract.ResourceIdentity
	}{
		{
			desc:  "with kind and namespaced key",
			input: `"Finished syncing" kind="Pod" key="default/my-pod"`,
			want: []*commonlogk8saudit_contract.ResourceIdentity{
				{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "my-pod",
				},
			},
		},
		{
			desc:  "with kind and cluster-scoped key",
			input: `"Finished syncing" kind="Node" key="my-node"`,
			want: []*commonlogk8saudit_contract.ResourceIdentity{
				{
					APIVersion: "core/v1",
					Kind:       "node",
					Namespace:  "cluster-scope",
					Name:       "my-node",
				},
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
			extractor := &K8sControllerManagerComponentExtractor{
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
			klogParser := logutil.NewKLogTextParser(false)
			paths := extractor.ReadResourceAssociationFromKindField(klogParser.TryParse(tc.input))
			if diff := cmp.Diff(tc.want, paths); diff != "" {
				t.Errorf("ReadResourceAssociationFromKindField() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sControllerManagerComponentExtractor_ReadResourceAssociationFromControllerSpecificField(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  []*commonlogk8saudit_contract.ResourceIdentity
	}{
		{
			desc:  "with multiple resources",
			input: `"Finished syncing" pod="default/my-job" node="node-foo"`,
			want: []*commonlogk8saudit_contract.ResourceIdentity{
				{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "my-job",
				},
				{
					APIVersion: "core/v1",
					Kind:       "node",
					Namespace:  "cluster-scope",
					Name:       "node-foo",
				},
			},
		},
		{
			desc:  "with single resource",
			input: `"Finished syncing" pod="default/my-job"`,
			want: []*commonlogk8saudit_contract.ResourceIdentity{
				{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "my-job",
				},
			},
		},
		{
			desc:  "with kind and cluster-scoped key and longer name",
			input: `"attacherDetacher.DetachVolume started" logger="persistentvolume-attach-detach-controller" node="node-foo" volumeName="kubernetes.io/csi/pd.csi.storage.gke.io^projects/UNSPECIFIED/zones/us-central1-a/disks/pvc-fe42fc7f-7618-4d3b-94d1-a2490cfd009d"`,
			want: []*commonlogk8saudit_contract.ResourceIdentity{
				{
					APIVersion: "core/v1",
					Kind:       "node",
					Namespace:  "cluster-scope",
					Name:       "node-foo",
				},
				{
					APIVersion: "core/v1",
					Kind:       "persistentvolume",
					Namespace:  "cluster-scope",
					Name:       "pvc-fe42fc7f-7618-4d3b-94d1-a2490cfd009d",
				},
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
			extractor := &K8sControllerManagerComponentExtractor{
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
					{
						APIVersion:   "core/v1",
						KindName:     "persistentvolume",
						KLogField:    "volumeName",
						IsNamespaced: false,
					},
				},
			}

			klogParser := logutil.NewKLogTextParser(false)
			paths := extractor.ReadResourceAssociationFromControllerSpecificField(klogParser.TryParse(tc.input))
			if diff := cmp.Diff(tc.want, paths, cmpopts.SortSlices(func(a, b *commonlogk8saudit_contract.ResourceIdentity) int {
				return strings.Compare(a.String(), b.String())
			})); diff != "" {
				t.Errorf("ReadResourceAssociationFromControllerSpecificField() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sControllerManagerComponentExtractor_ReadResourceAssociationFromItems(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *commonlogk8saudit_contract.ResourceIdentity
	}{
		{
			desc:  "valid item field - namespaced",
			input: `"Deleting item" logger="garbage-collector-controller" item="[coordination.k8s.io/v1/Lease, namespace: kube-node-lease, name: gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v, uid: 8aba20bf-0392-40c9-ae35-240b7c099523]" propagationPolicy="Background"`,
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "coordination.k8s.io/v1",
				Kind:       "lease",
				Namespace:  "kube-node-lease",
				Name:       "gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v",
			},
		},
		{
			desc:  "valid item field - cluster-scoped",
			input: `"Deleting item" logger="garbage-collector-controller" item="[rbac.authorization.k8s.io/v1/ClusterRole, namespace: , name: admin, uid: 8aba20bf-0392-40c9-ae35-240b7c099523]" propagationPolicy="Background"`,
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "clusterrole",
				Namespace:  "cluster-scope",
				Name:       "admin",
			},
		},
		{
			desc:  "valid item field - in core api version",
			input: `"Deleting item" logger="garbage-collector-controller" item="[v1/Pod, namespace: kube-system, name: gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v, uid: 8aba20bf-0392-40c9-ae35-240b7c099523]" propagationPolicy="Background"`,
			want: &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "kube-system",
				Name:       "gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v",
			},
		},
		{
			desc:  "item field missing",
			input: `"Deleting item" logger="garbage-collector-controller" propagationPolicy="Background"`,
			want:  nil,
		},
		{
			desc:  "item field malformed",
			input: `"Deleting item" logger="garbage-collector-controller" item="malformed-item" propagationPolicy="Background"`,
			want:  nil,
		},
		{
			desc:  "item field malformed - no slash contained in apiVersion",
			input: `"Deleting item" logger="garbage-collector-controller" item="[Pod, namespace: kube-system, name: gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v, uid: 8aba20bf-0392-40c9-ae35-240b7c099523]" propagationPolicy="Background"`,
			want:  nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			extractor := &K8sControllerManagerComponentExtractor{}
			klogParser := logutil.NewKLogTextParser(false)
			path := extractor.ReadResourceAssociationFromItems(klogParser.TryParse(tc.input))

			if diff := cmp.Diff(tc.want, path); diff != "" {
				t.Errorf("ReadResourceAssociationFromItems() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractK8sControllerManagerComponent_FromMock(t *testing.T) {
	mock := K8sControllerManagerComponentFieldSet{
		Controller: "mock-controller",
		AssociatedResources: []*commonlogk8saudit_contract.ResourceIdentity{
			{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "pod-1",
			},
		},
	}
	l := testlog.NewMockLog(mock)
	extractor := &K8sControllerManagerComponentExtractor{}
	got, err := extractor.Extract(l.NodeReader)
	if err != nil {
		t.Fatalf("Extract() unexpected error: %v", err)
	}
	if diff := cmp.Diff(mock, got); diff != "" {
		t.Errorf("Extract() mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractK8sHPAControllerComponent(t *testing.T) {
	testCases := []struct {
		desc      string
		inputYAML string
		mock      *K8sHPAControllerFieldSet
		want      K8sHPAControllerFieldSet
	}{
		{
			desc: "atomic recommendation log entry",
			inputYAML: `
jsonPayload:
  atomicRecommendation:
    hpa: gke-managed-cim/kube-state-metrics
    metric:
      newestSampleAgeSeconds: -34.480346499
      newestSampleTime: "2026-08-26T01:07:26.771Z"
      spec:
        name: memory
        target:
          averageValue: "400Mi"
      status:
        averageUtilization: 27
        averageValue: "36745216"
        value: "36745216"
      type: Resource
    podCount:
      ready: 1
      total: 1
    startTime: "2026-08-26T01:08:01.251346499Z"
    summary:
      dampening: none
      override: none
      replicas: 1
`,
			want: K8sHPAControllerFieldSet{
				AtomicRecommendation: &HPAAtomicRecommendation{
					HPA:                    "gke-managed-cim/kube-state-metrics",
					HPANamespace:           "gke-managed-cim",
					HPAName:                "kube-state-metrics",
					StartTime:              "2026-08-26T01:08:01.251346499Z",
					MetricType:             "Resource",
					MetricName:             "memory",
					SpecTargetAvgValue:     "400Mi",
					SpecTargetAvgUtil:      -1,
					StatusAvgValue:         "36745216",
					StatusAvgUtil:          27,
					StatusValue:            "36745216",
					NewestSampleAgeSeconds: -34.480346499,
					NewestSampleTime:       "2026-08-26T01:07:26.771Z",
					PodCount: HPAPodCount{
						Ready: 1,
						Total: 1,
					},
					Dampening: "none",
					Override:  "none",
					Replicas:  1,
				},
			},
		},
		{
			desc: "final recommendation log entry",
			inputYAML: `
jsonPayload:
  finalRecommendation:
    actuationLatencySeconds: 0.000970647
    actuationTime: "2026-08-25T00:24:25.753642670Z"
    configuredSize: 1
    hpa: gke-managed-cim/kube-state-metrics
    leadingMetricIndex: 0
    replicas: 1
    startTime: "2026-08-25T00:24:25.752672023Z"
    targetRef:
      apiVersion: apps/v1
      kind: StatefulSet
      name: kube-state-metrics
    topLevelLimit: none
    topLevelOverride: none
`,
			want: K8sHPAControllerFieldSet{
				FinalRecommendation: &HPAFinalRecommendation{
					HPA:                     "gke-managed-cim/kube-state-metrics",
					HPANamespace:            "gke-managed-cim",
					HPAName:                 "kube-state-metrics",
					StartTime:               "2026-08-25T00:24:25.752672023Z",
					ActuationTime:           "2026-08-25T00:24:25.753642670Z",
					ActuationLatencySeconds: 0.000970647,
					ConfiguredSize:          1,
					Replicas:                1,
					LeadingMetricIndex:      0,
					TargetRef: HPATargetRef{
						APIVersion: "apps/v1",
						Kind:       "StatefulSet",
						Name:       "kube-state-metrics",
					},
					TopLevelLimit:    "none",
					TopLevelOverride: "none",
				},
			},
		},
		{
			desc: "fallback message entry",
			inputYAML: `
jsonPayload:
  message: "hpa controller leader elected"
`,
			want: K8sHPAControllerFieldSet{
				Message: "hpa controller leader elected",
			},
		},
		{
			desc: "from mock",
			mock: &K8sHPAControllerFieldSet{
				Message: "mock-message",
			},
			want: K8sHPAControllerFieldSet{
				Message: "mock-message",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var l *log.Log
			var err error
			if tc.mock != nil {
				l = testlog.NewMockLog(*tc.mock)
			} else {
				l, err = log.NewLogFromYAMLString(tc.inputYAML)
				if err != nil {
					t.Fatalf("failed to parse test input YAML: %v", err)
				}
			}

			got, err := ExtractK8sHPAControllerComponent(l.NodeReader)
			if err != nil {
				t.Fatalf("ExtractK8sHPAControllerComponent() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractK8sHPAControllerComponent() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sHPAControllerFieldSet_Summary(t *testing.T) {
	testCases := []struct {
		desc  string
		field K8sHPAControllerFieldSet
		want  string
	}{
		{
			desc: "final recommendation scaling workload",
			field: K8sHPAControllerFieldSet{
				FinalRecommendation: &HPAFinalRecommendation{
					HPA:            "default/web",
					ConfiguredSize: 1,
					Replicas:       3,
					TargetRef: HPATargetRef{
						Kind: "Deployment",
						Name: "web",
					},
				},
			},
			want: "[HPA Final] default/web: Deployment/web 1 -> 3 replicas",
		},
		{
			desc: "final recommendation maintaining workload",
			field: K8sHPAControllerFieldSet{
				FinalRecommendation: &HPAFinalRecommendation{
					HPA:            "gke-managed-cim/kube-state-metrics",
					ConfiguredSize: 1,
					Replicas:       1,
					TargetRef: HPATargetRef{
						Kind: "StatefulSet",
						Name: "kube-state-metrics",
					},
				},
			},
			want: "[HPA Final] gke-managed-cim/kube-state-metrics: StatefulSet/kube-state-metrics = 1 replicas",
		},
		{
			desc: "final recommendation with limit and override",
			field: K8sHPAControllerFieldSet{
				FinalRecommendation: &HPAFinalRecommendation{
					HPA:              "default/web",
					ConfiguredSize:   2,
					Replicas:         5,
					TopLevelLimit:    "maxReplicas",
					TopLevelOverride: "scaleUpStabilization",
					TargetRef: HPATargetRef{
						Kind: "Deployment",
						Name: "web",
					},
				},
			},
			want: "[HPA Final] default/web: Deployment/web 2 -> 5 replicas (limit: maxReplicas, override: scaleUpStabilization)",
		},
		{
			desc: "final recommendation without targetRef",
			field: K8sHPAControllerFieldSet{
				FinalRecommendation: &HPAFinalRecommendation{
					HPA:            "default/web",
					ConfiguredSize: 2,
					Replicas:       4,
				},
			},
			want: "[HPA Final] default/web: 2 -> 4 replicas",
		},
		{
			desc: "atomic recommendation with metric name, status util, and target value",
			field: K8sHPAControllerFieldSet{
				AtomicRecommendation: &HPAAtomicRecommendation{
					HPA:                "gke-managed-cim/kube-state-metrics",
					MetricName:         "memory",
					StatusAvgUtil:      27,
					SpecTargetAvgUtil:  -1,
					SpecTargetAvgValue: "400Mi",
					Replicas:           1,
				},
			},
			want: "[HPA Metric] gke-managed-cim/kube-state-metrics: memory (27% / target 400Mi) -> 1 replicas",
		},
		{
			desc: "atomic recommendation with 0% utilization",
			field: K8sHPAControllerFieldSet{
				AtomicRecommendation: &HPAAtomicRecommendation{
					HPA:               "default/web",
					MetricName:        "cpu",
					StatusAvgUtil:     0,
					SpecTargetAvgUtil: 50,
					Replicas:          1,
				},
			},
			want: "[HPA Metric] default/web: cpu (0% / target 50%) -> 1 replicas",
		},
		{
			desc: "atomic recommendation with dampening and override",
			field: K8sHPAControllerFieldSet{
				AtomicRecommendation: &HPAAtomicRecommendation{
					HPA:               "default/web",
					MetricName:        "cpu",
					StatusAvgUtil:     85,
					SpecTargetAvgUtil: 80,
					Dampening:         "up",
					Override:          "tolerance",
					Replicas:          3,
				},
			},
			want: "[HPA Metric] default/web: cpu (85% / target 80%) -> 3 replicas (dampened: up, override: tolerance)",
		},
		{
			desc: "atomic recommendation minimal",
			field: K8sHPAControllerFieldSet{
				AtomicRecommendation: &HPAAtomicRecommendation{
					HPA:               "default/web",
					MetricType:        "Resource",
					SpecTargetAvgUtil: -1,
					StatusAvgUtil:     -1,
					Replicas:          2,
				},
			},
			want: "[HPA Metric] default/web: Resource -> 2 replicas",
		},
		{
			desc: "fallback message",
			field: K8sHPAControllerFieldSet{
				Message: "custom message",
			},
			want: "custom message",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.field.Summary()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Summary() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
