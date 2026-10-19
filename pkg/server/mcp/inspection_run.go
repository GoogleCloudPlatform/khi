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
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/GoogleCloudPlatform/khi/pkg/server/uploadurl"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DryRunInspectionInput defines the input parameters for the dry_run_inspection tool.
type DryRunInspectionInput struct {
	InspectionID string         `json:"inspectionId" jsonschema:"The unique identifier of the inspection."`
	Parameters   map[string]any `json:"parameters,omitempty" jsonschema:"Key-value map of form parameters. Pass text fields as strings, set fields as arrays of strings, and checkbox fields as booleans. File fields cannot be set here; upload the file with request_file_upload. Time range parameters should be in UTC RFC3339 format. Wider time ranges increase execution time and memory usage."`
}

// RunInspectionInput defines the input parameters for the run_inspection tool.
type RunInspectionInput struct {
	InspectionID string         `json:"inspectionId" jsonschema:"The unique identifier of the inspection."`
	Parameters   map[string]any `json:"parameters,omitempty" jsonschema:"Key-value map of form parameters. Pass text fields as strings, set fields as arrays of strings, and checkbox fields as booleans. File fields cannot be set here; upload the file with request_file_upload. Time range parameters should be in UTC RFC3339 format. Wider time ranges increase execution time and memory usage."`
}

type formFieldData struct {
	ID          string
	Label       string
	Description string
	Type        string
	// Value is the value that the inspection uses: the passed parameter, or the default when the parameter is not passed.
	Value string
	// IsDefaulted reports that the parameter is not passed and Value is the default.
	IsDefaulted bool
	Default     string
	Suggestions string
	Options     string
	// UploadStatus is the upload status line of a file field. It is empty for other field types.
	UploadStatus string
	HintType     string
	Hint         string
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
	// DefaultedFields are the fields whose value is the default because the parameter is not passed.
	DefaultedFields []formFieldData
	Groups          []formGroupData
	Queries         []queryData
}

