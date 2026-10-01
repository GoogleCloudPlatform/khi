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
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

func TestMergeTimelinePaths(t *testing.T) {
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	p1 := MustResolveTimelinePath(ctx, "c", &k8saudit.ResourceIdentity{APIVersion: "core/v1", Kind: "pod", Namespace: "default", Name: "p1"})
	p2 := MustResolveTimelinePath(ctx, "c", &k8saudit.ResourceIdentity{APIVersion: "core/v1", Kind: "pod", Namespace: "default", Name: "p2"})
	p3 := MustResolveTimelinePath(ctx, "c", &k8saudit.ResourceIdentity{APIVersion: "core/v1", Kind: "pod", Namespace: "default", Name: "p3"})

	testCases := []struct {
		name   string
		inputs []k8saudit.TimelinePathSet
		want   k8saudit.TimelinePathSet
	}{
		{
			name: "merge disjoint sets",
			inputs: []k8saudit.TimelinePathSet{
				{p1: struct{}{}},
				{p2: struct{}{}, p3: struct{}{}},
			},
			want: k8saudit.TimelinePathSet{
				p1: struct{}{},
				p2: struct{}{},
				p3: struct{}{},
			},
		},
		{
			name: "merge overlapping sets",
			inputs: []k8saudit.TimelinePathSet{
				{p1: struct{}{}, p2: struct{}{}},
				{p2: struct{}{}, p3: struct{}{}},
			},
			want: k8saudit.TimelinePathSet{
				p1: struct{}{},
				p2: struct{}{},
				p3: struct{}{},
			},
		},
		{
			name:   "empty inputs",
			inputs: []k8saudit.TimelinePathSet{},
			want:   k8saudit.TimelinePathSet{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mergeTimelinePaths(tc.inputs)
			if err != nil {
				t.Fatalf("mergeTimelinePaths() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got, cmp.Comparer(func(a, b *khifilev6.TimelinePath) bool { return a == b })); diff != "" {
				t.Errorf("mergeTimelinePaths() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTimelinePathDiscoveryTask(t *testing.T) {
	testTime := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name          string
		resource      *k8saudit.ResourceIdentity
		manifest      string
		clusterName   string
		isDryRun      bool
		wantPathsFunc func(ctx context.Context) []*khifilev6.TimelinePath
	}{
		{
			name: "standard pod resource with UID and spec.nodeName",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "test-pod",
			},
			manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  uid: pod-uid-123
spec:
  nodeName: worker-node-1
`,
			clusterName: "test-cluster",
			wantPathsFunc: func(ctx context.Context) []*khifilev6.TimelinePath {
				podPath := MustResolveTimelinePath(ctx, "test-cluster", &k8saudit.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "test-pod",
				})
				podPhasePath := MustPodPhaseTimelinePath(ctx, "test-cluster", "worker-node-1", "default", "test-pod", "pod-uid-123")
				return []*khifilev6.TimelinePath{podPath, podPhasePath}
			},
		},
		{
			name: "pending pod without spec.nodeName",
			resource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "pending-pod",
			},
			manifest: `apiVersion: v1
kind: Pod
metadata:
  name: pending-pod
  namespace: default
  uid: pod-uid-456
spec: {}
`,
			clusterName: "test-cluster",
			wantPathsFunc: func(ctx context.Context) []*khifilev6.TimelinePath {
				podPath := MustResolveTimelinePath(ctx, "test-cluster", &k8saudit.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "pending-pod",
				})
				return []*khifilev6.TimelinePath{podPath}
			},
		},
		{
			name: "pod binding subresource",
			resource: &k8saudit.ResourceIdentity{
				APIVersion:      "core/v1",
				Kind:            "pod",
				Namespace:       "default",
				Name:            "test-pod",
				SubresourceName: "binding",
			},
			manifest: `apiVersion: v1
kind: Binding
metadata:
  name: test-pod
  namespace: default
target:
  kind: Node
  name: worker-node-1
`,
			clusterName: "test-cluster",
			wantPathsFunc: func(ctx context.Context) []*khifilev6.TimelinePath {
				bindingPath := MustResolveTimelinePath(ctx, "test-cluster", &k8saudit.ResourceIdentity{
					APIVersion:      "core/v1",
					Kind:            "pod",
					Namespace:       "default",
					Name:            "test-pod",
					SubresourceName: "binding",
				})
				return []*khifilev6.TimelinePath{bindingPath}
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
			manifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
`,
			clusterName: "test-cluster",
			isDryRun:    true,
			wantPathsFunc: func(ctx context.Context) []*khifilev6.TimelinePath {
				return nil
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
			manifest: `apiVersion: v1
kind: Namespace
metadata:
  name: kube-system
`,
			clusterName: "test-cluster",
			wantPathsFunc: func(ctx context.Context) []*khifilev6.TimelinePath {
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fieldSet := &k8saudit.K8sAuditLogFieldSet{
				ClusterName: tc.clusterName,
				IsDryRun:    tc.isDryRun,
			}
			l := testlog.NewMockLog(testTime, fieldSet)

			yamlNode, err := structured.FromYAML(tc.manifest)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			input := k8saudit.ResourceManifestLogGroupMap{
				"test": &k8saudit.ResourceManifestLogGroup{
					Resource: tc.resource,
					Logs: []*k8saudit.ResourceManifestLog{
						{
							Log:                l,
							ResourceBodyReader: structured.NewNodeReader(yamlNode),
						},
					},
				},
			}

			mockExtractor := k8saudit.K8sAuditLogExtractor(func(reader *structured.NodeReader) (*k8saudit.K8sAuditLogFieldSet, error) {
				return fieldSet, nil
			})

			got, _, err := inspectiontest.RunInspectionTask(ctx, TimelinePathDiscoveryTask, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(k8saudit.ManifestGeneratorTaskID.Ref(), input),
				tasktest.NewTaskDependencyValuePair(k8saudit.K8sAuditLogExtractorRef, mockExtractor),
			)
			if err != nil {
				t.Fatalf("RunInspectionTask failed: %v", err)
			}

			wantPaths := tc.wantPathsFunc(ctx)
			want := k8saudit.TimelinePathSet{}
			for _, p := range wantPaths {
				want[p] = struct{}{}
			}
			if diff := cmp.Diff(want, got, cmp.Comparer(func(a, b *khifilev6.TimelinePath) bool { return a == b })); diff != "" {
				t.Errorf("TimelinePathDiscoveryTask mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
