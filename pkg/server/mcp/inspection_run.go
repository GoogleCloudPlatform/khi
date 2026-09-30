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
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"cloud.google.com/go/auth"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DryRunInspectionInput defines the input parameters for the dry_run_inspection tool.
type DryRunInspectionInput struct {
	InspectionID string         `json:"inspectionId" jsonschema:"The unique identifier of the inspection."`
	Parameters   map[string]any `json:"parameters,omitempty" jsonschema:"Key-value map of form parameters. Time range parameters should be in UTC RFC3339 format. Wider time ranges increase execution time and memory usage."`
}

// RunInspectionInput defines the input parameters for the run_inspection tool.
type RunInspectionInput struct {
	InspectionID string         `json:"inspectionId" jsonschema:"The unique identifier of the inspection."`
	Parameters   map[string]any `json:"parameters,omitempty" jsonschema:"Key-value map of form parameters. Time range parameters should be in UTC RFC3339 format. Wider time ranges increase execution time and memory usage."`
}

type formFieldData struct {
	ID          string
	Label       string
	Description string
	Type        string
	Value       string
	Default     string
	Suggestions string
	Options     string
	HintType    string
	Hint        string
}

type formGroupData struct {
	Title  string
	Fields []formFieldData
}

type queryData struct {
	ID             string
	Name           string
	EstimatedCount string
	Query          string
}

type dryRunData struct {
	ID           string
	ErrorCount   int
	WarningCount int
	Groups       []formGroupData
	Queries      []queryData
}

type invalidParameterError struct {
	Key string
}

func (e *invalidParameterError) Error() string {
	return fmt.Sprintf("parameter %q must be a string, boolean, or an array of strings", e.Key)
}

// convertMCPParameters sanitizes parameter types from JSON decoding into types expected by tasks.
// Strings, booleans, and slices of strings are supported. Nil values are skipped.
// Other types return an invalidParameterError for the first invalid key in sorted order.
func convertMCPParameters(params map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(params))
	for _, k := range slices.Sorted(maps.Keys(params)) {
		switch val := params[k].(type) {
		case nil:
			continue
		case string, bool:
			result[k] = val
		case []any:
			strs := make([]string, len(val))
			for i, elem := range val {
				s, ok := elem.(string)
				if !ok {
					return nil, &invalidParameterError{Key: k}
				}
				strs[i] = s
			}
			result[k] = strs
		default:
			return nil, &invalidParameterError{Key: k}
		}
	}
	return result, nil
}

// prepareParameters validates and converts input parameters and injects the registered inspection name if omitted.
func (h *InspectionHandler) prepareParameters(id string, rawParams map[string]any) (map[string]any, *mcpsdk.CallToolResult, error) {
	preparedParams, err := convertMCPParameters(rawParams)
	if err != nil {
		var invErr *invalidParameterError
		if errors.As(err, &invErr) {
			res, _, _ := mdtemplate.ErrorResult("INVALID_PARAMETERS",
				fmt.Sprintf("Parameter %s must be a string, boolean, or an array of strings.", mdtemplate.Code(invErr.Key)))
			return nil, res, nil
		}
		return nil, nil, err
	}

	nameKey := inspectioncore.InputInspectionNameTaskID.ReferenceIDString()
	if _, exists := preparedParams[nameKey]; !exists {
		if name, ok := h.server.InspectionNameRegistry().NameOf(id); ok && name != "" {
			preparedParams[nameKey] = name
		}
	}

	return preparedParams, nil, nil
}

// dryRun prepares rawParams with prepareParameters and runs a dry run of runner, shared by dry_run_inspection
// and run_inspection.
// Exactly one of the following holds on return:
//   - Success: the dry run data and the prepared parameters are non-nil, and the rest are nil.
//   - Tool error: the CallToolResult is non-nil with IsError set, for problems the agent can fix,
//     such as invalid parameter types or expired Google Cloud credentials.
//   - Go error: the error is non-nil for unexpected failures that become protocol errors.
//
// retryTool is the tool name that the GOOGLE_CLOUD_AUTH error tells the agent to call again after login.
func (h *InspectionHandler) dryRun(ctx context.Context, id string, runner *coreinspection.InspectionTaskRunner, rawParams map[string]any, retryTool string) (*dryRunData, map[string]any, *mcpsdk.CallToolResult, error) {
	preparedParams, errRes, err := h.prepareParameters(id, rawParams)
	if errRes != nil || err != nil {
		return nil, nil, errRes, err
	}

	result, err := runner.DryRun(ctx, &inspectioncore.InspectionRequest{
		Values: preparedParams,
	})
	if err != nil {
		if isGoogleCloudAuthError(err) {
			authRes, _, _ := h.googleCloudAuthErrorResult(retryTool)
			return nil, nil, authRes, nil
		}
		return nil, nil, nil, err
	}

	mdMap, ok := result.Metadata.(map[string]interface{})
	if !ok {
		return nil, nil, nil, fmt.Errorf("unexpected dryrun metadata format")
	}

	var formFields []inspectionmetadata.ParameterFormField
	if f, ok := mdMap["form"].([]inspectionmetadata.ParameterFormField); ok {
		formFields = f
	}

	var queryItems []*inspectionmetadata.QueryItem
	if q, ok := mdMap["query"].([]*inspectionmetadata.QueryItem); ok {
		queryItems = q
	}

	groups, errCount, warnCount := groupFormFields(formFields, preparedParams)
	queries := toQueryData(queryItems)

	return &dryRunData{
		ID:           id,
		ErrorCount:   errCount,
		WarningCount: warnCount,
		Groups:       groups,
		Queries:      queries,
	}, preparedParams, nil, nil
}

