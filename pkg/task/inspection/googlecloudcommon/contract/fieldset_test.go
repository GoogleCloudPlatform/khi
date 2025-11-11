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
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
)

func TestGCPAuditLogFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *GCPAuditLogFieldSet
	}{
		{
			desc: "basic input",
			input: `
operation:
  id: "12345"
  first: true
  last: false
protoPayload:
  methodName: "test.method"
  resourceName: "projects/123/resources/abc"
  authenticationInfo:
    principalEmail: "user@example.com"
  status:
    code: 200
`,
			want: &GCPAuditLogFieldSet{
				OperationID:    "12345",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "test.method",
				ResourceName:   "projects/123/resources/abc",
				PrincipalEmail: "user@example.com",
				Status:         200,
				Request:        nil,
				Response:       nil,
			},
		},
		{
			desc:  "default input",
			input: "{}",
			want: &GCPAuditLogFieldSet{
				OperationID:    "",
				OperationFirst: false,
				OperationLast:  false,
				MethodName:     "unknown",
				ResourceName:   "unknown",
				PrincipalEmail: "unknown",
				Status:         -1,
				Request:        nil,
				Response:       nil,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse YAML test input to log: %v", err)
			}
			err = l.SetFieldSetReader(&GCPOperationAuditLogFieldSetReader{})
			if err != nil {
				t.Errorf("failed to run GCPOperationAuditLogFieldSetReader.Read(): %v", err)
			}
			got := log.MustGetFieldSet(l, &GCPAuditLogFieldSet{})
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GCPOperationAuditLogFieldSet mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGCPAuditLogFieldSet_OperationMethods(t *testing.T) {
	operationPathParent := resourcepath.Node("node-foo")
	testCases := []struct {
		desc                   string
		input                  GCPAuditLogFieldSet
		wantStarting           bool
		wantEnding             bool
		wantImmediateOperation bool
		wantOperationPath      resourcepath.ResourcePath
	}{
		{
			desc: "operation started",
			input: GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "compute.instances.insert",
			},
			wantStarting:           true,
			wantEnding:             false,
			wantImmediateOperation: false,
			wantOperationPath:      resourcepath.Operation(operationPathParent, "insert", "op-1"),
		},
		{
			desc: "operation ended",
			input: GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "compute.instances.insert",
			},
			wantStarting:           false,
			wantEnding:             true,
			wantImmediateOperation: false,
			wantOperationPath:      resourcepath.Operation(operationPathParent, "insert", "op-1"),
		},
		{
			desc: "immediate operation",
			input: GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  true,
				MethodName:     "compute.instances.delete",
			},
			wantStarting:           false,
			wantEnding:             false,
			wantImmediateOperation: true,
			wantOperationPath:      operationPathParent,
		},
		{
			// this is not expected to happen
			desc: "neither start nor end",
			input: GCPAuditLogFieldSet{
				OperationID:    "op-3",
				OperationFirst: false,
				OperationLast:  false,
				MethodName:     "compute.instances.update",
			},
			wantStarting:           false,
			wantEnding:             false,
			wantImmediateOperation: true,
			wantOperationPath:      operationPathParent,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			gotStarting := tc.input.Starting()
			gotEnding := tc.input.Ending()
			gotImmediateOperation := tc.input.ImmediateOperation()
			gotOperationPath := tc.input.OperationPath(operationPathParent)

			if gotStarting != tc.wantStarting {
				t.Errorf("GCPAuditLogFieldSet.Starting() = %v, want %v", gotStarting, tc.wantStarting)
			}
			if gotEnding != tc.wantEnding {
				t.Errorf("GCPAuditLogFieldSet.Ending() = %v, want %v", gotEnding, tc.wantEnding)
			}
			if gotImmediateOperation != tc.wantImmediateOperation {
				t.Errorf("GCPAuditLogFieldSet.ImmediateOperation() = %v, want %v", gotImmediateOperation, tc.wantImmediateOperation)
			}
			if diff := cmp.Diff(tc.wantOperationPath, gotOperationPath); diff != "" {
				t.Errorf("GCPAuditLogFieldSet.OperationPath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGCPAuditLogFieldSet_RequestResponseString(t *testing.T) {
	testCases := []struct {
		desc            string
		input           string
		wantRequest     string
		wantRequestErr  error
		wantResponse    string
		wantResponseErr error
	}{
		{
			desc: "request and response present",
			input: `
protoPayload:
  request:
    foo: bar
  response:
    status: ok
`,
			wantRequest:     "foo: bar\n",
			wantRequestErr:  nil,
			wantResponse:    "status: ok\n",
			wantResponseErr: nil,
		},
		{
			desc: "request and response absent",
			input: `
protoPayload: {}
`,
			wantRequest:     "",
			wantRequestErr:  khierrors.ErrNotFound,
			wantResponse:    "",
			wantResponseErr: khierrors.ErrNotFound,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse YAML test input to log: %v", err)
			}
			err = l.SetFieldSetReader(&GCPOperationAuditLogFieldSetReader{})
			if err != nil {
				t.Errorf("failed to run GCPOperationAuditLogFieldSetReader.Read(): %v", err)
			}
			fieldSet := log.MustGetFieldSet(l, &GCPAuditLogFieldSet{})

			gotRequest, gotRequestErr := fieldSet.RequestString()
			gotResponse, gotResponseErr := fieldSet.ResponseString()

			if gotRequest != tc.wantRequest {
				t.Errorf("RequestString() got = %v, want %v", gotRequest, tc.wantRequest)
			}
			if tc.wantRequestErr != nil && !errors.Is(gotRequestErr, tc.wantRequestErr) {
				t.Errorf("RequestString() error got = %v, want %v", gotRequestErr, tc.wantRequestErr)
			}
			if gotResponse != tc.wantResponse {
				t.Errorf("ResponseString() got = %v, want %v", gotResponse, tc.wantResponse)
			}
			if tc.wantResponseErr != nil && !errors.Is(gotResponseErr, tc.wantResponseErr) {
				t.Errorf("ResponseString() error got = %v, want %v", gotResponseErr, tc.wantResponseErr)
			}
		})
	}

}

func TestGCPAccessLogFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *GCPAccessLogFieldSet
	}{
		{
			desc: "basic input",
			input: `
httpRequest:
  requestMethod: "GET"
  requestUrl: "/path/to/resource"
  requestSize: "1234"
  status: 200
  responseSize: "5678"
  userAgent: "test-agent"
  remoteIp: "192.168.1.1"
  serverIp: "10.0.0.1"
  referer: "http://example.com"
  latency: "1s"
  protocol: "HTTP/1.1"
`,
			want: &GCPAccessLogFieldSet{
				Method:       "GET",
				RequestURL:   "/path/to/resource",
				RequestSize:  1234,
				Status:       200,
				ResponseSize: 5678,
				UserAgent:    "test-agent",
				RemoteIP:     "192.168.1.1",
				ServerIP:     "10.0.0.1",
				Referer:      "http://example.com",
				Latency:      "1s",
				Protocol:     "HTTP/1.1",
			},
		},
		{
			desc:  "default input",
			input: "{}",
			want: &GCPAccessLogFieldSet{
				Method:       "",
				RequestURL:   "",
				RequestSize:  0,
				Status:       0,
				ResponseSize: 0,
				UserAgent:    "",
				RemoteIP:     "",
				ServerIP:     "",
				Referer:      "",
				Latency:      "",
				Protocol:     "",
			},
		},
		{
			desc: "input with non-numeric sizes",
			input: `
httpRequest:
  requestMethod: "GET"
  requestUrl: "/path"
  requestSize: "invalid"
  status: 200
  responseSize: "not-a-number"
`,
			want: &GCPAccessLogFieldSet{
				Method:       "GET",
				RequestURL:   "/path",
				RequestSize:  0,
				Status:       200,
				ResponseSize: 0,
				UserAgent:    "",
				RemoteIP:     "",
				ServerIP:     "",
				Referer:      "",
				Latency:      "",
				Protocol:     "",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse YAML test input to log: %v", err)
			}
			err = l.SetFieldSetReader(&GCPAccessLogFieldSetReader{})
			if err != nil {
				t.Errorf("failed to run GCPAccessLogFieldSetReader.Read(): %v", err)
			}
			got := log.MustGetFieldSet(l, &GCPAccessLogFieldSet{})
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GCPAccessLogFieldSet mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
