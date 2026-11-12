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

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

// TestResourceOwnerReferenceTimelineMapperTask_ProcessLog verifies that owner references in the resource body are mapped correctly to timeline aliases.
func TestResourceOwnerReferenceTimelineMapperTask_ProcessLog(t *testing.T) {
	task := &resourceOwnerReferenceTimelineMapperTaskSetting{
		nonNamespacedOwnerTypes: map[string]struct{}{
			"core/v1#node": {},
		},
	}

	parseYAML := func(yamlStr string) *structured.NodeReader {
		if yamlStr == "" {
			return nil
		}
		node, err := structured.FromYAML(yamlStr)
		if err != nil {
			t.Fatalf("failed to parse YAML: %v", err)
		}
		return structured.NewNodeReader(node)
	}

	tests := []struct {
		name     string
		yaml     string
		isDryRun bool
		assert   func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context)
	}{
		{
			name: "No Owner References",
			yaml: `
metadata:
  name: nginx
  namespace: default
`,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				ownerPath := MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
					APIVersion: "apps/v1",
					Kind:       "replicaset",
					Namespace:  "default",
					Name:       "nginx-replicaset",
				})
				expectedTargetPath := k8saudit.MustOwnedResourceTimeline(ctx, ownerPath, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasNoAlias(expectedTargetPath)
			},
		},
		{
			name: "Single Owner (Namespaced)",
			yaml: `
metadata:
  name: nginx
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    kind: ReplicaSet
    name: nginx-replicaset
    uid: 12345
`,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				aliasPath := MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "nginx",
				})
				ownerPath := MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
					APIVersion: "apps/v1",
					Kind:       "replicaset",
					Namespace:  "default",
					Name:       "nginx-replicaset",
				})
				expectedTargetPath := k8saudit.MustOwnedResourceTimeline(ctx, ownerPath, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasAlias(expectedTargetPath, aliasPath)
			},
		},
		{
			name: "Single Owner (Cluster Scoped)",
			yaml: `
metadata:
  name: nginx
  namespace: default
  ownerReferences:
  - apiVersion: v1
    kind: Node
    name: node-1
    uid: 67890
`,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				aliasPath := MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "nginx",
				})
				ownerPath := MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "node",
					Namespace:  "cluster-scope",
					Name:       "node-1",
				})
				expectedTargetPath := k8saudit.MustOwnedResourceTimeline(ctx, ownerPath, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasAlias(expectedTargetPath, aliasPath)
			},
		},
		{
			name: "Multiple Owners",
			yaml: `
metadata:
  name: nginx
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    kind: ReplicaSet
    name: nginx-replicaset
  - apiVersion: v1
    kind: Node
    name: node-1
`,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				aliasPath := MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "nginx",
				})
				ownerPath1 := MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
					APIVersion: "apps/v1",
					Kind:       "replicaset",
					Namespace:  "default",
					Name:       "nginx-replicaset",
				})
				expectedTargetPath1 := k8saudit.MustOwnedResourceTimeline(ctx, ownerPath1, "nginx[kind:pod]")

				ownerPath2 := MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "node",
					Namespace:  "cluster-scope",
					Name:       "node-1",
				})
				expectedTargetPath2 := k8saudit.MustOwnedResourceTimeline(ctx, ownerPath2, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasAlias(expectedTargetPath1, aliasPath).
					HasAlias(expectedTargetPath2, aliasPath)
			},
		},
		{
			name: "Missing Fields",
			yaml: `
metadata:
  name: nginx
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    # Missing kind
    name: nginx-replicaset
  - apiVersion: apps/v1
    kind: ReplicaSet
    # Missing name
`,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				ownerPath := MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
					APIVersion: "apps/v1",
					Kind:       "replicaset",
					Namespace:  "default",
					Name:       "nginx-replicaset",
				})
				expectedTargetPath := k8saudit.MustOwnedResourceTimeline(ctx, ownerPath, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasNoAlias(expectedTargetPath)
			},
		},
		{
			name: "Nil Body",
			yaml: "",
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				ownerPath := MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
					APIVersion: "apps/v1",
					Kind:       "replicaset",
					Namespace:  "default",
					Name:       "nginx-replicaset",
				})
				expectedTargetPath := k8saudit.MustOwnedResourceTimeline(ctx, ownerPath, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasNoAlias(expectedTargetPath)
			},
		},
		{
			name:     "DryRun log should be ignored",
			isDryRun: true,
			yaml: `
metadata:
  name: nginx
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    kind: ReplicaSet
    name: nginx-replicaset
    uid: uid-1
`,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				ownerPath := MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
					APIVersion: "apps/v1",
					Kind:       "replicaset",
					Namespace:  "default",
					Name:       "nginx-replicaset",
				})
				expectedTargetPath := k8saudit.MustOwnedResourceTimeline(ctx, ownerPath, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasNoAlias(expectedTargetPath)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewTestBuilder(id.NewGenerator())
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
			nodeReader := parseYAML(tc.yaml)

			logObj := testlog.NewMockLog(
				time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				k8saudit.K8sAuditLogFieldSet{
					Principal:    "user-1",
					APIVersion:   "core/v1",
					PluralKind:   "pods",
					ResourceName: "nginx",
					Namespace:    "default",
					ClusterName:  "k8s",
					Verb:         k8saudit.VerbUpdate,
					IsDryRun:     tc.isDryRun,
				},
			)

			targetResource := &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "nginx",
			}

			groupSet := k8saudit.RelatedGroupSet{
				Roles: map[string]*k8saudit.ResourceManifestLogGroup{
					"target": {
						Resource: targetResource,
						Logs: []*k8saudit.ResourceManifestLog{
							{Log: logObj, ResourceBodyReader: nodeReader},
						},
					},
				},
			}

			event := k8saudit.MultiGroupLogEvent{
				Log:              logObj,
				GroupRole:        "target",
				ResourceIdentity: targetResource,
				EventType:        k8saudit.ChangeEventTypeModification,
				GroupSet:         groupSet,
			}

			cs, _, err := task.ProcessLog(ctx, event, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLog() failed: %v", err)
			}

			tc.assert(t, cs, ctx)
		})
	}
}

