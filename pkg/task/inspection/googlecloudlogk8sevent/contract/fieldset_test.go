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

package googlecloudlogk8sevent_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
)

func TestGCPKubernetesEventFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *KubernetesEventFieldSet
	}{
		{
			desc: "with all parameters",
			input: `resource:
  labels:
    cluster_name: test-cluster
jsonPayload:
  kind: Event
  involvedObject:
    apiVersion: v1
    kind: Pod
    namespace: default
    name: test-pod
  reason: Scheduled
  message: Successfully assigned default/test-pod to node-1`,
			want: &KubernetesEventFieldSet{
				ClusterName:  "test-cluster",
				APIVersion:   "core/v1",
				ResourceKind: "pod",
				Namespace:    "default",
				Resource:     "test-pod",
				Reason:       "Scheduled",
				Message:      "Successfully assigned default/test-pod to node-1",
			},
		},
		{
			desc: "cluster-scoped event",
			input: `resource:
  labels:
    cluster_name: test-cluster
jsonPayload:
  kind: Event
  involvedObject:
    apiVersion: apps/v1
    kind: Deployment
    name: test-deployment
  reason: ScalingReplicaSet
  message: Scaled up replica set test-deployment-xyz to 3`,
			want: &KubernetesEventFieldSet{
				ClusterName:  "test-cluster",
				APIVersion:   "apps/v1",
				ResourceKind: "deployment",
				Namespace:    "cluster-scope",
				Resource:     "test-deployment",
				Reason:       "ScalingReplicaSet",
				Message:      "Scaled up replica set test-deployment-xyz to 3",
			},
		},
		{
			desc: "message comes from action",
			input: `resource:
  labels:
    cluster_name: test-cluster
jsonPayload:
  kind: Event
  involvedObject:
    apiVersion: apps/v1
    kind: Deployment
    name: test-deployment
  reason: ScalingReplicaSet
  action: Scaled up replica set test-deployment-xyz to 3`,
			want: &KubernetesEventFieldSet{
				ClusterName:  "test-cluster",
				APIVersion:   "apps/v1",
				ResourceKind: "deployment",
				Namespace:    "cluster-scope",
				Resource:     "test-deployment",
				Reason:       "ScalingReplicaSet",
				Message:      "Scaled up replica set test-deployment-xyz to 3",
			},
		},
		{
			desc: "non json payload",
			input: `resource:
  labels:
    cluster_name: test-cluster
textPayload: Event exporter started watching. Some events may have been lost up to this point.`,
			want: &KubernetesEventFieldSet{
				ClusterName:  "test-cluster",
				APIVersion:   "",
				ResourceKind: "",
				Namespace:    "",
				Resource:     "",
				Reason:       "",
				Message:      "Event exporter started watching. Some events may have been lost up to this point.",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse test YAML data: %v", err)
			}

			err = l.SetFieldSetReader(&GCPKubernetesEventFieldSetReader{})
			if err != nil {
				t.Fatalf("l.SetFieldSetReader failed: %v", err)
			}

			fieldSet := log.MustGetFieldSet(l, &KubernetesEventFieldSet{})
			if diff := cmp.Diff(tc.want, fieldSet); diff != "" {
				t.Errorf("GCPKubernetesEventFieldSetReader mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
