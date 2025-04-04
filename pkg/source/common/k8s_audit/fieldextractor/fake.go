// Copyright 2025 Google LLC
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

package common_k8saudit_fieldextactor

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
)

type StubFieldExtractor struct {
	Extractor func(ctx context.Context, log *log.LogEntity) (*types.AuditLogParserInput, error)
}

// ExtractFields implements types.AuditLogFieldExtractor.
func (f *StubFieldExtractor) ExtractFields(ctx context.Context, log *log.LogEntity) (*types.AuditLogParserInput, error) {
	return f.Extractor(ctx, log)
}

var _ types.AuditLogFieldExtractor = (*StubFieldExtractor)(nil)