func (h *InspectionHandler) handleDryRunInspection(ctx context.Context, req *mcpsdk.CallToolRequest, in DryRunInspectionInput) (*mcpsdk.CallToolResult, any, error) {
	runner := h.server.GetInspection(in.InspectionID)
	if runner == nil {
		return inspectionNotFoundResult(in.InspectionID)
	}

	if runner.Started() {
		return inspectionAlreadyStartedResult(in.InspectionID)
	}

	data, _, errRes, err := h.dryRun(ctx, in.InspectionID, runner, in.Parameters, "dry_run_inspection")
	if errRes != nil || err != nil {
		return errRes, nil, err
	}

	return h.templates.ToolResult("dry_run_inspection.md.tmpl", *data)
}

type inspectionIDData struct {
	ID string
}

func (h *InspectionHandler) handleRunInspection(ctx context.Context, req *mcpsdk.CallToolRequest, in RunInspectionInput) (*mcpsdk.CallToolResult, any, error) {
	runner := h.server.GetInspection(in.InspectionID)
	if runner == nil {
		return inspectionNotFoundResult(in.InspectionID)
	}

	if runner.Started() {
		return inspectionAlreadyStartedResult(in.InspectionID)
	}

	data, preparedParams, errRes, err := h.dryRun(ctx, in.InspectionID, runner, in.Parameters, "run_inspection")
	if errRes != nil || err != nil {
		return errRes, nil, err
	}

	var bullets []string
	for _, group := range data.Groups {
		for _, field := range group.Fields {
			if field.HintType == "Error" && field.Hint != "" {
				bullets = append(bullets, fmt.Sprintf("%s: %s", mdtemplate.Code(field.ID), field.Hint))
			}
		}
	}
	if len(bullets) > 0 {
		bullets = append([]string{"The inspection has parameter errors. Fix them and call `dry_run_inspection` again."}, bullets...)
		return mdtemplate.ErrorResult("INVALID_PARAMETERS", bullets...)
	}

	if err := runner.Run(context.WithoutCancel(ctx), &inspectioncore.InspectionRequest{
		Values: preparedParams,
	}); err != nil {
		return nil, nil, err
	}

	return h.templates.ToolResult("run_inspection.md.tmpl", inspectionIDData{
		ID: in.InspectionID,
	})
}

// groupFormFields organizes flat/nested form fields into Web UI groups and counts errors/warnings.
func groupFormFields(fields []inspectionmetadata.ParameterFormField, params map[string]any) ([]formGroupData, int, int) {
	var generalFields []formFieldData
	var groups []formGroupData
	totalErrors := 0
	totalWarnings := 0

	for _, field := range fields {
		if group, ok := field.(inspectionmetadata.GroupParameterFormField); ok {
			var groupFields []formFieldData
			for _, child := range group.Children {
				data, ok := toFormFieldData(child, params)
				if !ok {
					continue
				}
				groupFields = append(groupFields, data)
				if data.HintType == "Error" {
					totalErrors++
				} else if data.HintType == "Warning" {
					totalWarnings++
				}
			}
			if len(groupFields) > 0 {
				groups = append(groups, formGroupData{
					Title:  group.Label,
					Fields: groupFields,
				})
			}
		} else {
			data, ok := toFormFieldData(field, params)
			if !ok {
				continue
			}
			generalFields = append(generalFields, data)
			if data.HintType == "Error" {
				totalErrors++
			} else if data.HintType == "Warning" {
				totalWarnings++
			}
		}
	}

	if len(generalFields) > 0 {
		groups = append([]formGroupData{{
			Title:  "General",
			Fields: generalFields,
		}}, groups...)
	}

	return groups, totalErrors, totalWarnings
}

