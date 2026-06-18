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

package googlecloudcommon_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func TestGCPAuditLogFieldSetReader(t *testing.T) {
	reader := &GCPOperationAuditLogFieldSetReader{}
	tests := []struct {
		name  string
		input map[string]any
		want  *GCPAuditLogFieldSet
	}{
		{
			name: "full audit log",
			input: map[string]any{
				"resource": map[string]any{
					"labels": map[string]any{
						"project_id": "p1",
					},
				},
				"operation": map[string]any{
					"id":    "op-1",
					"first": true,
					"last":  false,
				},
				"protoPayload": map[string]any{
					"methodName":   "google.compute.v1.Instances.Insert",
					"resourceName": "projects/p1/zones/z1/instances/i1",
					"authenticationInfo": map[string]any{
						"principalEmail": "user@example.com",
					},
					"status": map[string]any{
						"code": 0,
					},
					"request": map[string]any{
						"name": "i1",
					},
					"response": map[string]any{
						"id": "123",
					},
				},
			},
			want: &GCPAuditLogFieldSet{
				ProjectID:      "p1",
				OperationID:    "op-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.compute.v1.Instances.Insert",
				ResourceName:   "projects/p1/zones/z1/instances/i1",
				PrincipalEmail: "user@example.com",
				Status:         0,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node, err := structured.FromGoValue(tc.input, &structured.AlphabeticalGoMapKeyOrderProvider{})
			if err != nil {
				t.Fatalf("failed to create node: %v", err)
			}
			nodeReader := structured.NewNodeReader(node)
			got, err := reader.Read(nodeReader)
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			gotAudit := got.(*GCPAuditLogFieldSet)

			// Compare fields except NodeReaders
			if gotAudit.ProjectID != tc.want.ProjectID ||
				gotAudit.OperationID != tc.want.OperationID ||
				gotAudit.OperationFirst != tc.want.OperationFirst ||
				gotAudit.OperationLast != tc.want.OperationLast ||
				gotAudit.MethodName != tc.want.MethodName ||
				gotAudit.ResourceName != tc.want.ResourceName ||
				gotAudit.PrincipalEmail != tc.want.PrincipalEmail ||
				gotAudit.Status != tc.want.Status {
				t.Errorf("Read() mismatch.\ngot:  %+v\nwant: %+v", gotAudit, tc.want)
			}
		})
	}
}

func TestGCPAuditLogFieldSet_GuessRevisionVerb(t *testing.T) {
	tests := []struct {
		name       string
		methodName string
		want       *pb.Verb
	}{
		{"Create", "google.compute.v1.Instances.Create", commonlogk8saudit_contract.VerbCreate},
		{"Insert", "google.compute.v1.BackendService.Insert", commonlogk8saudit_contract.VerbCreate},
		{"Update", "google.compute.v1.Instances.Update", commonlogk8saudit_contract.VerbUpdate},
		{"Patch", "google.compute.v1.Instances.Patch", commonlogk8saudit_contract.VerbUpdate},
		{"Delete", "google.compute.v1.Instances.Delete", commonlogk8saudit_contract.VerbDelete},
		{"Unknown", "google.compute.v1.Instances.Get", commonlogk8saudit_contract.VerbUpdate},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &GCPAuditLogFieldSet{MethodName: tc.methodName}
			if got := g.GuessRevisionVerb(); got != tc.want {
				t.Errorf("GuessRevisionVerb() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGCPDefaultSeverityFieldSetReader(t *testing.T) {
	reader := &GCPDefaultSeverityFieldSetReader{}
	tests := []struct {
		name  string
		input map[string]any
		want  *inspectioncore_contract.DefaultSeverityFieldSet
	}{
		{
			name: "severity info",
			input: map[string]any{
				"severity": "INFO",
			},
			want: &inspectioncore_contract.DefaultSeverityFieldSet{
				Severity: inspectioncore_contract.SeverityInfo,
			},
		},
		{
			name:  "severity absent defaults to empty string which is Unknown",
			input: map[string]any{},
			want: &inspectioncore_contract.DefaultSeverityFieldSet{
				Severity: inspectioncore_contract.SeverityUnknown,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node, err := structured.FromGoValue(tc.input, &structured.AlphabeticalGoMapKeyOrderProvider{})
			if err != nil {
				t.Fatalf("failed to create node: %v", err)
			}
			nodeReader := structured.NewNodeReader(node)
			gotFS, err := reader.Read(nodeReader)
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			got := gotFS.(*inspectioncore_contract.DefaultSeverityFieldSet)
			if got.Severity != tc.want.Severity {
				t.Errorf("Read() severity = %v, want %v", got.Severity, tc.want.Severity)
			}
		})
	}
}

func TestGCPMainMessageFieldSet(t *testing.T) {
	testCase := []struct {
		Name                string
		ExpectedMainMessage string
		InputYAML           string
	}{
		{
			Name:                "from textPayload field",
			ExpectedMainMessage: "foo",
			InputYAML:           `textPayload: foo`,
		},
		{
			Name:                "from jsonPayload.message field",
			ExpectedMainMessage: "bar",
			InputYAML: `jsonPayload:
  message: bar`,
		},
		{
			Name:                "from jsonPayload.MESSAGE field",
			ExpectedMainMessage: "bar",
			InputYAML: `jsonPayload:
  MESSAGE: bar`,
		},
		{
			Name:                "from jsonPayload.msg field",
			ExpectedMainMessage: "bar",
			InputYAML: `jsonPayload:
  msg: bar`,
		},
		{
			Name:                "from jsonPayload.log field",
			ExpectedMainMessage: "bar",
			InputYAML: `jsonPayload:
  log: bar`,
		},
		{
			Name:                "from the whole jsonPayload field",
			ExpectedMainMessage: `{"foo":"bar"}`,
			InputYAML: `jsonPayload:
  foo: bar`,
		},
		{
			Name:                "from the whole labels field",
			ExpectedMainMessage: `{"foo":"bar"}`,
			InputYAML: `labels:
  foo: bar`,
		},
		{
			Name:                "ignore when the message is protoPayload even labels are provided",
			ExpectedMainMessage: "",
			InputYAML: `labels:
  foo: bar
protoPayload:
  qux: quux`,
		},
		{
			Name:                "empty if no proper field is given",
			ExpectedMainMessage: "",
			InputYAML:           `foo: bar`,
		},
		{
			Name:                "prioritize textPayload rather than jsonPayload.msg or labels",
			ExpectedMainMessage: "bar",
			InputYAML: `jsonPayload:
  msg: foo
textPayload: bar
labels:
  qux: quux`,
		},
		{
			Name:                "prioritize jsonPayload.msg over labels",
			ExpectedMainMessage: "foo",
			InputYAML: `jsonPayload:
  msg: foo
labels:
  qux: quux`,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.InputYAML)
			if err != nil {
				t.Fatalf("failed to parse log from yaml: %v", err)
			}
			l.SetFieldSetReader(&GCPMainMessageFieldSetReader{})
			gcpMainMessageField, err := log.GetFieldSet(l, &GCPMainMessageFieldSet{})
			if err != nil {
				t.Fatalf("failed to extract gcp main message field: %v", err)
			}
			if gcpMainMessageField.MainMessage != tc.ExpectedMainMessage {
				t.Errorf("expected main message: %v, got: %v", tc.ExpectedMainMessage, gcpMainMessageField.MainMessage)
			}
		})
	}

}
