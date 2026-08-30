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

package googlecloudlogcomposerapiaudit_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

var (
	pathEnvironmentName = structured.CompileFieldPath("resource.labels.environment_name")
	pathLocation        = structured.CompileFieldPath("resource.labels.location")
	pathProjectID       = structured.CompileFieldPath("resource.labels.project_id")
)

// ComposerAuditLogResourceFieldSet represents resource identifiers extracted from a Cloud Composer audit log entry.
type ComposerAuditLogResourceFieldSet struct {
	EnvironmentName string
	Location        string
	ProjectID       string
}

// Kind returns the field set kind identifier.
func (c *ComposerAuditLogResourceFieldSet) Kind() string {
	return "composer_audit"
}

var _ log.FieldSet = (*ComposerAuditLogResourceFieldSet)(nil)

// ExtractComposerAuditLogResource extracts Composer resource fields from a NodeReader.
func ExtractComposerAuditLogResource(reader *structured.NodeReader) (ComposerAuditLogResourceFieldSet, error) {
	if mock, ok := structured.GetMock[ComposerAuditLogResourceFieldSet](reader); ok {
		return mock, nil
	}
	var result ComposerAuditLogResourceFieldSet
	result.EnvironmentName = reader.ReadStringOrDefaultByPath(pathEnvironmentName, "unknown")
	result.Location = reader.ReadStringOrDefaultByPath(pathLocation, "unknown")
	result.ProjectID = reader.ReadStringOrDefaultByPath(pathProjectID, "unknown")
	return result, nil
}

// ComposerAuditLogResourceFieldSetReader parses resource identification fields from Cloud Composer audit logs.
type ComposerAuditLogResourceFieldSetReader struct{}

// FieldSetKind returns the kind identifier of the field set read by this reader.
func (r *ComposerAuditLogResourceFieldSetReader) FieldSetKind() string {
	return (&ComposerAuditLogResourceFieldSet{}).Kind()
}

// Read extracts Composer resource labels from the provided log node reader.
func (r *ComposerAuditLogResourceFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	result, err := ExtractComposerAuditLogResource(reader)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

var _ log.FieldSetReader = (*ComposerAuditLogResourceFieldSetReader)(nil)