// toFormFieldData converts a ParameterFormField into formFieldData for template rendering.
// File-type form fields are omitted because file uploads are not supported via MCP.
func toFormFieldData(field inspectionmetadata.ParameterFormField, params map[string]any) (formFieldData, bool) {
	base := inspectionmetadata.GetParameterFormFieldBase(field)
	if base.Type == inspectionmetadata.File {
		return formFieldData{}, false
	}

	data := formFieldData{
		ID:          base.ID,
		Label:       base.Label,
		Description: base.Description,
		Type:        string(base.Type),
	}

	if base.Hint != "" {
		switch base.HintType {
		case inspectionmetadata.Error:
			data.HintType = "Error"
		case inspectionmetadata.Warning:
			data.HintType = "Warning"
		case inspectionmetadata.Info:
			data.HintType = "Info"
		}
		if data.HintType != "" {
			data.Hint = base.Hint
		}
	}

	if val, ok := params[base.ID]; ok {
		switch v := val.(type) {
		case []string:
			if len(v) > 0 {
				items := make([]string, len(v))
				for i, s := range v {
					items[i] = mdtemplate.Code(s)
				}
				data.Value = strings.Join(items, ", ")
			}
		case string:
			if v != "" {
				data.Value = mdtemplate.Code(v)
			}
		case bool:
			data.Value = mdtemplate.Code(strconv.FormatBool(v))
		}
	}

	switch tf := field.(type) {
	case inspectionmetadata.TextParameterFormField:
		if tf.Default != "" {
			data.Default = mdtemplate.Code(tf.Default)
		}
		if len(tf.Suggestions) > 0 {
			items := make([]string, len(tf.Suggestions))
			for i, s := range tf.Suggestions {
				items[i] = mdtemplate.Code(s)
			}
			data.Suggestions = strings.Join(items, ", ")
		}
	case inspectionmetadata.SetParameterFormField:
		if len(tf.Default) > 0 {
			items := make([]string, len(tf.Default))
			for i, s := range tf.Default {
				items[i] = mdtemplate.Code(s)
			}
			data.Default = strings.Join(items, ", ")
		}
		if len(tf.Options) > 0 {
			items := make([]string, len(tf.Options))
			for i, opt := range tf.Options {
				items[i] = mdtemplate.Code(opt.ID)
			}
			data.Options = strings.Join(items, ", ")
		}
	case inspectionmetadata.CheckboxParameterFormField:
		if tf.Default {
			data.Default = mdtemplate.Code("true")
		}
	}

	return data, true
}

// toQueryData converts QueryItem pointers from dry run metadata into queryData for template rendering.
func toQueryData(queries []*inspectionmetadata.QueryItem) []queryData {
	result := make([]queryData, 0, len(queries))
	for _, q := range queries {
		est := ""
		if q.EstimatedCount != nil {
			est = fmt.Sprintf("%d", *q.EstimatedCount)
		}
		result = append(result, queryData{
			ID:             q.Id,
			Name:           q.Name,
			EstimatedCount: est,
			Query:          q.Query,
		})
	}
	return result
}

// isGoogleCloudAuthError checks if the given error is caused by missing, expired, or rejected Google Cloud credentials.
func isGoogleCloudAuthError(err error) bool {
	// gRPC clients turn token failures into this status and flatten the token error into its message.
	// Servers also return it for rejected tokens. status.Code walks wrapped errors.
	if status.Code(err) == codes.Unauthenticated {
		return true
	}
	// REST clients keep the token error from cloud.google.com/go/auth in the chain.
	// Only 4xx responses from the token endpoint mean the credentials themselves were rejected.
	var tokenErr *auth.Error
	if errors.As(err, &tokenErr) && tokenErr.Response != nil &&
		tokenErr.Response.StatusCode >= http.StatusBadRequest && tokenErr.Response.StatusCode < http.StatusInternalServerError {
		return true
	}
	// credentials.DetectDefault reports missing Application Default Credentials only as a formatted error
	// without a type or sentinel, so this case can only be detected by its message.
	return strings.Contains(err.Error(), "could not find default credentials")
}

type googleCloudAuthErrorData struct {
	RetryTool string
}

func (h *InspectionHandler) googleCloudAuthErrorResult(retryTool string) (*mcpsdk.CallToolResult, any, error) {
	return h.templates.ErrorToolResult("google_cloud_auth_error.md.tmpl", googleCloudAuthErrorData{
		RetryTool: retryTool,
	})
}
