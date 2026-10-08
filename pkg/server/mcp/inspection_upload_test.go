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
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/server/uploadurl"
	"github.com/google/go-cmp/cmp"
)

func TestRequestFileUpload_Golden(t *testing.T) {
	server, err := coreinspection.NewServer(nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	// The grant is fixed here, so the handler does not issue upload URLs.
	handler := NewInspectionHandler(server, nil)
	got, err := handler.templates.Render("request_file_upload.md.tmpl", uploadurl.Grant{
		URL:          "http://127.0.0.1:8080/api/v1/file-upload/ABCDEF",
		FieldID:      logFileFieldID,
		MaxSizeBytes: testMaxUploadSizeBytes,
		ExpiresAt:    time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("failed to render request_file_upload.md.tmpl: %v", err)
	}
	want := readGolden(t, "testdata/request_file_upload.golden.md")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("request_file_upload.md.tmpl mismatch (-want +got):\n%s", diff)
	}
}

func TestFindUploadTokenID(t *testing.T) {
	groups := []formGroupData{
		{
			Title: "General",
			Fields: []formFieldData{
				{ID: "name", Type: "text"},
			},
		},
		{
			Title: "Logs",
			Fields: []formFieldData{
				{ID: "audit-log", Type: "file", uploadTokenID: "inspection-1_task_audit-log"},
			},
		},
	}

	testCases := []struct {
		name      string
		fieldID   string
		wantID    string
		wantFound bool
	}{
		{
			name:      "file field",
			fieldID:   "audit-log",
			wantID:    "inspection-1_task_audit-log",
			wantFound: true,
		},
		{
			name:    "field that is not a file field",
			fieldID: "name",
		},
		{
			name:    "missing field",
			fieldID: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotFound := findUploadTokenID(groups, tc.fieldID)
			if gotID != tc.wantID || gotFound != tc.wantFound {
				t.Errorf("findUploadTokenID(%q) = (%q, %v), want (%q, %v)", tc.fieldID, gotID, gotFound, tc.wantID, tc.wantFound)
			}
		})
	}
}

// putFile sends body to an upload URL and returns the response status code and body.
func putFile(t *testing.T, url string, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest() failed: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s failed: %v", url, err)
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read the upload response: %v", err)
	}
	return res.StatusCode, strings.TrimRight(string(resBody), "\n")
}

// requestUploadURL calls request_file_upload for the log file field and returns the issued URL.
func (e *toolsTestEnv) requestUploadURL(t *testing.T, id string) string {
	t.Helper()
	text, isError := callTool(t, e.ctx, e.session, "request_file_upload", map[string]any{
		"inspectionId": id,
		"fieldId":      logFileFieldID,
	})
	if isError {
		t.Fatalf("request_file_upload returned error: %s", text)
	}
	urlPrefix := fmt.Sprintf("# Upload URL for `%s`\n\n- URL: `%s/api/v1/file-upload/", logFileFieldID, e.serverURL)
	if !strings.HasPrefix(text, urlPrefix) {
		t.Fatalf("request_file_upload result does not start with %q:\n%s", urlPrefix, text)
	}
	token, _, found := strings.Cut(strings.TrimPrefix(text, urlPrefix), "`")
	if !found {
		t.Fatalf("request_file_upload result has no closing backtick after the URL:\n%s", text)
	}
	return e.serverURL + "/api/v1/file-upload/" + token
}

