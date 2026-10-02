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

package k8saudit

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

// K8sAuditLogCacheKey identifies the cached K8sAuditLogFieldSet on a NodeReader.
var K8sAuditLogCacheKey = structured.NewCacheKey[*K8sAuditLogFieldSet]()

// K8sAuditLogExtractor is a function type for extracting K8sAuditLogFieldSet from a NodeReader.
type K8sAuditLogExtractor func(reader *structured.NodeReader) (*K8sAuditLogFieldSet, error)

// ExtractK8sAuditLog extracts K8s audit log data from a NodeReader using the given extractor.
func ExtractK8sAuditLog(reader *structured.NodeReader, extractor K8sAuditLogExtractor) (*K8sAuditLogFieldSet, error) {
	if mock, ok := structured.GetMock[*K8sAuditLogFieldSet](reader); ok {
		return mock, nil
	}
	if cached, ok := structured.GetCache(reader, K8sAuditLogCacheKey); ok {
		return cached, nil
	}
	res, err := extractor(reader)
	if err == nil && res != nil {
		structured.SetCache(reader, K8sAuditLogCacheKey, res)
	}
	return res, err
}

// K8sAuditLogErrorExtractor is a function type for extracting whether a log represents an error from a NodeReader.
type K8sAuditLogErrorExtractor func(reader *structured.NodeReader) (bool, error)

// ExtractK8sAuditLogError extracts whether the K8s audit log is an error using the given error extractor.
func ExtractK8sAuditLogError(reader *structured.NodeReader, extractor K8sAuditLogErrorExtractor) (bool, error) {
	if mock, ok := structured.GetMock[*K8sAuditLogFieldSet](reader); ok {
		return mock.IsError, nil
	}
	if cached, ok := structured.GetCache(reader, K8sAuditLogCacheKey); ok {
		return cached.IsError, nil
	}
	return extractor(reader)
}
