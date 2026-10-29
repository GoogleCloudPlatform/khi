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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

func TestContainerIDDiscoveryTask(t *testing.T) {
	testTime := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		desc          string
		inputResource *k8saudit.ResourceIdentity
		inputManifest string
		want          k8saudit.ContainerIDToContainerIdentity
	}{
		{
			desc: "extracts containerStatuses, initContainerStatuses, and ephemeralContainerStatuses with pod identity",
			inputResource: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Name:       "test-pod",
				Namespace:  "test-namespace",
			},
			inputManifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: test-namespace
status:
  containerStatuses:
    - name: main-app
      containerID: containerd://1111111111111111111111111111111111111111111111111111111111111111
  initContainerStatuses:
    - name: init-db
      containerID: containerd://2222222222222222222222222222222222222222222222222222222222222222
  ephemeralContainerStatuses:
    - name: debugger
      containerID: docker://3333333333333333333333333333333333333333333333333333333333333333`,
			want: k8saudit.ContainerIDToContainerIdentity{
				"1111111111111111111111111111111111111111111111111111111111111111": {
					ContainerID:   "1111111111111111111111111111111111111111111111111111111111111111",
					ContainerName: "main-app",
					PodNamespace:  "test-namespace",
					PodName:       "test-pod",
				},
				"2222222222222222222222222222222222222222222222222222222222222222": {
					ContainerID:   "2222222222222222222222222222222222222222222222222222222222222222",
					ContainerName: "init-db",
					PodNamespace:  "test-namespace",
					PodName:       "test-pod",
				},
				"3333333333333333333333333333333333333333333333333333333333333333": {
					ContainerID:   "3333333333333333333333333333333333333333333333333333333333333333",
					ContainerName: "debugger",
					PodNamespace:  "test-namespace",
					PodName:       "test-pod",
				},
			},
		},
		{
			desc: "ignores non-pod resources",
			inputResource: &k8saudit.ResourceIdentity{
				APIVersion: "apps/v1",
				Kind:       "deployment",
				Name:       "test-deploy",
				Namespace:  "test-namespace",
			},
			inputManifest: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy
  namespace: test-namespace`,
			want: k8saudit.ContainerIDToContainerIdentity{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := testlog.NewMockLog(testTime)
			yamlNode, err := structured.FromYAML(tc.inputManifest)
			if err != nil {
				t.Fatalf("FromYAML() failed: %v", err)
			}

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			input := k8saudit.ResourceManifestLogGroupMap{
				"test": {
					Resource: tc.inputResource,
					Logs: []*k8saudit.ResourceManifestLog{
						{
							Log:                l,
							ResourceBodyReader: structured.NewNodeReader(yamlNode),
						},
					},
				},
			}
			got, _, err := inspectiontest.Run(t, ctx, containerIDDiscoveryTask, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.Given(k8saudit.ManifestGeneratorTaskID.Ref(), input),
			)
			if err != nil {
				t.Fatalf("inspectiontest.Run() failed: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("containerIDDiscoveryTask mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeContainerIDs(t *testing.T) {
	testCases := []struct {
		desc   string
		inputs []k8saudit.ContainerIDToContainerIdentity
		want   k8saudit.ContainerIDToContainerIdentity
	}{
		{
			desc: "merges audit and containerd discoveries for the same container ID",
			inputs: []k8saudit.ContainerIDToContainerIdentity{
				{
					"cid-1": {
						ContainerID:   "cid-1",
						ContainerName: "app",
						PodNamespace:  "ns-1",
						PodName:       "pod-1",
					},
				},
				{
					"cid-1": {
						ContainerID:   "cid-1",
						ContainerName: "app",
						PodSandboxID:  "sandbox-1",
					},
				},
			},
			want: k8saudit.ContainerIDToContainerIdentity{
				"cid-1": {
					ContainerID:   "cid-1",
					ContainerName: "app",
					PodSandboxID:  "sandbox-1",
					PodNamespace:  "ns-1",
					PodName:       "pod-1",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := mergeContainerIDs(tc.inputs)
			if err != nil {
				t.Fatalf("mergeContainerIDs() failed: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mergeContainerIDs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
