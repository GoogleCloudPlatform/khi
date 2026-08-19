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

package apiv1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
)

func TestCELValidationServer_ValidateTimelineQuery(t *testing.T) {
	server := NewCELValidationServer()

	testCases := []struct {
		name      string
		query     string
		wantValid bool
	}{
		{
			name:      "empty query is valid",
			query:     "",
			wantValid: true,
		},
		{
			name:      "valid query with match function",
			query:     `match("Pod", ".*") && minSeverity(INFO)`,
			wantValid: true,
		},
		{
			name:      "invalid query syntax",
			query:     `name == &&`,
			wantValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := server.ValidateTimelineQuery(context.Background(), connect.NewRequest(&v1.ValidateTimelineQueryRequest{
				Query: proto.String(tc.query),
			}))
			if err != nil {
				t.Fatalf("ValidateTimelineQuery() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.wantValid, res.Msg.GetValid()); diff != "" {
				t.Errorf("ValidateTimelineQuery() valid mismatch (-want +got):\n%s", diff)
			}
			if tc.wantValid && res.Msg.GetErrorMessage() != "" {
				t.Errorf("ValidateTimelineQuery() expected empty error message, got %q", res.Msg.GetErrorMessage())
			}
			if !tc.wantValid && res.Msg.GetErrorMessage() == "" {
				t.Errorf("ValidateTimelineQuery() expected error message, got empty string")
			}
		})
	}
}

func TestCELValidationServer_ValidateLogQuery(t *testing.T) {
	server := NewCELValidationServer()

	testCases := []struct {
		name      string
		query     string
		wantValid bool
	}{
		{
			name:      "empty query is valid",
			query:     "",
			wantValid: true,
		},
		{
			name:      "valid query with body function",
			query:     `severity >= INFO && body("verb", "create")`,
			wantValid: true,
		},
		{
			name:      "invalid query syntax",
			query:     `severity >= `,
			wantValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := server.ValidateLogQuery(context.Background(), connect.NewRequest(&v1.ValidateLogQueryRequest{
				Query: proto.String(tc.query),
			}))
			if err != nil {
				t.Fatalf("ValidateLogQuery() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.wantValid, res.Msg.GetValid()); diff != "" {
				t.Errorf("ValidateLogQuery() valid mismatch (-want +got):\n%s", diff)
			}
			if tc.wantValid && res.Msg.GetErrorMessage() != "" {
				t.Errorf("ValidateLogQuery() expected empty error message, got %q", res.Msg.GetErrorMessage())
			}
			if !tc.wantValid && res.Msg.GetErrorMessage() == "" {
				t.Errorf("ValidateLogQuery() expected error message, got empty string")
			}
		})
	}
}