// TestResourceOwnerReferenceTimelineMapperTask_ProcessLog_PodOwnedByName verifies that resources owned by the Pod with the same name are linked to the Pod without their manifests.
func TestResourceOwnerReferenceTimelineMapperTask_ProcessLog_PodOwnedByName(t *testing.T) {
	task := &resourceOwnerReferenceTimelineMapperTaskSetting{
		nonNamespacedOwnerTypes: map[string]struct{}{
			"core/v1#node": {},
		},
		podOwnedTypesByName: map[string]struct{}{
			"cilium.io#ciliumendpoint": {},
		},
	}

	ciliumEndpoint := &k8saudit.ResourceIdentity{
		APIVersion: "cilium.io/v2",
		Kind:       "ciliumendpoint",
		Namespace:  "default",
		Name:       "nginx",
	}
	configMap := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "configmap",
		Namespace:  "default",
		Name:       "nginx",
	}
	podPath := func(ctx context.Context) *khifilev6.TimelinePath {
		return MustResolveTimelinePath(ctx, "k8s", &k8saudit.ResourceIdentity{
			APIVersion: "core/v1",
			Kind:       "pod",
			Namespace:  "default",
			Name:       "nginx",
		})
	}

	parseYAML := func(yamlStr string) *structured.NodeReader {
		if yamlStr == "" {
			return nil
		}
		node, err := structured.FromYAML(yamlStr)
		if err != nil {
			t.Fatalf("failed to parse YAML: %v", err)
		}
		return structured.NewNodeReader(node)
	}

	tests := []struct {
		name     string
		target   *k8saudit.ResourceIdentity
		yaml     string
		isDryRun bool
		assert   func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context)
	}{
		{
			name:   "CiliumEndpoint without manifest is linked to the Pod with the same name",
			target: ciliumEndpoint,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				ownedPath := k8saudit.MustOwnedResourceTimeline(ctx, podPath(ctx), "nginx[kind:ciliumendpoint]")
				targetPath := MustResolveTimelinePath(ctx, "k8s", ciliumEndpoint)

				testchangeset.AssertTimeline(t, cs).
					HasAlias(ownedPath, targetPath)
			},
		},
		{
			name:   "CiliumEndpoint owned by the Pod in its manifest is linked to the same alias",
			target: ciliumEndpoint,
			yaml: `
metadata:
  name: nginx
  namespace: default
  ownerReferences:
  - apiVersion: v1
    kind: Pod
    name: nginx
    uid: uid-1
`,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				ownedPath := k8saudit.MustOwnedResourceTimeline(ctx, podPath(ctx), "nginx[kind:ciliumendpoint]")
				targetPath := MustResolveTimelinePath(ctx, "k8s", ciliumEndpoint)

				testchangeset.AssertTimeline(t, cs).
					HasAlias(ownedPath, targetPath)
				if got := len(cs.Aliases); got != 1 {
					t.Errorf("len(cs.Aliases) = %d, want 1", got)
				}
			},
		},
		{
			name:   "resource of another type without manifest is not linked to the Pod",
			target: configMap,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				ownedPath := k8saudit.MustOwnedResourceTimeline(ctx, podPath(ctx), "nginx[kind:configmap]")

				testchangeset.AssertTimeline(t, cs).
					HasNoAlias(ownedPath)
			},
		},
		{
			name:     "dry run request for CiliumEndpoint is not linked",
			target:   ciliumEndpoint,
			isDryRun: true,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				ownedPath := k8saudit.MustOwnedResourceTimeline(ctx, podPath(ctx), "nginx[kind:ciliumendpoint]")

				testchangeset.AssertTimeline(t, cs).
					HasNoAlias(ownedPath)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewTestBuilder(id.NewGenerator())
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

			logObj := testlog.NewMockLog(
				time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				k8saudit.K8sAuditLogFieldSet{
					Principal:    "system:node:test-node",
					APIVersion:   tc.target.APIVersion,
					ResourceName: tc.target.Name,
					Namespace:    tc.target.Namespace,
					ClusterName:  "k8s",
					Verb:         k8saudit.VerbCreate,
					IsDryRun:     tc.isDryRun,
				},
			)

			groupSet := k8saudit.RelatedGroupSet{
				Roles: map[string]*k8saudit.ResourceManifestLogGroup{
					"target": {
						Resource: tc.target,
						Logs: []*k8saudit.ResourceManifestLog{
							{Log: logObj, ResourceBodyReader: parseYAML(tc.yaml)},
						},
					},
				},
			}

			event := k8saudit.MultiGroupLogEvent{
				Log:              logObj,
				GroupRole:        "target",
				ResourceIdentity: tc.target,
				EventType:        k8saudit.ChangeEventTypeCreation,
				GroupSet:         groupSet,
			}

			cs, _, err := task.ProcessLog(ctx, event, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLog() failed: %v", err)
			}

			tc.assert(t, cs, ctx)
		})
	}
}

