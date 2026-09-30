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

package mcp

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"cloud.google.com/go/auth"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// createInspection creates a gcp-gke inspection, optionally enables only the given features, and returns its ID.
func (e *toolsTestEnv) createInspection(t *testing.T, featureIDs ...string) string {
	t.Helper()
	text, isError := callTool(t, e.ctx, e.session, "create_inspection", map[string]any{
		"inspectionTypeId": "gcp-gke",
	})
	if isError {
		t.Fatalf("create_inspection returned error: %s", text)
	}
	id := extractInspectionID(t, text)
	if len(featureIDs) > 0 {
		text, isError := callTool(t, e.ctx, e.session, "update_inspection_features", map[string]any{
			"inspectionId":      id,
			"enabledFeatureIds": featureIDs,
		})
		if isError {
			t.Fatalf("update_inspection_features returned error: %s", text)
		}
	}
	return id
}

func TestInspectionRun_Golden(t *testing.T) {
	server, err := coreinspection.NewServer(nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	handler := NewInspectionHandler(server)

	testCases := []struct {
		name       string
		template   string
		goldenFile string
		data       any
	}{
		{
			name:       "DryRunInspection",
			template:   "dry_run_inspection.md.tmpl",
			goldenFile: "testdata/dry_run_inspection.golden.md",
			data: dryRunData{
				ID:           "2026-09-24-0130-a1b2",
				ErrorCount:   1,
				WarningCount: 0,
				Groups: []formGroupData{
					{
						Title: "Target",
						Fields: []formFieldData{
							{
								ID:          "clusterName",
								Label:       "Cluster name",
								Description: "The name of the GKE cluster.",
								Type:        "text",
								Value:       "`prod-cluster-1`",
								Suggestions: "`prod-cluster-1`, `staging-cluster`",
							},
							{
								ID:          "location",
								Label:       "Location",
								Description: "The region or zone of the cluster.",
								Type:        "text",
								Suggestions: "`us-central1`",
								HintType:    "Error",
								Hint:        "Location is required.",
							},
						},
					},
					{
						Title: "Time range",
						Fields: []formFieldData{
							{
								ID:          "endTime",
								Label:       "End time",
								Description: "The end of the query range in RFC3339 format.",
								Type:        "text",
								Value:       "`2026-09-24T02:00:00Z`",
								Default:     "`2026-09-24T06:40:00Z`",
							},
						},
					},
				},
				Queries: []queryData{
					{
						ID:             "k8s-audit",
						Name:           "K8s audit logs",
						EstimatedCount: "152000",
						Query:          "resource.type=\"k8s_cluster\"\nresource.labels.cluster_name=\"prod-cluster-1\"",
					},
				},
			},
		},
		{
			name:       "RunInspection",
			template:   "run_inspection.md.tmpl",
			goldenFile: "testdata/run_inspection.golden.md",
			data: inspectionIDData{
				ID: "2026-09-24-0130-a1b2",
			},
		},
		{
			name:       "GoogleCloudAuthError",
			template:   "google_cloud_auth_error.md.tmpl",
			goldenFile: "testdata/google_cloud_auth_error.golden.md",
			data: googleCloudAuthErrorData{
				RetryTool: "dry_run_inspection",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := handler.templates.Render(tc.template, tc.data)
			if err != nil {
				t.Fatalf("failed to render %s: %v", tc.template, err)
			}

			want := readGolden(t, tc.goldenFile)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("%s mismatch (-want +got):\n%s", tc.template, diff)
			}
		})
	}
}

