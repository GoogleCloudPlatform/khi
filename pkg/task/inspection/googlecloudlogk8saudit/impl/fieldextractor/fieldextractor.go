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

	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

// GCPAuditLogFieldExtractor extracts fields from a GCP Kubernetes audit log entry.
type GCPAuditLogFieldExtractor struct{}

// ExtractFields extracts relevant fields from a GCP audit log and populates an AuditLogParserInput.
// It implements the common.AuditLogFieldExtractor interface.
func (g *GCPAuditLogFieldExtractor) ExtractFields(ctx context.Context, l *log.Log) (*commonlogk8saudit_contract.AuditLogParserInput, error) {
	resourceName, err := l.ReadString("protoPayload.resourceName")
	if err != nil {
		return nil, err
	}

	methodName, err := l.ReadString("protoPayload.methodName")
	if err != nil {
		return nil, err
	}

	userEmail := l.ReadStringOrDefault("protoPayload.authenticationInfo.principalEmail", "")

	operation := parseKubernetesOperation(resourceName, methodName)
	// /status subresource contains the actual content of the parent.
	// It's easier to see timeline merged with the parent timeline instead of showing status as the single subresource timeline.
	// TODO: There would be the other subresources needed to be cared like this.
	if operation.SubResourceName == "status" {
		operation.SubResourceName = ""
	}

	responseErrorCode := l.ReadIntOrDefault("protoPayload.status.code", 0)
	responseErrorMessage := l.ReadStringOrDefault("protoPayload.status.message", "")

	requestType := commonlogk8saudit_contract.RTypeUnknown
	request, _ := l.GetReader("protoPayload.request")
	if request != nil && request.Has("@type") {
		rtypeInStr := request.ReadStringOrDefault("@type", "")
		if rt, found := commonlogk8saudit_contract.AtTypesOnGCPAuditLog[rtypeInStr]; found {
			requestType = rt
		}
	}

	responseType := commonlogk8saudit_contract.RTypeUnknown
	response, _ := l.GetReader("protoPayload.response")
	if response != nil && response.Has("@type") {
		rtypeInStr := response.ReadStringOrDefault("@type", "")
		if rt, found := commonlogk8saudit_contract.AtTypesOnGCPAuditLog[rtypeInStr]; found {
			responseType = rt
		}
	}

	return &commonlogk8saudit_contract.AuditLogParserInput{
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

var _ commonlogk8saudit_contract.AuditLogFieldExtractor = (*GCPAuditLogFieldExtractor)(nil)
