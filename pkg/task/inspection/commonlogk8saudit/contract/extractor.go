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

package commonlogk8saudit_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

// K8sAuditLogExtractor is a function type for extracting K8sAuditLogFieldSet from a NodeReader.
type K8sAuditLogExtractor func(reader *structured.NodeReader) (K8sAuditLogFieldSet, error)

var defaultK8sAuditLogExtractor K8sAuditLogExtractor

// SetDefaultK8sAuditLogExtractor sets the default extractor for K8s audit logs.
func SetDefaultK8sAuditLogExtractor(extractor K8sAuditLogExtractor) {
	defaultK8sAuditLogExtractor = extractor
}

// ExtractK8sAuditLog extracts K8s audit log data from a NodeReader.
func ExtractK8sAuditLog(reader *structured.NodeReader) (K8sAuditLogFieldSet, error) {
	if mock, ok := structured.GetMock[K8sAuditLogFieldSet](reader); ok {
		return mock, nil
	}
	if defaultK8sAuditLogExtractor != nil {
		return defaultK8sAuditLogExtractor(reader)
	}
	return K8sAuditLogFieldSet{}, nil
}
