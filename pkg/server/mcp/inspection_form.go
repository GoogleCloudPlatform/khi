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
	"maps"
	"strconv"
	"strings"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/md"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/GoogleCloudPlatform/khi/pkg/server/uploadurl"
)

// maxDisplayedFieldChoices is the maximum number of suggestions or options rendered for a field unless expanded via fieldId.
const maxDisplayedFieldChoices = 20

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

// groupFormFields organizes flat and nested form fields into groups and counts errors and warnings.
func groupFormFields(fields []inspectionmetadata.ParameterFormField, params map[string]any, expandFieldID string) ([]formGroupData, int, int) {
	var generalFields []formFieldData
	var groups []formGroupData
	totalErrors := 0
	totalWarnings := 0

	for _, field := range fields {
		if group, ok := field.(inspectionmetadata.GroupParameterFormField); ok {
			var groupFields []formFieldData
			for _, child := range group.Children {
				data, errs, warns := convertFormField(child, params, expandFieldID)
				groupFields = append(groupFields, data)
				totalErrors += errs
				totalWarnings += warns
			}
			if len(groupFields) > 0 {
				groups = append(groups, formGroupData{
					Title:  group.Label,
					Fields: groupFields,
				})
			}
			continue
		}
		data, errs, warns := convertFormField(field, params, expandFieldID)
		generalFields = append(generalFields, data)
		totalErrors += errs
		totalWarnings += warns
	}

	if len(generalFields) > 0 {
		groups = append([]formGroupData{{
			Title:  "General",
			Fields: generalFields,
		}}, groups...)
	}

	return groups, totalErrors, totalWarnings
}

// convertFormField converts a ParameterFormField into formFieldData and returns its error and warning counts (0 or 1).
func convertFormField(field inspectionmetadata.ParameterFormField, params map[string]any, expandFieldID string) (formFieldData, int, int) {
	base := inspectionmetadata.GetParameterFormFieldBase(field)
	data := toFormFieldData(field, params, expandFieldID != "" && base.ID == expandFieldID)
	switch data.HintType {
	case "Error":
		return data, 1, 0
	case "Warning":
		return data, 0, 1
	default:
		return data, 0, 0
	}
}

// toFormFieldData converts a ParameterFormField into formFieldData for template rendering.
func toFormFieldData(field inspectionmetadata.ParameterFormField, params map[string]any, expand bool) formFieldData {
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
		data.Suggestions = formatFieldChoices(base.ID, tf.Suggestions, expand)
	case inspectionmetadata.SetParameterFormField:
		ids := make([]string, len(tf.Options))
		for i, opt := range tf.Options {
			ids[i] = opt.ID
		}
		data.Options = formatFieldChoices(base.ID, ids, expand)
	case inspectionmetadata.FileParameterFormField:
		data.UploadStatus = fileUploadStatus(tf)
	}

	return data
}

// formatFieldChoices renders items as a comma-separated list of inline code, truncating beyond maxDisplayedFieldChoices unless expand is true.
func formatFieldChoices(id string, items []string, expand bool) string {
	if expand || len(items) <= maxDisplayedFieldChoices {
		return codeList(items)
	}
	omitted := len(items) - maxDisplayedFieldChoices
	return fmt.Sprintf("%s ... and %d more (call `dry_run_inspection` with `fieldId: %q` to list all)",
		codeList(items[:maxDisplayedFieldChoices]), omitted, id)
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
		return md.Code(v)
	case bool:
		return md.Code(strconv.FormatBool(v))
	default:
		return ""
	}
}

// codeList renders items as a comma separated list of inline code.
func codeList(items []string) string {
	codes := make([]string, len(items))
	for i, item := range items {
		codes[i] = md.Code(item)
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
		return fmt.Sprintf("COMPLETED, %s, %d bytes", md.Code(field.FileName), field.SizeBytes)
	}
}
