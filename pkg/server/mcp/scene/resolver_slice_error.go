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
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/md"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

func sliceErrorResult(err error, inspectionID string, timelineID, logID uint32, fieldPath string) (*mcpsdk.CallToolResult, bool) {
	if res, ok := revisionLookupErrorResult(err, inspectionID, timelineID); ok {
		return res, true
	}
	if errors.Is(err, workbench.ErrLogNotFound) {
		res, _, _ := mdtemplate.ErrorResult(
			"LOG_NOT_FOUND",
			fmt.Sprintf("No log with ID %d exists in inspection %s.", logID, md.Code(inspectionID)),
			"Use a log ID from `search_logs` or `get_timeline_logs`.",
		)
		return res, true
	}
	var pathErr *InvalidJSONPathError
	if errors.As(err, &pathErr) {
		return invalidJSONPathErrorResult(pathErr, fieldPath), true
	}
	var winErr *InvalidSliceWindowError
	if errors.As(err, &winErr) {
		return invalidSliceWindowErrorResult(winErr, fieldPath), true
	}
	var notFoundErr *SliceQueryNotFoundError
	if errors.As(err, &notFoundErr) {
		return sliceQueryNotFoundErrorResult(notFoundErr, fieldPath), true
	}
	return nil, false
}

func invalidJSONPathErrorResult(pathErr *InvalidJSONPathError, fieldPath string) *mcpsdk.CallToolResult {
	msg := fmt.Sprintf("`%s.paths` is required and must contain at least one JSONPath selector.", fieldPath)
	if pathErr.PathIndex >= 0 {
		msg = fmt.Sprintf("Invalid JSONPath %s in `%s.paths[%d]`: %s.", md.Code(pathErr.Path), fieldPath, pathErr.PathIndex, pathErr.Reason)
	}
	res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", msg)
	return res
}

func invalidSliceWindowErrorResult(winErr *InvalidSliceWindowError, fieldPath string) *mcpsdk.CallToolResult {
	msg := fmt.Sprintf("Invalid `%s.windows`: %s.", fieldPath, winErr.Reason)
	if winErr.WindowIndex >= 0 {
		msg = fmt.Sprintf("Invalid `%s.windows[%d]`: %s.", fieldPath, winErr.WindowIndex, winErr.Reason)
	}
	res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", msg)
	return res
}

func sliceQueryNotFoundErrorResult(notFoundErr *SliceQueryNotFoundError, fieldPath string) *mcpsdk.CallToolResult {
	bullets := []string{formatSliceNotFoundLead(notFoundErr, fieldPath)}
	if notFoundErr.MatchedPrefix != "" && len(notFoundErr.AvailableKeys) > 0 {
		bullets = append(bullets, fmt.Sprintf("Available keys under %s: `%s`.",
			md.Code(notFoundErr.MatchedPrefix), strings.Join(notFoundErr.AvailableKeys, "`, `")))
	} else if len(notFoundErr.TopLevelKeys) > 0 {
		bullets = append(bullets, fmt.Sprintf("Available top-level keys: `%s`.", strings.Join(notFoundErr.TopLevelKeys, "`, `")))
	}
	if notFoundErr.Summary != "" {
		bullets = append(bullets, fmt.Sprintf("Log summary: %s.", md.Code(notFoundErr.Summary)))
	}
	if notFoundErr.Preview != "" {
		if strings.HasPrefix(fieldPath, "annotations.manifestExcerpts") {
			bullets = append(bullets, fmt.Sprintf("Manifest preview:\n%s", md.Fence("yaml", notFoundErr.Preview)))
		} else {
			bullets = append(bullets, fmt.Sprintf("Changed hunk preview:\n%s", md.Fence("diff", notFoundErr.Preview)))
		}
	}
	bullets = append(bullets, "Inspect the full content with `get_resource_diff` / `get_resource_manifest` / `get_log` to verify available fields.")
	res, _, _ := mdtemplate.ErrorResult("SLICE_QUERY_NOT_FOUND", bullets...)
	return res
}

func formatSliceNotFoundLead(notFoundErr *SliceQueryNotFoundError, fieldPath string) string {
	switch {
	case notFoundErr.Path != "" && notFoundErr.Query != "":
		return fmt.Sprintf("Query %s within JSONPath %s in `%s.windows[%d]` matched 0 lines.",
			md.Code(notFoundErr.Query), md.Code(notFoundErr.Path), fieldPath, notFoundErr.WindowIndex)
	case notFoundErr.Path != "":
		selectorRef := fmt.Sprintf("%s.paths[%d]", fieldPath, notFoundErr.WindowIndex)
		if strings.HasPrefix(fieldPath, "annotations.logExcerpts") {
			selectorRef = fmt.Sprintf("%s.windows[%d].path", fieldPath, notFoundErr.WindowIndex)
		}
		return fmt.Sprintf("JSONPath %s in `%s` matched 0 YAML nodes.", md.Code(notFoundErr.Path), selectorRef)
	default:
		return fmt.Sprintf("Query %s in `%s.windows[%d]` matched 0 lines.", md.Code(notFoundErr.Query), fieldPath, notFoundErr.WindowIndex)
	}
}
