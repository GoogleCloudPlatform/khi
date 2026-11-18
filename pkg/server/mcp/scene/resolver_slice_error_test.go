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

package scene

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

func TestSliceErrorResult(t *testing.T) {
	type sliceErrorOutcome struct {
		Handled bool
		Result  *mcpsdk.CallToolResult
	}

	testCases := []struct {
		name       string
		err        error
		timelineID uint32
		logID      uint32
		fieldPath  string
		want       sliceErrorOutcome
	}{
		{
			name:       "ErrTimelineNotFound produces TIMELINE_NOT_FOUND",
			err:        workbench.ErrTimelineNotFound,
			timelineID: 7,
			fieldPath:  "annotations.diffExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"TIMELINE_NOT_FOUND",
					"No timeline with ID 7 exists in inspection `insp-1`.",
					"Use a timeline ID from `search_timelines` or `search_logs`.",
				),
			},
		},
		{
			name:       "ErrRevisionNotFound produces REVISION_NOT_FOUND",
			err:        workbench.ErrRevisionNotFound,
			timelineID: 7,
			fieldPath:  "annotations.manifestExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"REVISION_NOT_FOUND",
					"Timeline 7 has no matching revision: revision not found.",
					"Use a revision index from `get_resource_revisions`.",
				),
			},
		},
		{
			name:      "ErrLogNotFound produces LOG_NOT_FOUND",
			err:       workbench.ErrLogNotFound,
			logID:     42,
			fieldPath: "annotations.logExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"LOG_NOT_FOUND",
					"No log with ID 42 exists in inspection `insp-1`.",
					"Use a log ID from `search_logs` or `get_timeline_logs`.",
				),
			},
		},
		{
			name:      "InvalidJSONPathError with PathIndex -1",
			err:       &InvalidJSONPathError{PathIndex: -1, Reason: "empty"},
			fieldPath: "annotations.diffExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"INVALID_ARGUMENT",
					"`annotations.diffExcerpts[0].paths` is required and must contain at least one JSONPath selector.",
				),
			},
		},
		{
			name:      "InvalidJSONPathError with PathIndex 0",
			err:       &InvalidJSONPathError{PathIndex: 0, Path: "$.spec[", Reason: "unclosed '['"},
			fieldPath: "annotations.manifestExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"INVALID_ARGUMENT",
					"Invalid JSONPath `$.spec[` in `annotations.manifestExcerpts[0].paths[0]`: unclosed '['.",
				),
			},
		},
		{
			name:      "InvalidSliceWindowError with WindowIndex -1",
			err:       &InvalidSliceWindowError{WindowIndex: -1, Reason: "empty windows"},
			fieldPath: "annotations.logExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"INVALID_ARGUMENT",
					"Invalid `annotations.logExcerpts[0].windows`: empty windows.",
				),
			},
		},
		{
			name:      "InvalidSliceWindowError with WindowIndex 0",
			err:       &InvalidSliceWindowError{WindowIndex: 0, Reason: "empty"},
			fieldPath: "annotations.logExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"INVALID_ARGUMENT",
					"Invalid `annotations.logExcerpts[0].windows[0]`: empty.",
				),
			},
		},
		{
			name: "SliceQueryNotFoundError with path and query on logExcerpts",
			err: &SliceQueryNotFoundError{
				WindowIndex: 1,
				Path:        "$.textPayload",
				Query:       "panic",
				Summary:     "Container restarted",
			},
			fieldPath: "annotations.logExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"SLICE_QUERY_NOT_FOUND",
					"Query `panic` within JSONPath `$.textPayload` in `annotations.logExcerpts[0].windows[1]` matched 0 lines.",
					"Log summary: `Container restarted`.",
					"Inspect the full content with `get_resource_diff` / `get_resource_manifest` / `get_log` to verify available fields.",
				),
			},
		},
		{
			name: "SliceQueryNotFoundError with path only on logExcerpts",
			err: &SliceQueryNotFoundError{
				WindowIndex:  0,
				Path:         "$.missing",
				TopLevelKeys: []string{"insertId", "message"},
			},
			fieldPath: "annotations.logExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"SLICE_QUERY_NOT_FOUND",
					"JSONPath `$.missing` in `annotations.logExcerpts[0].windows[0].path` matched 0 YAML nodes.",
					"Available top-level keys: `insertId`, `message`.",
					"Inspect the full content with `get_resource_diff` / `get_resource_manifest` / `get_log` to verify available fields.",
				),
			},
		},
		{
			name: "SliceQueryNotFoundError with path, availableKeys, and manifest preview",
			err: &SliceQueryNotFoundError{
				WindowIndex:   0,
				Path:          "$.status.missing",
				MatchedPrefix: "$.status",
				AvailableKeys: []string{"phase"},
				Preview:       "apiVersion: v1",
			},
			fieldPath: "annotations.manifestExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"SLICE_QUERY_NOT_FOUND",
					"JSONPath `$.status.missing` in `annotations.manifestExcerpts[0].paths[0]` matched 0 YAML nodes.",
					"Available keys under `$.status`: `phase`.",
					"Manifest preview:\n```yaml\napiVersion: v1\n```",
					"Inspect the full content with `get_resource_diff` / `get_resource_manifest` / `get_log` to verify available fields.",
				),
			},
		},
		{
			name: "SliceQueryNotFoundError with diff preview",
			err: &SliceQueryNotFoundError{
				WindowIndex: 0,
				Path:        "$.status.missing",
				Preview: `-old
+new`,
			},
			fieldPath: "annotations.diffExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"SLICE_QUERY_NOT_FOUND",
					"JSONPath `$.status.missing` in `annotations.diffExcerpts[0].paths[0]` matched 0 YAML nodes.",
					"Changed hunk preview:\n```diff\n-old\n+new\n```",
					"Inspect the full content with `get_resource_diff` / `get_resource_manifest` / `get_log` to verify available fields.",
				),
			},
		},
		{
			name: "SliceQueryNotFoundError with query only",
			err: &SliceQueryNotFoundError{
				WindowIndex: 0,
				Query:       "OOM",
			},
			fieldPath: "annotations.logExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"SLICE_QUERY_NOT_FOUND",
					"Query `OOM` in `annotations.logExcerpts[0].windows[0]` matched 0 lines.",
					"Inspect the full content with `get_resource_diff` / `get_resource_manifest` / `get_log` to verify available fields.",
				),
			},
		},
		{
			name: "SliceQueryNotFoundError with matchedPrefix but empty availableKeys falls back to topLevelKeys",
			err: &SliceQueryNotFoundError{
				WindowIndex:   0,
				Path:          "$.status.phase.extra",
				MatchedPrefix: "$.status.phase",
				TopLevelKeys:  []string{"apiVersion", "status"},
			},
			fieldPath: "annotations.manifestExcerpts[0]",
			want: sliceErrorOutcome{
				Handled: true,
				Result: wantErrorResult(
					"SLICE_QUERY_NOT_FOUND",
					"JSONPath `$.status.phase.extra` in `annotations.manifestExcerpts[0].paths[0]` matched 0 YAML nodes.",
					"Available top-level keys: `apiVersion`, `status`.",
					"Inspect the full content with `get_resource_diff` / `get_resource_manifest` / `get_log` to verify available fields.",
				),
			},
		},
		{
			name:      "unrelated error returns unhandled",
			err:       errors.New("unexpected io error"),
			fieldPath: "annotations.diffExcerpts[0]",
			want:      sliceErrorOutcome{Handled: false},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, handled := sliceErrorResult(tc.err, "insp-1", tc.timelineID, tc.logID, tc.fieldPath)
			got := sliceErrorOutcome{Handled: handled, Result: res}
			if diff := cmp.Diff(tc.want, got, ignoreMCPUnexported); diff != "" {
				t.Errorf("sliceErrorResult() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
