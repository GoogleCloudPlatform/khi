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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestOSSK8sAuditLogFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *commonlogk8sauditv2_contract.K8sAuditLogFieldSet
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
			want: &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
				OperationID:   "test-audit-id",
				IsFirst:       true,
				IsLast:        true,
				Principal:     "test-user",
				StatusCode:    200,
				StatusMessage: "OK",
				IsError:       false,
				RequestURI:    "/api/v1/namespaces/default/pods/test-pod",
				K8sOperation: &model.KubernetesObjectOperation{
					APIVersion:      "core/v1",
					PluralKind:      "pods",
					Namespace:       "default",
					Name:            "test-pod",
					SubResourceName: "",
					Verb:            enum.RevisionVerbCreate,
				},
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
			want: &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
				OperationID:   "test-audit-id-2",
				IsFirst:       true,
				IsLast:        true,
				Principal:     "unknown",
				StatusCode:    201,
				StatusMessage: "",
				IsError:       false,
				RequestURI:    "",
				K8sOperation: &model.KubernetesObjectOperation{
					APIVersion:      "apps/v1",
					PluralKind:      "deployments",
					Namespace:       "default",
					Name:            "generated-deployment-name",
					SubResourceName: "",
					Verb:            enum.RevisionVerbCreate,
				},
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
			want: &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
				OperationID:   "error-audit-id",
				IsFirst:       true,
				IsLast:        true,
				Principal:     "unknown",
				StatusCode:    404,
				StatusMessage: "Not Found",
				IsError:       true,
				RequestURI:    "",
				K8sOperation: &model.KubernetesObjectOperation{
					APIVersion:      "core/unknown",
					PluralKind:      "pods",
					Namespace:       "cluster-scope",
					Name:            "missing-pod",
					SubResourceName: "",
					Verb:            enum.RevisionVerbUpdate,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse YAML test input to log: %v", err)
			}
			err = l.SetFieldSetReader(&OSSK8sAuditLogFieldSetReader{})
			if err != nil {
				t.Errorf("failed to run OSSK8sAuditLogFieldSetReader.Read(): %v", err)
			}
			got := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})

			// Ignore Request and Response fields for now as they are NodeReaders and hard to compare directly with cmp.Diff without custom options
			opts := []cmp.Option{
				cmpopts.IgnoreFields(commonlogk8sauditv2_contract.K8sAuditLogFieldSet{}, "Request", "Response"),
			}

			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Errorf("OSSK8sAuditLogFieldSet mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOSSK8sAuditLogCommonFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *log.CommonFieldSet
	}{
		{
			desc: "basic fields",
			input: `
auditID: "test-audit-id"
stageTimestamp: "2023-10-26T10:00:00Z"
`,
			want: &log.CommonFieldSet{
				DisplayID: "test-audit-id",
				Timestamp: time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC),
				Severity:  enum.SeverityUnknown,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse YAML test input to log: %v", err)
			}
			err = l.SetFieldSetReader(&OSSK8sAuditLogCommonFieldSetReader{})
			if err != nil {
				t.Errorf("failed to run OSSK8sAuditLogCommonFieldSetReader.Read(): %v", err)
			}
			got := log.MustGetFieldSet(l, &log.CommonFieldSet{})
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("CommonFieldSet mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