// TestIsOwnedByPodWithSameName verifies that only namespaced resources of the declared types are treated as owned by the Pod with the same name.
func TestIsOwnedByPodWithSameName(t *testing.T) {
	task := &resourceOwnerReferenceTimelineMapperTaskSetting{
		podOwnedTypesByName: map[string]struct{}{
			"cilium.io#ciliumendpoint": {},
		},
	}

	testCases := []struct {
		desc     string
		identity *k8saudit.ResourceIdentity
		want     bool
	}{
		{
			desc:     "namespaced CiliumEndpoint",
			identity: &k8saudit.ResourceIdentity{APIVersion: "cilium.io/v2", Kind: "ciliumendpoint", Namespace: "default", Name: "nginx"},
			want:     true,
		},
		{
			desc:     "resource of a type not declared",
			identity: &k8saudit.ResourceIdentity{APIVersion: "core/v1", Kind: "configmap", Namespace: "default", Name: "nginx"},
			want:     false,
		},
		{
			desc:     "CiliumEndpoint in the cluster scope",
			identity: &k8saudit.ResourceIdentity{APIVersion: "cilium.io/v2", Kind: "ciliumendpoint", Namespace: k8saudit.ClusterScopeNamespace, Name: "nginx"},
			want:     false,
		},
		{
			desc:     "CiliumEndpoint without a namespace",
			identity: &k8saudit.ResourceIdentity{APIVersion: "cilium.io/v2", Kind: "ciliumendpoint", Name: "nginx"},
			want:     false,
		},
		{
			desc:     "subresource of a CiliumEndpoint",
			identity: &k8saudit.ResourceIdentity{APIVersion: "cilium.io/v2", Kind: "ciliumendpoint", Namespace: "default", Name: "nginx", SubresourceName: "status"},
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if got := task.isOwnedByPodWithSameName(tc.identity); got != tc.want {
				t.Errorf("isOwnedByPodWithSameName() = %v, want %v", got, tc.want)
			}
		})
	}
}
