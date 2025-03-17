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

package types

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/rtype"
)

type AuditLogParserLogSource struct {
	Logs      []*log.LogEntity
	Extractor AuditLogFieldExtractor
}

// AuditLogParserInput is a type passed to ResourceSpecificParser from the prestep parser.
type AuditLogParserInput struct {
	// Current Log. Do not depend Log.Fields directly to avoid reading fields specific to log backend.
	Log *log.LogEntity
	// Requestor field of the log
	Requestor string
	// Kubernetes operation read from resource name and method name
	Operation *model.KubernetesObjectOperation
	// The request field of this log. This can be nil depending on the audit policy.
	Request     *structure.Reader
	RequestType rtype.Type
	// The response field of this log. This can be nil depending on the audit policy.
	Response     *structure.Reader
	ResponseType rtype.Type
	// Current resource body changed by this request.
	ResourceBodyReader *structure.Reader
	ResourceBodyYaml   string
	// The response code from the API server.
	IsErrorResponse      bool
	ResponseErrorCode    int
	ResponseErrorMessage string

	// RequestTarget is the address of target resource modified by this request.
	RequestTarget                          string
	GeneratedFromDeleteCollectionOperation bool
}

type TimelineGrouperResult struct {
	TimelineResourcePath string
	PreParsedLogs        []*AuditLogParserInput
}

// AuditLogFieldMapper handles log specific field mappings before passing AuditLogParserInput to the later parser steps.
type AuditLogFieldExtractor interface {
	ExtractFields(ctx context.Context, log *log.LogEntity) (*AuditLogParserInput, error)
}
