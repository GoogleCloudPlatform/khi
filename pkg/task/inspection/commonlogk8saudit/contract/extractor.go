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
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

// K8sAuditLogExtractor is a function type for extracting K8sAuditLogFieldSet from a NodeReader.
type K8sAuditLogExtractor func(reader *structured.NodeReader) (K8sAuditLogFieldSet, error)

// ExtractK8sAuditLog extracts K8s audit log data from a NodeReader using the extractor in the task context.
func ExtractK8sAuditLog(ctx context.Context, reader *structured.NodeReader) (K8sAuditLogFieldSet, error) {
	if mock, ok := structured.GetMock[K8sAuditLogFieldSet](reader); ok {
		return mock, nil
	}
	if extractor, found := coretask.GetTaskResultOptional(ctx, K8sAuditLogExtractorRef); found && extractor != nil {
		return extractor(reader)
	}
	return K8sAuditLogFieldSet{}, nil
}

// K8sAuditLogErrorExtractor is a function type for extracting whether a log represents an error from a NodeReader.
type K8sAuditLogErrorExtractor func(reader *structured.NodeReader) (bool, error)

// ExtractK8sAuditLogError extracts whether the K8s audit log is an error using the error extractor in the task context.
func ExtractK8sAuditLogError(ctx context.Context, reader *structured.NodeReader) (bool, error) {
	if mock, ok := structured.GetMock[K8sAuditLogFieldSet](reader); ok {
		return mock.IsError, nil
	}
	if extractor, found := coretask.GetTaskResultOptional(ctx, K8sAuditLogErrorExtractorRef); found && extractor != nil {
		return extractor(reader)
	}
	return false, nil
}
