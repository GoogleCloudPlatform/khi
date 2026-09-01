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

package ossclusterk8s_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestExtractOSSK8sAuditLog(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  commonlogk8saudit_contract.K8sAuditLogFieldSet
	}{
		{
			desc: "standard operation",
			input: `
auditID: "test-audit-id"
verb: "create"
user:
  username: "test-user"
responseStatus:
  code: 200
  message: "OK"
objectRef:
  apiGroup: "core"
  apiVersion: "v1"
  resource: "pods"
  namespace: "default"
  name: "test-pod"
requestURI: "/api/v1/namespaces/default/pods/test-pod"
`,
			want: commonlogk8saudit_contract.K8sAuditLogFieldSet{
				OperationID:     "test-audit-id",
				IsFirst:         true,
				IsLast:          true,
				Principal:       "test-user",
				StatusCode:      200,
				StatusMessage:   "OK",
				IsError:         false,
				RequestURI:      "/api/v1/namespaces/default/pods/test-pod",
				APIVersion:      "core/v1",
				PluralKind:      "pods",
				Namespace:       "default",
				ResourceName:    "test-pod",
				SubresourceName: "",
				ClusterName:     "cluster",
				Verb:            commonlogk8saudit_contract.VerbCreate,
			},
		},
		{
			desc: "server generated name",
			input: `
auditID: "test-audit-id-2"
verb: "create"
objectRef:
  apiGroup: "apps"
  apiVersion: "v1"
  resource: "deployments"
  namespace: "default"
responseObject:
  metadata:
    name: "generated-deployment-name"
responseStatus:
  code: 201
`,
			want: commonlogk8saudit_contract.K8sAuditLogFieldSet{
				OperationID:     "test-audit-id-2",
				IsFirst:         true,
				IsLast:          true,
				Principal:       "unknown",
				StatusCode:      201,
				StatusMessage:   "",
				IsError:         false,
				RequestURI:      "",
				APIVersion:      "apps/v1",
				PluralKind:      "deployments",
				Namespace:       "default",
				ResourceName:    "generated-deployment-name",
				SubresourceName: "",
				ClusterName:     "cluster",
				Verb:            commonlogk8saudit_contract.VerbCreate,
			},
		},
		{
			desc: "error status",
			input: `
auditID: "error-audit-id"
verb: "update"
responseStatus:
  code: 404
  message: "Not Found"
objectRef:
  resource: "pods"
  name: "missing-pod"
`,
			want: commonlogk8saudit_contract.K8sAuditLogFieldSet{
				OperationID:     "error-audit-id",
				IsFirst:         true,
				IsLast:          true,
				Principal:       "unknown",
				StatusCode:      404,
				StatusMessage:   "Not Found",
				IsError:         true,
				RequestURI:      "",
				APIVersion:      "core/unknown",
				PluralKind:      "pods",
				Namespace:       "cluster-scope",
				ResourceName:    "missing-pod",
				SubresourceName: "",
				ClusterName:     "cluster",
				Verb:            commonlogk8saudit_contract.VerbUpdate,
			},
		},
		{
			desc: "truncated log",
			input: `
auditID: "truncated-audit-id"
verb: "update"
annotations:
  audit.k8s.io/truncated: "true"
responseStatus:
  code: 200
  message: "OK"
objectRef:
  apiGroup: "core"
  apiVersion: "v1"
  resource: "pods"
  namespace: "default"
  name: "truncated-pod"
`,
			want: commonlogk8saudit_contract.K8sAuditLogFieldSet{
				OperationID:     "truncated-audit-id",
				IsFirst:         true,
				IsLast:          true,
				IsTruncated:     true,
				Principal:       "unknown",
				StatusCode:      200,
				StatusMessage:   "OK",
				IsError:         false,
				RequestURI:      "",
				APIVersion:      "core/v1",
				PluralKind:      "pods",
				Namespace:       "default",
				ResourceName:    "truncated-pod",
				SubresourceName: "",
				ClusterName:     "cluster",
				Verb:            commonlogk8saudit_contract.VerbUpdate,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := testlog.MustLogFromYAML(tc.input)
			got, err := ExtractOSSK8sAuditLog(l.NodeReader)
			if err != nil {
				t.Fatalf("ExtractOSSK8sAuditLog() returned unexpected error: %v", err)
			}

			// Ignore Request and Response fields for now as they are NodeReaders and hard to compare directly with cmp.Diff without custom options
			opts := []cmp.Option{
				cmpopts.IgnoreFields(commonlogk8saudit_contract.K8sAuditLogFieldSet{}, "Request", "Response"),
				protocmp.Transform(),
			}

			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Errorf("ExtractOSSK8sAuditLog() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("MockNode returns mock data directly", func(t *testing.T) {
		want := commonlogk8saudit_contract.K8sAuditLogFieldSet{
			ResourceName: "mock-pod",
			Namespace:    "mock-ns",
			PluralKind:   "pods",
			APIVersion:   "core/v1",
		}
		reader := structured.NewNodeReader(structured.NewMockNode(want))
		got, err := ExtractOSSK8sAuditLog(reader)
		if err != nil {
			t.Fatalf("ExtractOSSK8sAuditLog() returned error: %v", err)
		}
		opts := []cmp.Option{
			cmpopts.IgnoreFields(commonlogk8saudit_contract.K8sAuditLogFieldSet{}, "Request", "Response"),
			protocmp.Transform(),
		}
		if diff := cmp.Diff(want, got, opts...); diff != "" {
			t.Errorf("ExtractOSSK8sAuditLog() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestExtractOSSK8sEvent(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  OSSK8sEventFieldSet
	}{
		{
			desc: "standard event log",
			input: `
responseObject:
  involvedObject:
    apiVersion: "apps/v1"
    kind: "Deployment"
    namespace: "default"
    name: "test-deployment"
    subresource: "status"
  reason: "ScalingReplicaSet"
  message: "Scaled up replica set test-deployment-123 to 3"
`,
			want: OSSK8sEventFieldSet{
				APIVersion:   "apps/v1",
				ResourceKind: "deployment",
				Namespace:    "default",
				Resource:     "test-deployment",
				Subresource:  "status",
				Reason:       "ScalingReplicaSet",
				Message:      "Scaled up replica set test-deployment-123 to 3",
			},
		},
		{
			desc: "default core apiVersion",
			input: `
responseObject:
  involvedObject:
    apiVersion: "v1"
    kind: "Pod"
    namespace: "default"
    name: "test-pod"
  reason: "Scheduled"
  message: "Successfully assigned default/test-pod to node-1"
`,
			want: OSSK8sEventFieldSet{
				APIVersion:   "core/v1",
				ResourceKind: "pod",
				Namespace:    "default",
				Resource:     "test-pod",
				Subresource:  "",
				Reason:       "Scheduled",
				Message:      "Successfully assigned default/test-pod to node-1",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := testlog.MustLogFromYAML(tc.input)
			got, err := ExtractOSSK8sEvent(l.NodeReader)
			if err != nil {
				t.Fatalf("ExtractOSSK8sEvent() returned error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractOSSK8sEvent() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("MockNode returns mock data directly", func(t *testing.T) {
		want := OSSK8sEventFieldSet{
			APIVersion:   "core/v1",
			ResourceKind: "pod",
			Namespace:    "default",
			Resource:     "test-pod",
			Reason:       "Scheduled",
		}
		reader := structured.NewNodeReader(structured.NewMockNode(want))
		got, err := ExtractOSSK8sEvent(reader)
		if err != nil {
			t.Fatalf("ExtractOSSK8sEvent() returned error: %v", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ExtractOSSK8sEvent() mismatch (-want +got):\n%s", diff)
		}
	})
}
