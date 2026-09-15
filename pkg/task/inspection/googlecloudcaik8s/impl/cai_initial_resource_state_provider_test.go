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

package googlecloudcaik8s_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
)

// snapshotLogParams describes a CAI temporal asset log the provider indexes.
type snapshotLogParams struct {
	assetName       string
	assetType       string
	kind            string
	namespace       string
	name            string
	label           string
	windowStartTime time.Time
	windowEndTime   time.Time
	isDeleted       bool
}

func TestCAIInitialResourceStateProvider(t *testing.T) {
	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	pathMetadataLabelsOrigin := structured.CompileFieldPath("metadata.labels.origin")

	generator := id.NewGenerator()
	newSnapshotLog := func(t *testing.T, params snapshotLogParams) *log.Log {
		t.Helper()
		window := map[string]any{}
		if !params.windowStartTime.IsZero() {
			window["startTime"] = params.windowStartTime.Format(time.RFC3339Nano)
		}
		if !params.windowEndTime.IsZero() {
			window["endTime"] = params.windowEndTime.Format(time.RFC3339Nano)
		}
		metadata := map[string]any{
			"name":   params.name,
			"labels": map[string]any{"origin": params.label},
		}
		if params.namespace != "" {
			metadata["namespace"] = params.namespace
		}
		node, err := structured.FromGoValue(map[string]any{
			"window":  window,
			"deleted": params.isDeleted,
			"asset": map[string]any{
				"name":      params.assetName,
				"assetType": params.assetType,
				"resource": map[string]any{
					"data": map[string]any{
						"apiVersion": "v1",
						"kind":       params.kind,
						"metadata":   metadata,
					},
				},
			},
		}, &structured.AlphabeticalGoMapKeyOrderProvider{})
		if err != nil {
			t.Fatalf("failed to build node: %v", err)
		}
		return log.NewLogWithTimestamp(generator, structured.NewNodeReader(node), params.windowStartTime)
	}

	podParams := snapshotLogParams{
		assetName: "//container.googleapis.com/projects/p/locations/us-central1-a/clusters/c/k8s/namespaces/default/pods/pod-1",
		assetType: "k8s.io/Pod",
		kind:      "Pod",
		namespace: "default",
		name:      "pod-1",
	}
	nodeParams := snapshotLogParams{
		assetName: "//container.googleapis.com/projects/p/locations/us-central1-a/clusters/c/k8s/nodes/node-1",
		assetType: "k8s.io/Node",
		kind:      "Node",
		name:      "node-1",
	}

	withWindow := func(params snapshotLogParams, label string, startOffset, endOffset time.Duration) snapshotLogParams {
		params.label = label
		params.windowStartTime = queryStartTime.Add(startOffset)
		if endOffset != 0 {
			params.windowEndTime = queryStartTime.Add(endOffset)
		}
		return params
	}

	testCases := []struct {
		name string
		logs []snapshotLogParams
		// lookup is the identity the audit log pipeline asks the provider about.
		lookup    *k8saudit.ResourceIdentity
		wantFound bool
		// wantLabel is the metadata.labels.origin value of the manifest the provider must report.
		wantLabel string
	}{
		{
			name: "reports the snapshot that was current at the inspection start",
			logs: []snapshotLogParams{withWindow(podParams, "active", -time.Hour, time.Hour)},
			lookup: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "pod-1",
			},
			wantFound: true,
			wantLabel: "active",
		},
		{
			name: "ignores a snapshot that ended before the inspection start",
			logs: []snapshotLogParams{withWindow(podParams, "ended", -2*time.Hour, -time.Hour)},
			lookup: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "pod-1",
			},
			wantFound: false,
		},
		{
			name: "ignores a snapshot that became current after the inspection start",
			logs: []snapshotLogParams{withWindow(podParams, "future", time.Hour, 2*time.Hour)},
			lookup: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "pod-1",
			},
			wantFound: false,
		},
		{
			name: "ignores a tombstone snapshot",
			logs: []snapshotLogParams{func() snapshotLogParams {
				params := withWindow(podParams, "deleted", -time.Hour, time.Hour)
				params.isDeleted = true
				return params
			}()},
			lookup: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "pod-1",
			},
			wantFound: false,
		},
		{
			name: "indexes a cluster scoped resource under the cluster scope namespace",
			logs: []snapshotLogParams{withWindow(nodeParams, "active", -time.Hour, time.Hour)},
			lookup: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "node",
				Namespace:  k8saudit.ClusterScopeNamespace,
				Name:       "node-1",
			},
			wantFound: true,
			wantLabel: "active",
		},
		{
			name: "resolves a cluster scoped resource when queried with an empty namespace",
			logs: []snapshotLogParams{withWindow(nodeParams, "active", -time.Hour, time.Hour)},
			lookup: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "node",
				Namespace:  "",
				Name:       "node-1",
			},
			wantFound: true,
			wantLabel: "active",
		},
		{
			name: "keeps the snapshot that became current last when several qualify",
			logs: []snapshotLogParams{
				withWindow(podParams, "newer", -time.Hour, time.Hour),
				withWindow(podParams, "older", -2*time.Hour, time.Hour),
			},
			lookup: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "pod-1",
			},
			wantFound: true,
			wantLabel: "newer",
		},
		{
			name: "overwrites an older snapshot when a newer one is processed later",
			logs: []snapshotLogParams{
				withWindow(podParams, "older", -2*time.Hour, time.Hour),
				withWindow(podParams, "newer", -time.Hour, time.Hour),
			},
			lookup: &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "pod-1",
			},
			wantFound: true,
			wantLabel: "newer",
		},
		{
			name: "reports nothing for a subresource the inventory does not carry",
			logs: []snapshotLogParams{withWindow(podParams, "active", -time.Hour, time.Hour)},
			lookup: &k8saudit.ResourceIdentity{
				APIVersion:      "core/v1",
				Kind:            "pod",
				Namespace:       "default",
				Name:            "pod-1",
				SubresourceName: "status",
			},
			wantFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logs := make([]*log.Log, 0, len(tc.logs))
			for _, params := range tc.logs {
				logs = append(logs, newSnapshotLog(t, params))
			}

			provider := newCAIInitialResourceStateProvider(logs, queryStartTime)
			body, found := provider.InitialResourceState(tc.lookup)
			if found != tc.wantFound {
				t.Fatalf("InitialResourceState(%v) found = %t, want %t", tc.lookup, found, tc.wantFound)
			}
			if !tc.wantFound {
				return
			}
			if got := body.ReadStringOrDefault(pathMetadataLabelsOrigin, ""); got != tc.wantLabel {
				t.Errorf("metadata.labels.origin = %q, want %q", got, tc.wantLabel)
			}
		})
	}
}