type runInspectionData struct {
	ID string
	// Parameters are the fields with a value that the run uses.
	Parameters []formFieldData
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

// dryRunResult is the outcome of a successful dry run.
type dryRunResult struct {
	// renderData is the render model of dry_run_inspection.
	renderData *dryRunData
	// formFields are the form fields that the dry run returned, before they are converted for rendering.
	formFields []inspectionmetadata.ParameterFormField
	// preparedParams are the parameters that the dry run used, which run_inspection passes to the run.
	preparedParams map[string]any
}

// dryRun prepares rawParams with prepareParameters and runs a dry run of runner, shared by dry_run_inspection,
// run_inspection, and request_file_upload.
// Exactly one of the following holds on return:
//   - Success: the dryRunResult is non-nil, and the rest are nil.
//   - Tool error: the CallToolResult is non-nil with IsError set, for problems the agent can fix,
//     such as invalid parameter types or expired Google Cloud credentials.
//   - Go error: the error is non-nil for unexpected failures that become protocol errors.
//
// retryTool is the tool name that the GOOGLE_CLOUD_AUTH error tells the agent to call again after login.
func (h *InspectionHandler) dryRun(ctx context.Context, id string, runner *coreinspection.InspectionTaskRunner, rawParams map[string]any, retryTool string) (*dryRunResult, *mcpsdk.CallToolResult, error) {
	preparedParams, errRes, err := h.prepareParameters(id, rawParams)
	if errRes != nil || err != nil {
		return nil, errRes, err
	}

	coreResult, err := runner.DryRun(ctx, &inspectioncore.InspectionRequest{
		Values: preparedParams,
	})
	if err != nil {
		if isGoogleCloudAuthError(err) {
			authRes, _, _ := h.googleCloudAuthErrorResult(retryTool)
			return nil, authRes, nil
		}
		return nil, nil, err
	}

	mdMap, ok := coreResult.Metadata.(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("unexpected dryrun metadata format")
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

	return &dryRunResult{
		renderData: &dryRunData{
			ID:           id,
			ErrorCount:   errCount,
			WarningCount: warnCount,
			DefaultedFields: fieldsWhere(groups, func(field formFieldData) bool {
				return field.IsDefaulted
			}),
			Groups:  groups,
			Queries: queries,
		},
		formFields:     formFields,
		preparedParams: preparedParams,
	}, nil, nil
}

func (h *InspectionHandler) handleDryRunInspection(ctx context.Context, req *mcpsdk.CallToolRequest, in DryRunInspectionInput) (*mcpsdk.CallToolResult, any, error) {
	runner := h.server.GetInspection(in.InspectionID)
	if runner == nil {
		return inspectionNotFoundResult(in.InspectionID)
	}

	if runner.Started() {
		return inspectionAlreadyStartedResult(in.InspectionID)
	}

	dryRunOutcome, errRes, err := h.dryRun(ctx, in.InspectionID, runner, in.Parameters, "dry_run_inspection")
	if errRes != nil || err != nil {
		return errRes, nil, err
	}

	return h.templates.ToolResult("dry_run_inspection.md.tmpl", *dryRunOutcome.renderData)
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

	dryRunOutcome, errRes, err := h.dryRun(ctx, in.InspectionID, runner, in.Parameters, "run_inspection")
	if errRes != nil || err != nil {
		return errRes, nil, err
	}

	var bullets []string
	for _, group := range dryRunOutcome.renderData.Groups {
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

	// Pass the defaults that the dry run showed so that a default computed again at run time cannot differ from them.
	if err := runner.Run(context.WithoutCancel(ctx), &inspectioncore.InspectionRequest{
		Values: withDefaults(dryRunOutcome.preparedParams, dryRunOutcome.formFields),
	}); err != nil {
		return nil, nil, err
	}

	return h.templates.ToolResult("run_inspection.md.tmpl", runInspectionData{
		ID: in.InspectionID,
		Parameters: fieldsWhere(dryRunOutcome.renderData.Groups, func(field formFieldData) bool {
			return field.Value != ""
		}),
	})
}

// withDefaults returns a copy of params with the default of every text, set, and checkbox field that params does not have,
// including the fields in groups.
func withDefaults(params map[string]any, fields []inspectionmetadata.ParameterFormField) map[string]any {
	result := maps.Clone(params)
	var addDefaults func(fields []inspectionmetadata.ParameterFormField)
	addDefaults = func(fields []inspectionmetadata.ParameterFormField) {
		for _, field := range fields {
			if group, ok := field.(inspectionmetadata.GroupParameterFormField); ok {
				addDefaults(group.Children)
				continue
			}
			id := inspectionmetadata.GetParameterFormFieldBase(field).ID
			if _, passed := result[id]; passed {
				continue
			}
			if value, ok := defaultValue(field); ok {
				result[id] = value
			}
		}
	}
	addDefaults(fields)
	return result
}

// defaultValue returns the default of a text, set, or checkbox field in the type that the form task accepts.
// It returns false for other fields and for text and set fields with an empty default.
func defaultValue(field inspectionmetadata.ParameterFormField) (any, bool) {
	switch tf := field.(type) {
	case inspectionmetadata.TextParameterFormField:
		return tf.Default, tf.Default != ""
	case inspectionmetadata.SetParameterFormField:
		return tf.Default, len(tf.Default) > 0
	case inspectionmetadata.CheckboxParameterFormField:
		return tf.Default, true
	default:
		return nil, false
	}
}

// fieldsWhere returns the fields of groups that keep reports true for, in display order.
func fieldsWhere(groups []formGroupData, keep func(field formFieldData) bool) []formFieldData {
	var result []formFieldData
	for _, group := range groups {
		for _, field := range group.Fields {
			if keep(field) {
				result = append(result, field)
			}
		}
	}
	return result
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
				data := toFormFieldData(child, params)
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
			data := toFormFieldData(field, params)
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
func toFormFieldData(field inspectionmetadata.ParameterFormField, params map[string]any) formFieldData {
	base := inspectionmetadata.GetParameterFormFieldBase(field)
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

	def, hasDefault := defaultValue(field)
	if hasDefault {
		data.Default = formatParameterValue(def)
	}
	if val, passed := params[base.ID]; passed {
		data.Value = formatParameterValue(val)
	} else if hasDefault {
		data.Value = data.Default
		data.IsDefaulted = true
	}

	switch tf := field.(type) {
	case inspectionmetadata.TextParameterFormField:
		data.Suggestions = codeList(tf.Suggestions)
	case inspectionmetadata.SetParameterFormField:
		ids := make([]string, len(tf.Options))
		for i, opt := range tf.Options {
			ids[i] = opt.ID
		}
		data.Options = codeList(ids)
	case inspectionmetadata.FileParameterFormField:
		data.UploadStatus = fileUploadStatus(tf)
	}

	return data
}

// formatParameterValue renders a parameter value of a text, set, or checkbox field as inline code.
// It returns an empty string for an empty text or set value.
func formatParameterValue(value any) string {
	switch v := value.(type) {
	case []string:
		return codeList(v)
	case string:
		if v == "" {
			return ""
		}
		return mdtemplate.Code(v)
	case bool:
		return mdtemplate.Code(strconv.FormatBool(v))
	default:
		return ""
	}
}

// codeList renders items as a comma separated list of inline code.
func codeList(items []string) string {
	codes := make([]string, len(items))
	for i, item := range items {
		codes[i] = mdtemplate.Code(item)
	}
	return strings.Join(codes, ", ")
}

// fileUploadStatus returns the upload status line of a file field: WAITING, PROCESSING, ERROR, or COMPLETED with the file size.
// A failed upload waits for another upload, so it is WAITING and its hint explains the failure.
// A file that fails verification is ERROR and its hint explains the verification error.
func fileUploadStatus(field inspectionmetadata.FileParameterFormField) string {
	switch {
	case field.Status == upload.UploadStatusWaiting:
		return "WAITING"
	case field.Status != upload.UploadStatusCompleted:
		return uploadurl.StatusProcessing
	case field.HintType == inspectionmetadata.Error:
		return "ERROR"
	case field.FileName == "":
		return fmt.Sprintf("COMPLETED, %d bytes", field.SizeBytes)
	default:
		return fmt.Sprintf("COMPLETED, %s, %d bytes", mdtemplate.Code(field.FileName), field.SizeBytes)
	}
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
