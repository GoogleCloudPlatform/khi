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
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestExtractGCPAuditLog(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]any
		want  GCPAuditLogFieldSet
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
			want: GCPAuditLogFieldSet{
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
		{
			name: "audit log with principalSubject",
			input: map[string]any{
				"resource": map[string]any{
					"labels": map[string]any{
						"project_id": "p1",
					},
				},
				"operation": map[string]any{
					"id":    "op-2",
					"first": true,
					"last":  false,
				},
				"protoPayload": map[string]any{
					"methodName":   "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
					"resourceName": "projects/p1/locations/us-central1/environments/env-1",
					"authenticationInfo": map[string]any{
						"principalSubject": "serviceAccount:khi-sa@proj.iam.gserviceaccount.com",
					},
					"status": map[string]any{},
				},
			},
			want: GCPAuditLogFieldSet{
				ProjectID:      "p1",
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
				ResourceName:   "projects/p1/locations/us-central1/environments/env-1",
				PrincipalEmail: "serviceAccount:khi-sa@proj.iam.gserviceaccount.com",
				Status:         -1,
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
			got, err := ExtractGCPAuditLog(nodeReader)
			if err != nil {
				t.Fatalf("ExtractGCPAuditLog() error = %v", err)
			}

			// Compare fields except NodeReaders
			gotComparable := GCPAuditLogFieldSet{
				ProjectID:      got.ProjectID,
				OperationID:    got.OperationID,
				OperationFirst: got.OperationFirst,
				OperationLast:  got.OperationLast,
				MethodName:     got.MethodName,
				ResourceName:   got.ResourceName,
				PrincipalEmail: got.PrincipalEmail,
				Status:         got.Status,
			}
			if diff := cmp.Diff(tc.want, gotComparable); diff != "" {
				t.Errorf("ExtractGCPAuditLog() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("mock node returns mock values", func(t *testing.T) {
		mockFS := GCPAuditLogFieldSet{
			ProjectID:   "mock-project",
			OperationID: "mock-op",
		}
		reader := structured.NewNodeReader(structured.NewMockNode(mockFS))
		got, err := ExtractGCPAuditLog(reader)
		if err != nil {
			t.Fatalf("ExtractGCPAuditLog() error = %v", err)
		}
		if diff := cmp.Diff(mockFS, got); diff != "" {
			t.Errorf("ExtractGCPAuditLog() mismatch (-want +got):\n%s", diff)
		}
	})
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

func TestExtractGCPSeverity(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]any
		want  *pb.Severity
	}{
		{
			name: "severity info",
			input: map[string]any{
				"severity": "INFO",
			},
			want: inspectioncore_contract.SeverityInfo,
		},
		{
			name:  "severity absent defaults to empty string which is Unknown",
			input: map[string]any{},
			want:  inspectioncore_contract.SeverityUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node, err := structured.FromGoValue(tc.input, &structured.AlphabeticalGoMapKeyOrderProvider{})
			if err != nil {
				t.Fatalf("failed to create node: %v", err)
			}
			nodeReader := structured.NewNodeReader(node)
			got, err := ExtractGCPSeverity(nodeReader)
			if err != nil {
				t.Fatalf("ExtractGCPSeverity() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("ExtractGCPSeverity() = %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("mock node returns mock severity", func(t *testing.T) {
		reader := structured.NewNodeReader(structured.NewMockNode(inspectioncore_contract.DefaultSeverityFieldSet{
			Severity: inspectioncore_contract.SeverityError,
		}))
		got, err := ExtractGCPSeverity(reader)
		if err != nil {
			t.Fatalf("ExtractGCPSeverity() error = %v", err)
		}
		if got != inspectioncore_contract.SeverityError {
			t.Errorf("ExtractGCPSeverity() = %v, want %v", got, inspectioncore_contract.SeverityError)
		}
	})
}

func TestExtractGCPMainMessage(t *testing.T) {
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
			node, err := structured.FromYAML(tc.InputYAML)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}
			nodeReader := structured.NewNodeReader(node)
			got, err := ExtractGCPMainMessage(nodeReader)
			if err != nil {
				t.Fatalf("ExtractGCPMainMessage() error = %v", err)
			}
			if got != tc.ExpectedMainMessage {
				t.Errorf("ExtractGCPMainMessage() = %v, want %v", got, tc.ExpectedMainMessage)
			}
		})
	}

	t.Run("mock node returns mock main message", func(t *testing.T) {
		reader := structured.NewNodeReader(structured.NewMockNode(GCPMainMessageFieldSet{
			MainMessage: "hello world",
		}))
		got, err := ExtractGCPMainMessage(reader)
		if err != nil {
			t.Fatalf("ExtractGCPMainMessage() error = %v", err)
		}
		if got != "hello world" {
			t.Errorf("ExtractGCPMainMessage() = %v, want %v", got, "hello world")
		}
	})
}
