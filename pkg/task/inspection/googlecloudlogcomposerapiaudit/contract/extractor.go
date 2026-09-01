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

// ExtractComposerAuditLogResource extracts Composer resource fields from a NodeReader.
func ExtractComposerAuditLogResource(reader *structured.NodeReader) (ComposerAuditLogResourceFieldSet, error) {
	if mock, ok := structured.GetMock[ComposerAuditLogResourceFieldSet](reader); ok {
		return mock, nil
	}
	var result ComposerAuditLogResourceFieldSet
	result.EnvironmentName = reader.ReadStringOrDefault(pathEnvironmentName, "unknown")
	result.Location = reader.ReadStringOrDefault(pathLocation, "unknown")
	result.ProjectID = reader.ReadStringOrDefault(pathProjectID, "unknown")
	return result, nil
}
