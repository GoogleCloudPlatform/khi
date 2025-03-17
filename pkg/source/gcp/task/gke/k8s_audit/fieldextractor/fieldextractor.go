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

package fieldextractor

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/rtype"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/k8s"
)

type GCPAuditLogFieldExtractor struct{}

// ExtractFields implements common.AuditLogFieldExtractor.
func (g *GCPAuditLogFieldExtractor) ExtractFields(ctx context.Context, l *log.LogEntity) (*types.AuditLogParserInput, error) {
	resourceName, err := l.GetString("protoPayload.resourceName")
	if err != nil {
		return nil, err
	}

	methodName, err := l.GetString("protoPayload.methodName")
	if err != nil {
		return nil, err
	}

	userEmail := l.GetStringOrDefault("protoPayload.authenticationInfo.principalEmail", "")

	operation := k8s.ParseKubernetesOperation(resourceName, methodName)
	// /status subresource contains the actual content of the parent.
	// It's easier to see timeline merged with the parent timeline instead of showing status as the single subresource timeline.
	// TODO: There would be the other subresources needed to be cared like this.
	if operation.SubResourceName == "status" {
		operation.SubResourceName = ""
	}

	responseErrorCode := l.GetIntOrDefault("protoPayload.status.code", 0)
	responseErrorMessage := l.GetStringOrDefault("protoPayload.status.message", "")

	requestType := rtype.RTypeUnknown
	request, _ := l.Fields.ReaderSingle("protoPayload.request")
	if request != nil && request.Has("@type") {
		rtypeInStr := request.ReadStringOrDefault("@type", "")
		if rt, found := rtype.AtTypesOnGCPAuditLog[rtypeInStr]; found {
			requestType = rt
		}
	}

	responseType := rtype.RTypeUnknown
	response, _ := l.Fields.ReaderSingle("protoPayload.response")
	if response != nil && response.Has("@type") {
		rtypeInStr := response.ReadStringOrDefault("@type", "")
		if rt, found := rtype.AtTypesOnGCPAuditLog[rtypeInStr]; found {
			responseType = rt
		}
	}

	return &types.AuditLogParserInput{
		Log:                  l,
		Requestor:            userEmail,
		Operation:            operation,
		ResponseErrorCode:    responseErrorCode,
		ResponseErrorMessage: responseErrorMessage,
		Request:              request,
		RequestType:          requestType,
		Response:             response,
		ResponseType:         responseType,
		IsErrorResponse:      responseErrorCode != 0, // GCP audit log response code is gRPC error code. non zero codes are regarded as an error.
	}, nil
}

var _ types.AuditLogFieldExtractor = (*GCPAuditLogFieldExtractor)(nil)
