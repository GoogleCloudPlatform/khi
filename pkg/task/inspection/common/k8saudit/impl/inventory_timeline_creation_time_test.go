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

package k8saudit_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

func TestMergeTimelineCreationTimes(t *testing.T) {
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	p1 := MustResolveTimelinePath(ctx, "c", &k8saudit.ResourceIdentity{APIVersion: "core/v1", Kind: "pod", Namespace: "default", Name: "p1"})
	p2 := MustResolveTimelinePath(ctx, "c", &k8saudit.ResourceIdentity{APIVersion: "core/v1", Kind: "pod", Namespace: "default", Name: "p2"})
	p3 := MustResolveTimelinePath(ctx, "c", &k8saudit.ResourceIdentity{APIVersion: "core/v1", Kind: "pod", Namespace: "default", Name: "p3"})

	t1 := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name   string
		inputs []k8saudit.TimelineCreationTimes
		want   k8saudit.TimelineCreationTimes
	}{
		{
			name: "merge disjoint sets",
			inputs: []k8saudit.TimelineCreationTimes{
				{p1: []time.Time{t1}},
				{p2: []time.Time{t2}, p3: []time.Time{t3}},
			},
			want: k8saudit.TimelineCreationTimes{
				p1: []time.Time{t1},
				p2: []time.Time{t2},
				p3: []time.Time{t3},
			},
		},
		{
			name: "merge overlapping sets with duplicate and out-of-order timestamps",
			inputs: []k8saudit.TimelineCreationTimes{
				{p1: []time.Time{t2, t1}},
				{p1: []time.Time{t1, t3}, p2: []time.Time{t1}},
			},
			want: k8saudit.TimelineCreationTimes{
				p1: []time.Time{t1, t2, t3},
				p2: []time.Time{t1},
			},
		},
		{
			name:   "empty inputs",
			inputs: []k8saudit.TimelineCreationTimes{},
			want:   k8saudit.TimelineCreationTimes{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mergeTimelineCreationTimes(tc.inputs)
			if err != nil {
				t.Fatalf("mergeTimelineCreationTimes() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got, cmp.Comparer(func(a, b *khifilev6.TimelinePath) bool { return a == b })); diff != "" {
				t.Errorf("mergeTimelineCreationTimes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResourceTimelineCreationTimeDiscoveryTask(t *testing.T) {
	testTime := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)

	type logSpec struct {
		manifest string
		verb     *pb.Verb
		logTime  time.Time
		isDryRun bool
	}

	testCases := []struct {
		name        string
		resource    *k8saudit.ResourceIdentity
		logs        []logSpec
		clusterName string
		wantFunc    func(ctx context.Context) k8saudit.TimelineCreationTimes
	}{
		{
			name: "standard pod resource with creationTimestamp",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "test-pod",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-123
  creationTimestamp: "2026-05-26T10:00:00Z"
spec:
  nodeName: worker-node-1
`,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				podPath := MustResolveTimelinePath(ctx, "test-cluster", &k8saudit.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "test-pod",
				})
				return k8saudit.TimelineCreationTimes{
					podPath: []time.Time{time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)},
				}
			},
		},
		{
			name: "recreated pod on the same timeline path with multiple creationTimestamps and duplicate logs",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "test-pod",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-2
  creationTimestamp: "2026-05-26T11:00:00Z"
spec: {}
`,
				},
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-1
  creationTimestamp: "2026-05-26T10:00:00Z"
spec: {}
`,
				},
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-1
  creationTimestamp: "2026-05-26T10:00:00Z"
spec: {}
`,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				podPath := MustResolveTimelinePath(ctx, "test-cluster", &k8saudit.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "test-pod",
				})
				return k8saudit.TimelineCreationTimes{
					podPath: []time.Time{
						time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC),
						time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC),
					},
				}
			},
		},
		{
			name: "pod log without creationTimestamp and non-create verb is excluded",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "test-pod",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
`,
					verb: k8saudit.VerbDelete,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				return k8saudit.TimelineCreationTimes{}
			},
		},
		{
			name: "pod binding subresource with VerbCreate falls back to log timestamp when creationTimestamp is absent",
			resource: &k8saudit.ResourceIdentity{
				APIVersion:      "core/v1",
				Kind:            "pod",
				Namespace:       "default",
				Name:            "test-pod",
				SubresourceName: "binding",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: Binding
metadata:
  name: test-pod
  namespace: default
target:
  kind: Node
  name: worker-node-1
`,
					verb:    k8saudit.VerbCreate,
					logTime: testTime,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				bindingPath := MustResolveTimelinePath(ctx, "test-cluster", &k8saudit.ResourceIdentity{
					APIVersion:      "core/v1",
					Kind:            "pod",
					Namespace:       "default",
					Name:            "test-pod",
					SubresourceName: "binding",
				})
				return k8saudit.TimelineCreationTimes{
					bindingPath: []time.Time{testTime},
				}
			},
		},
		{
			name: "dry run log is excluded",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "test-pod",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  creationTimestamp: "2026-05-26T10:00:00Z"
`,
					isDryRun: true,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				return k8saudit.TimelineCreationTimes{}
			},
		},
		{
			name: "namespace resource type is skipped",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: Namespace
metadata:
  name: kube-system
  creationTimestamp: "2026-05-26T10:00:00Z"
`,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				return k8saudit.TimelineCreationTimes{}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())

			var manifestLogs []*k8saudit.ResourceManifestLog
			for _, ls := range tc.logs {
				logTime := ls.logTime
				if logTime.IsZero() {
					logTime = testTime
				}
				fieldSet := &k8saudit.K8sAuditLogFieldSet{
					ClusterName: tc.clusterName,
					IsDryRun:    ls.isDryRun,
					Verb:        ls.verb,
				}
				l := testlog.NewMockLog(logTime, fieldSet)

				var reader *structured.NodeReader
				if ls.manifest != "" {
					yamlNode, err := structured.FromYAML(ls.manifest)
					if err != nil {
						t.Fatalf("failed to parse yaml: %v", err)
					}
					reader = structured.NewNodeReader(yamlNode)
				}
				manifestLogs = append(manifestLogs, &k8saudit.ResourceManifestLog{
					Log:                l,
					ResourceBodyReader: reader,
				})
			}

			input := k8saudit.ResourceManifestLogGroupMap{
				"test": &k8saudit.ResourceManifestLogGroup{
					Resource: tc.resource,
					Logs:     manifestLogs,
				},
			}

			mockExtractor := k8saudit.K8sAuditLogExtractor(func(reader *structured.NodeReader) (*k8saudit.K8sAuditLogFieldSet, error) {
				if mock, ok := structured.GetMock[*k8saudit.K8sAuditLogFieldSet](reader); ok {
					return mock, nil
				}
				return nil, nil
			})

			got, _, err := inspectiontest.Run(t, ctx, resourceTimelineCreationTimeDiscoveryTask, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.Given(k8saudit.ManifestGeneratorTaskID.Ref(), input),
				tasktest.Given(k8saudit.K8sAuditLogExtractorRef, mockExtractor),
			)
			if err != nil {
				t.Fatalf("inspectiontest.Run() failed: %v", err)
			}

			want := tc.wantFunc(ctx)
			if diff := cmp.Diff(want, got, cmp.Comparer(func(a, b *khifilev6.TimelinePath) bool { return a == b })); diff != "" {
				t.Errorf("resourceTimelineCreationTimeDiscoveryTask mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPodPhaseTimelineCreationTimeDiscoveryTask(t *testing.T) {
	testTime := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)

	type logSpec struct {
		manifest string
		verb     *pb.Verb
		logTime  time.Time
		isDryRun bool
	}

	testCases := []struct {
		name        string
		resource    *k8saudit.ResourceIdentity
		logs        []logSpec
		clusterName string
		wantFunc    func(ctx context.Context) k8saudit.TimelineCreationTimes
	}{
		{
			name: "scheduled pod with creationTimestamp, uid, and spec.nodeName",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "test-pod",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-123
  creationTimestamp: "2026-05-26T10:00:00Z"
spec:
  nodeName: worker-node-1
`,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				podPhasePath := MustPodPhaseTimelinePath(ctx, "test-cluster", "worker-node-1", "default", "test-pod", "pod-uid-123")
				return k8saudit.TimelineCreationTimes{
					podPhasePath: []time.Time{time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)},
				}
			},
		},
		{
			name: "multiple logs with duplicate and multiple timestamps deduplicated and sorted",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "test-pod",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-123
  creationTimestamp: "2026-05-26T11:00:00Z"
spec:
  nodeName: worker-node-1
`,
				},
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-123
  creationTimestamp: "2026-05-26T10:00:00Z"
spec:
  nodeName: worker-node-1
`,
				},
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-123
  creationTimestamp: "2026-05-26T10:00:00Z"
spec:
  nodeName: worker-node-1
`,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				podPhasePath := MustPodPhaseTimelinePath(ctx, "test-cluster", "worker-node-1", "default", "test-pod", "pod-uid-123")
				return k8saudit.TimelineCreationTimes{
					podPhasePath: []time.Time{
						time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC),
						time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC),
					},
				}
			},
		},
		{
			name: "unscheduled pod without spec.nodeName is excluded",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "test-pod",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-123
  creationTimestamp: "2026-05-26T10:00:00Z"
spec: {}
`,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				return k8saudit.TimelineCreationTimes{}
			},
		},
		{
			name: "pod without creationTimestamp and non-create verb is excluded",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "test-pod",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-123
spec:
  nodeName: worker-node-1
`,
					verb: k8saudit.VerbDelete,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				return k8saudit.TimelineCreationTimes{}
			},
		},
		{
			name: "non-pod resource skipped",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "configmap",
				Namespace:  "default",
				Name:       "test-cm",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: default
  uid: cm-uid-123
  creationTimestamp: "2026-05-26T10:00:00Z"
`,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				return k8saudit.TimelineCreationTimes{}
			},
		},
		{
			name: "dry run log is excluded",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "test-pod",
			},
			logs: []logSpec{
				{
					manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-123
  creationTimestamp: "2026-05-26T10:00:00Z"
spec:
  nodeName: worker-node-1
`,
					isDryRun: true,
				},
			},
			clusterName: "test-cluster",
			wantFunc: func(ctx context.Context) k8saudit.TimelineCreationTimes {
				return k8saudit.TimelineCreationTimes{}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())

			var manifestLogs []*k8saudit.ResourceManifestLog
			for _, ls := range tc.logs {
				logTime := ls.logTime
				if logTime.IsZero() {
					logTime = testTime
				}
				fieldSet := &k8saudit.K8sAuditLogFieldSet{
					ClusterName: tc.clusterName,
					IsDryRun:    ls.isDryRun,
					Verb:        ls.verb,
				}
				l := testlog.NewMockLog(logTime, fieldSet)

				var reader *structured.NodeReader
				if ls.manifest != "" {
					yamlNode, err := structured.FromYAML(ls.manifest)
					if err != nil {
						t.Fatalf("failed to parse yaml: %v", err)
					}
					reader = structured.NewNodeReader(yamlNode)
				}
				manifestLogs = append(manifestLogs, &k8saudit.ResourceManifestLog{
					Log:                l,
					ResourceBodyReader: reader,
				})
			}

			input := k8saudit.ResourceManifestLogGroupMap{
				"test": &k8saudit.ResourceManifestLogGroup{
					Resource: tc.resource,
					Logs:     manifestLogs,
				},
			}

			mockExtractor := k8saudit.K8sAuditLogExtractor(func(reader *structured.NodeReader) (*k8saudit.K8sAuditLogFieldSet, error) {
				if mock, ok := structured.GetMock[*k8saudit.K8sAuditLogFieldSet](reader); ok {
					return mock, nil
				}
				return nil, nil
			})

			got, _, err := inspectiontest.Run(t, ctx, podPhaseTimelineCreationTimeDiscoveryTask, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.Given(k8saudit.ManifestGeneratorTaskID.Ref(), input),
				tasktest.Given(k8saudit.K8sAuditLogExtractorRef, mockExtractor),
			)
			if err != nil {
				t.Fatalf("inspectiontest.Run() failed: %v", err)
			}

			want := tc.wantFunc(ctx)
			if diff := cmp.Diff(want, got, cmp.Comparer(func(a, b *khifilev6.TimelinePath) bool { return a == b })); diff != "" {
				t.Errorf("podPhaseTimelineCreationTimeDiscoveryTask mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
