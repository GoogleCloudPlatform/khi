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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/rtype"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
)

type OSSJSONLAuditLogFieldExtractor struct{}

// ExtractFields implements common.AuditLogFieldExtractor.
func (g *OSSJSONLAuditLogFieldExtractor) ExtractFields(ctx context.Context, l *log.Log) (*types.AuditLogParserInput, error) {
	apiGroup := l.ReadStringOrDefault("objectRef.apiGroup", "core")
	apiVersion := l.ReadStringOrDefault("objectRef.apiVersion", "unknown")
	kind := l.ReadStringOrDefault("objectRef.resource", "unknown")
	namespace := l.ReadStringOrDefault("objectRef.namespace", "cluster-scope")
	name := l.ReadStringOrDefault("objectRef.name", "unknown")
	subresource := l.ReadStringOrDefault("objectRef.subresource", "")
	verb := l.ReadStringOrDefault("verb", "")

	if subresource == "status" {
		subresource = "" // status subresource response should contain the full body data of its parent
	}
	if name == "unknown" && verb == "create" {
		// the name may be generated from the server side.
		name = l.ReadStringOrDefault("responseObject.metadata.name", "unknown")
	}

	k8sOp := model.KubernetesObjectOperation{
		APIVersion:      fmt.Sprintf("%s/%s", apiGroup, apiVersion),
		PluralKind:      kind,
		Namespace:       namespace,
		Name:            name,
		SubResourceName: subresource,
		Verb:            verbStringToEnum(verb),
	}

	requestor := l.ReadStringOrDefault("user.username", "unknown")

	responseCode := l.ReadIntOrDefault("responseStatus.code", 0)
	responseMessage := l.ReadStringOrDefault("responseStatus.message", "")

	// response, request can be nil when the fields are missing, error can be ignorable.
	response, _ := l.GetReader("responseObject")
	responseType := rtype.RTypeUnknown
	if response != nil {
		responseType = rtype.RtypeFromOSSK8sObject(response)
	}
	request, _ := l.GetReader("requestObject")
	requestType := rtype.RTypeUnknown
	if request != nil {
		requestType = rtype.RtypeFromOSSK8sObject(request)
	}

	return &types.AuditLogParserInput{
		Log:                  l,
		Requestor:            requestor,
		Operation:            &k8sOp,
		ResponseErrorCode:    responseCode,
		ResponseErrorMessage: responseMessage,
		IsErrorResponse:      responseCode >= 400, // The response code is HTTP response code. Treat 4XX,5XX as error code.
		RequestType:          requestType,
		Request:              request,
		ResponseType:         responseType,
		Response:             response,
	}, nil
}

func verbStringToEnum(verbStr string) enum.RevisionVerb {
	switch verbStr {
	case "create":
		return enum.RevisionVerbCreate
	case "update":
		return enum.RevisionVerbUpdate
	case "patch":
		return enum.RevisionVerbPatch
	case "delete":
		return enum.RevisionVerbDelete
	case "deletecollection":
		return enum.RevisionVerbDelete
	default:
		// Add verbs for get/list/watch
		return enum.RevisionVerbUpdate
	}
}

var _ types.AuditLogFieldExtractor = (*OSSJSONLAuditLogFieldExtractor)(nil)