func TestConvertMCPParameters(t *testing.T) {
	testCases := []struct {
		name  string
		input map[string]any
		want  map[string]any
		// wantErrKey is the key expected in the returned invalidParameterError. Empty means no error.
		wantErrKey string
	}{
		{
			name:  "nil value is skipped",
			input: map[string]any{"skipped": nil, "kept": "value"},
			want:  map[string]any{"kept": "value"},
		},
		{
			name:  "string and bool are kept as is",
			input: map[string]any{"str": "hello", "flag": true},
			want:  map[string]any{"str": "hello", "flag": true},
		},
		{
			name:  "array of strings becomes string slice",
			input: map[string]any{"list": []any{"a", "b"}},
			want:  map[string]any{"list": []string{"a", "b"}},
		},
		{
			name:       "number returns error",
			input:      map[string]any{"count": float64(42)},
			wantErrKey: "count",
		},
		{
			name:       "non-string array element returns error",
			input:      map[string]any{"list": []any{"a", float64(10)}},
			wantErrKey: "list",
		},
		{
			name:       "nested map returns error",
			input:      map[string]any{"nested": map[string]any{"key": "value"}},
			wantErrKey: "nested",
		},
		{
			name:       "nested array returns error",
			input:      map[string]any{"nested": []any{[]any{"value"}}},
			wantErrKey: "nested",
		},
		{
			name: "first invalid key in sorted order is reported",
			input: map[string]any{
				"b": map[string]any{"key": "value"},
				"a": map[string]any{"key": "value"},
			},
			wantErrKey: "a",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertMCPParameters(tc.input)
			if tc.wantErrKey != "" {
				var invErr *invalidParameterError
				if !errors.As(err, &invErr) {
					t.Fatalf("convertMCPParameters() error = %v, want *invalidParameterError", err)
				}
				if invErr.Key != tc.wantErrKey {
					t.Errorf("invalidParameterError.Key = %q, want %q", invErr.Key, tc.wantErrKey)
				}
				return
			}
			if err != nil {
				t.Fatalf("convertMCPParameters() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("convertMCPParameters() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroupFormFields(t *testing.T) {
	type groupResult struct {
		Groups   []formGroupData
		Errors   int
		Warnings int
	}

	testCases := []struct {
		name   string
		fields []inspectionmetadata.ParameterFormField
		params map[string]any
		want   groupResult
	}{
		{
			name: "text field with value, default and suggestions",
			fields: []inspectionmetadata.ParameterFormField{
				inspectionmetadata.TextParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:          "clusterName",
						Label:       "Cluster name",
						Description: "The name of the GKE cluster.",
						Type:        inspectionmetadata.Text,
					},
					Default:     "default-cluster",
					Suggestions: []string{"prod-cluster-1", "staging-cluster"},
				},
			},
			params: map[string]any{"clusterName": "prod-cluster-1"},
			want: groupResult{
				Groups: []formGroupData{
					{
						Title: "General",
						Fields: []formFieldData{
							{
								ID:          "clusterName",
								Label:       "Cluster name",
								Description: "The name of the GKE cluster.",
								Type:        "text",
								Value:       "`prod-cluster-1`",
								Default:     "`default-cluster`",
								Suggestions: "`prod-cluster-1`, `staging-cluster`",
							},
						},
					},
				},
			},
		},
		{
			name: "set field with options and default",
			fields: []inspectionmetadata.ParameterFormField{
				inspectionmetadata.GroupParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:    "filters",
						Label: "Filters",
						Type:  inspectionmetadata.Group,
					},
					Children: []inspectionmetadata.ParameterFormField{
						inspectionmetadata.SetParameterFormField{
							ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
								ID:    "namespaces",
								Label: "Namespaces",
								Type:  inspectionmetadata.Set,
							},
							Options: []inspectionmetadata.SetParameterFormFieldOptionItem{
								{ID: "default"},
								{ID: "kube-system"},
							},
							Default: []string{"default"},
						},
					},
				},
			},
			params: map[string]any{"namespaces": []string{"default", "kube-system"}},
			want: groupResult{
				Groups: []formGroupData{
					{
						Title: "Filters",
						Fields: []formFieldData{
							{
								ID:      "namespaces",
								Label:   "Namespaces",
								Type:    "set",
								Value:   "`default`, `kube-system`",
								Default: "`default`",
								Options: "`default`, `kube-system`",
							},
						},
					},
				},
			},
		},
		{
			name: "checkbox field with default",
			fields: []inspectionmetadata.ParameterFormField{
				inspectionmetadata.CheckboxParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:    "includeEvents",
						Label: "Include events",
						Type:  inspectionmetadata.Checkbox,
					},
					Default: true,
				},
			},
			params: map[string]any{"includeEvents": false},
			want: groupResult{
				Groups: []formGroupData{
					{
						Title: "General",
						Fields: []formFieldData{
							{
								ID:      "includeEvents",
								Label:   "Include events",
								Type:    "checkbox",
								Value:   "`false`",
								Default: "`true`",
							},
						},
					},
				},
			},
		},
		{
			name: "file fields are skipped and empty groups are dropped",
			fields: []inspectionmetadata.ParameterFormField{
				inspectionmetadata.FileParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:   "topFile",
						Type: inspectionmetadata.File,
					},
				},
				inspectionmetadata.GroupParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:    "upload",
						Label: "Upload",
						Type:  inspectionmetadata.Group,
					},
					Children: []inspectionmetadata.ParameterFormField{
						inspectionmetadata.FileParameterFormField{
							ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
								ID:   "childFile",
								Type: inspectionmetadata.File,
							},
						},
					},
				},
			},
			params: map[string]any{},
			want:   groupResult{},
		},
		{
			name: "hints are rendered and errors and warnings are counted",
			fields: []inspectionmetadata.ParameterFormField{
				inspectionmetadata.GroupParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:    "target",
						Label: "Target",
						Type:  inspectionmetadata.Group,
					},
					Children: []inspectionmetadata.ParameterFormField{
						inspectionmetadata.TextParameterFormField{
							ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
								ID:       "info",
								Label:    "Info field",
								Type:     inspectionmetadata.Text,
								HintType: inspectionmetadata.Info,
								Hint:     "Info hint.",
							},
						},
						inspectionmetadata.TextParameterFormField{
							ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
								ID:       "warning",
								Label:    "Warning field",
								Type:     inspectionmetadata.Text,
								HintType: inspectionmetadata.Warning,
								Hint:     "Warning hint.",
							},
						},
						inspectionmetadata.TextParameterFormField{
							ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
								ID:       "error",
								Label:    "Error field",
								Type:     inspectionmetadata.Text,
								HintType: inspectionmetadata.Error,
								Hint:     "Error hint.",
							},
						},
						inspectionmetadata.TextParameterFormField{
							ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
								ID:       "none",
								Label:    "None field",
								Type:     inspectionmetadata.Text,
								HintType: inspectionmetadata.None,
								Hint:     "Suppressed hint.",
							},
						},
					},
				},
			},
			params: map[string]any{},
			want: groupResult{
				Groups: []formGroupData{
					{
						Title: "Target",
						Fields: []formFieldData{
							{ID: "info", Label: "Info field", Type: "text", HintType: "Info", Hint: "Info hint."},
							{ID: "warning", Label: "Warning field", Type: "text", HintType: "Warning", Hint: "Warning hint."},
							{ID: "error", Label: "Error field", Type: "text", HintType: "Error", Hint: "Error hint."},
							{ID: "none", Label: "None field", Type: "text"},
						},
					},
				},
				Errors:   1,
				Warnings: 1,
			},
		},
		{
			name: "ungrouped fields form the General group placed first",
			fields: []inspectionmetadata.ParameterFormField{
				inspectionmetadata.GroupParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:    "target",
						Label: "Target",
						Type:  inspectionmetadata.Group,
					},
					Children: []inspectionmetadata.ParameterFormField{
						inspectionmetadata.TextParameterFormField{
							ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
								ID:       "location",
								Label:    "Location",
								Type:     inspectionmetadata.Text,
								HintType: inspectionmetadata.Error,
								Hint:     "Location is required.",
							},
						},
					},
				},
				inspectionmetadata.TextParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:       "name",
						Label:    "Name",
						Type:     inspectionmetadata.Text,
						HintType: inspectionmetadata.Warning,
						Hint:     "Name is long.",
					},
				},
			},
			params: map[string]any{"name": "my inspection"},
			want: groupResult{
				Groups: []formGroupData{
					{
						Title: "General",
						Fields: []formFieldData{
							{ID: "name", Label: "Name", Type: "text", Value: "`my inspection`", HintType: "Warning", Hint: "Name is long."},
						},
					},
					{
						Title: "Target",
						Fields: []formFieldData{
							{ID: "location", Label: "Location", Type: "text", HintType: "Error", Hint: "Location is required."},
						},
					},
				},
				Errors:   1,
				Warnings: 1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			groups, errs, warns := groupFormFields(tc.fields, tc.params)
			got := groupResult{Groups: groups, Errors: errs, Warnings: warns}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("groupFormFields() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsGoogleCloudAuthError(t *testing.T) {
	taskErr := func(err error) error {
		return &coretask.TaskError{TaskID: taskid.NewDefaultImplementationID[any]("query-task"), Err: err}
	}
	tokenErr := func(statusCode int) error {
		return &url.Error{Op: "Post", URL: "https://oauth2.googleapis.com/token", Err: &auth.Error{
			Response: &http.Response{StatusCode: statusCode},
			Body:     []byte(`{"error":"invalid_grant"}`),
		}}
	}
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "gRPC unauthenticated status from a failed token fetch",
			err:  taskErr(status.Error(codes.Unauthenticated, "transport: per-RPC creds failed due to error: auth: cannot fetch token: 400")),
			want: true,
		},
		{
			name: "gRPC unauthenticated status wrapped with fmt.Errorf",
			err:  taskErr(fmt.Errorf("failed to list log entries: %w", status.Error(codes.Unauthenticated, "request had invalid authentication credentials"))),
			want: true,
		},
		{
			name: "REST token error with a 4xx response",
			err:  taskErr(fmt.Errorf("failed to list regions: %w", tokenErr(http.StatusBadRequest))),
			want: true,
		},
		{
			name: "REST token error with a 5xx response",
			err:  taskErr(tokenErr(http.StatusServiceUnavailable)),
			want: false,
		},
		{
			name: "default credentials missing",
			err:  taskErr(fmt.Errorf("credentials: could not find default credentials. See https://cloud.google.com/docs/authentication/external/set-up-adc for more information")),
			want: true,
		},
		{
			name: "gRPC permission denied status",
			err:  taskErr(status.Error(codes.PermissionDenied, "logging.logEntries.list denied")),
			want: false,
		},
		{
			name: "untyped invalid_grant message",
			err:  taskErr(errors.New("oauth2: cannot fetch token: 400 Bad Request invalid_grant")),
			want: false,
		},
		{
			name: "google_cloud_auth literal string",
			err:  taskErr(errors.New("google_cloud_auth error occurred")),
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isGoogleCloudAuthError(tc.err)
			if got != tc.want {
				t.Errorf("isGoogleCloudAuthError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// runInspection starts the inspection and fails the test if run_inspection does not report RUNNING.
func (e *toolsTestEnv) runInspection(t *testing.T, id string, params map[string]any) {
	t.Helper()
	text, isError := callTool(t, e.ctx, e.session, "run_inspection", map[string]any{
		"inspectionId": id,
		"parameters":   params,
	})
	want := fmt.Sprintf("# Started `%s`\n\nStatus: RUNNING. Call `wait_inspection` to wait for the result.", id)
	checkToolResult(t, "run_inspection", text, isError, want, false)
}

func TestInspectionTools_InvalidParameters(t *testing.T) {
	env := newToolsTestEnv(t)

	testCases := []struct {
		name   string
		params map[string]any
		want   string
	}{
		{
			name:   "required field is empty",
			params: map[string]any{},
			want:   "Error: INVALID_PARAMETERS\n\n- The inspection has parameter errors. Fix them and call `dry_run_inspection` again.\n- `cluster-name`: cluster name is required",
		},
		{
			name:   "unsupported parameter type",
			params: map[string]any{"cluster-name": map[string]any{"nested": "value"}},
			want:   "Error: INVALID_PARAMETERS\n\n- Parameter `cluster-name` must be a string, boolean, or an array of strings.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id := env.createInspection(t)
			text, isError := callTool(t, env.ctx, env.session, "run_inspection", map[string]any{
				"inspectionId": id,
				"parameters":   tc.params,
			})
			checkToolResult(t, "run_inspection", text, isError, tc.want, true)
		})
	}
}

func TestInspectionTools_DryRunInspection(t *testing.T) {
	env := newToolsTestEnv(t)
	id := env.createInspection(t)

	text, isError := callTool(t, env.ctx, env.session, "dry_run_inspection", map[string]any{
		"inspectionId": id,
		"parameters": map[string]any{
			"cluster-name": "prod-cluster-1",
		},
	})
	want := fmt.Sprintf("# Dry run of `%s`\n\n"+
		"Errors: 0, warnings: 0. Ready to run. Call `run_inspection` with the parameters to start the inspection.\n\n"+
		"## Fields\n\n"+
		"### General\n\n"+
		"#### `khi.google.com/inspection/input/inspection-name`\n- Label: Inspection name\n- Description: The display name of this inspection.\n- Type: text\n"+
		"- Value: `Google Kubernetes Engine`\n- Default: `Google Kubernetes Engine`\n\n"+
		"#### `cluster-name`\n- Label: Cluster Name\n- Type: text\n- Value: `prod-cluster-1`", id)
	checkToolResult(t, "dry_run_inspection", text, isError, want, false)
}

func TestInspectionTools_GoogleCloudAuthError(t *testing.T) {
	env := newToolsTestEnv(t)

	testCases := []struct {
		tool string
		want string
	}{
		{
			tool: "dry_run_inspection",
			want: readGolden(t, "testdata/google_cloud_auth_error.golden.md"),
		},
		{
			tool: "run_inspection",
			want: "Error: GOOGLE_CLOUD_AUTH\n\n" +
				"- The application default credentials have expired.\n" +
				"- Run this command in your local terminal:\n\n" +
				"  ```sh\n  gcloud auth application-default login\n  ```\n\n" +
				"- Then call `run_inspection` again.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.tool, func(t *testing.T) {
			id := env.createInspection(t, authFailFeatureID)
			text, isError := callTool(t, env.ctx, env.session, tc.tool, map[string]any{
				"inspectionId": id,
			})
			checkToolResult(t, tc.tool, text, isError, tc.want, true)
		})
	}
}