// dryRunUntilUploadSettles calls dry_run_inspection until the log file field leaves PROCESSING.
// Verification of an uploaded file runs asynchronously, so the first dry runs after an upload may still report it.
func (e *toolsTestEnv) dryRunUntilUploadSettles(t *testing.T, id string) string {
	t.Helper()
	for {
		text, isError := callTool(t, e.ctx, e.session, "dry_run_inspection", map[string]any{"inspectionId": id})
		if isError {
			t.Fatalf("dry_run_inspection returned error: %s", text)
		}
		if !strings.Contains(text, "- Upload: PROCESSING") {
			return text
		}
		select {
		case <-e.ctx.Done():
			t.Fatalf("upload verification did not finish: %v", e.ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// logFileDryRunText builds the expected dry_run_inspection output of an inspection with only the file feature enabled.
func logFileDryRunText(id, summary, fieldLines string) string {
	return fmt.Sprintf("# Dry run of `%s`\n\n%s\n\n"+
		"## Fields\n\n"+
		"### General\n\n"+
		"#### `khi.google.com/inspection/input/inspection-name`\n- Label: Inspection name\n- Description: The display name of this inspection.\n- Type: text\n"+
		"- Value: `Google Kubernetes Engine`\n- Default: `Google Kubernetes Engine`\n\n"+
		"#### `log-file`\n- Label: Log File\n- Type: file\n%s", id, summary, fieldLines)
}

func TestInspectionTools_FileUpload(t *testing.T) {
	const notReadySummary = "Errors: 1, warnings: 0. Fix the fields with errors and call `dry_run_inspection` again."

	t.Run("upload, verify and run", func(t *testing.T) {
		env := newToolsTestEnv(t)
		id := env.createInspection(t, fileFeatureID)

		text, isError := callTool(t, env.ctx, env.session, "dry_run_inspection", map[string]any{"inspectionId": id})
		want := logFileDryRunText(id, notReadySummary, "- Upload: WAITING\n- Error: Waiting a file to be uploaded.")
		checkToolResult(t, "dry_run_inspection", text, isError, want, false)

		status, body := putFile(t, env.requestUploadURL(t, id), "{\"a\":1}\n")
		if status != http.StatusOK {
			t.Fatalf("upload status = %d, want %d: %s", status, http.StatusOK, body)
		}
		wantBody := `{"fieldId":"log-file","status":"PROCESSING","sizeBytes":8}`
		if body != wantBody {
			t.Errorf("upload response = %s, want %s", body, wantBody)
		}

		text = env.dryRunUntilUploadSettles(t, id)
		want = logFileDryRunText(id, "Errors: 0, warnings: 0. Ready to run. Call `run_inspection` with the parameters to start the inspection.", "- Upload: COMPLETED, 8 bytes")
		checkToolResult(t, "dry_run_inspection", text, false, want, false)

		env.runInspection(t, id, map[string]any{})
		text, isError = callTool(t, env.ctx, env.session, "wait_inspection", map[string]any{
			"inspectionId":   id,
			"timeoutSeconds": 30,
		})
		want = fmt.Sprintf("# Inspection `%s`\n\nStatus: DONE. Read `khi://inspections/%s` for the summary.", id, id)
		checkToolResult(t, "wait_inspection", text, isError, want, false)
	})

	t.Run("verification error appears in the dry run", func(t *testing.T) {
		env := newToolsTestEnv(t)
		id := env.createInspection(t, fileFeatureID)

		status, body := putFile(t, env.requestUploadURL(t, id), "not json\n")
		if status != http.StatusOK {
			t.Fatalf("upload status = %d, want %d: %s", status, http.StatusOK, body)
		}

		text := env.dryRunUntilUploadSettles(t, id)
		want := logFileDryRunText(id, notReadySummary, "- Upload: ERROR\n- Error: invalid JSON on line 1: invalid character 'o' in literal null (expecting 'u')")
		checkToolResult(t, "dry_run_inspection", text, false, want, false)
	})

	t.Run("field that is not a file field", func(t *testing.T) {
		env := newToolsTestEnv(t)
		id := env.createInspection(t, fileFeatureID)
		const fieldID = "khi.google.com/inspection/input/inspection-name"

		text, isError := callTool(t, env.ctx, env.session, "request_file_upload", map[string]any{
			"inspectionId": id,
			"fieldId":      fieldID,
		})
		want := fmt.Sprintf("Error: FILE_FIELD_NOT_FOUND\n\n- Inspection `%s` has no field `%s` with `Type: file`.\n- Call `dry_run_inspection` and use the ID of a field with `Type: file`.", id, fieldID)
		checkToolResult(t, "request_file_upload", text, isError, want, true)
	})
}
