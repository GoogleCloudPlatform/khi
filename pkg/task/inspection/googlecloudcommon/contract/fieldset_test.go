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
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
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
			if gotAudit.OperationID != tc.want.OperationID ||
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
		want       enum.RevisionVerb
	}{
		{"Create", "google.compute.v1.Instances.Create", enum.RevisionVerbCreate},
		{"Insert", "google.compute.v1.BackendService.Insert", enum.RevisionVerbCreate},
		{"Update", "google.compute.v1.Instances.Update", enum.RevisionVerbUpdate},
		{"Patch", "google.compute.v1.Instances.Patch", enum.RevisionVerbUpdate},
		{"Delete", "google.compute.v1.Instances.Delete", enum.RevisionVerbDelete},
		{"Unknown", "google.compute.v1.Instances.Get", enum.RevisionVerbUpdate},
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
