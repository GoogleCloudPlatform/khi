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

package ossclusterk8s_contract

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

var (
	pathAuditID               = structured.CompileFieldPath("auditID")
	pathAnnotationsTruncated  = structured.CompileFieldPath("annotations.audit\\.k8s\\.io/truncated")
	pathLabelsTruncated       = structured.CompileFieldPath("labels.audit\\.k8s\\.io/truncated")
	pathObjectRef             = structured.CompileFieldPath("objectRef")
	pathObjectRefAPIGroup     = structured.CompileFieldPath("objectRef.apiGroup")
	pathObjectRefAPIVersion   = structured.CompileFieldPath("objectRef.apiVersion")
	pathObjectRefResource     = structured.CompileFieldPath("objectRef.resource")
	pathObjectRefNamespace    = structured.CompileFieldPath("objectRef.namespace")
	pathObjectRefName         = structured.CompileFieldPath("objectRef.name")
	pathObjectRefSubresource  = structured.CompileFieldPath("objectRef.subresource")
	pathVerb                  = structured.CompileFieldPath("verb")
	pathResponseObjectName    = structured.CompileFieldPath("responseObject.metadata.name")
	pathRequestURI            = structured.CompileFieldPath("requestURI")
	pathUserUsername          = structured.CompileFieldPath("user.username")
	pathResponseStatusCode    = structured.CompileFieldPath("responseStatus.code")
	pathResponseStatusMessage = structured.CompileFieldPath("responseStatus.message")
	pathRequestObject         = structured.CompileFieldPath("requestObject")
	pathResponseObject        = structured.CompileFieldPath("responseObject")

	pathEventInvolvedAPIVersion  = structured.CompileFieldPath("responseObject.involvedObject.apiVersion")
	pathEventInvolvedKind        = structured.CompileFieldPath("responseObject.involvedObject.kind")
	pathEventInvolvedNamespace   = structured.CompileFieldPath("responseObject.involvedObject.namespace")
	pathEventInvolvedName        = structured.CompileFieldPath("responseObject.involvedObject.name")
	pathEventInvolvedSubresource = structured.CompileFieldPath("responseObject.involvedObject.subresource")
	pathEventReason              = structured.CompileFieldPath("responseObject.reason")
	pathEventMessage             = structured.CompileFieldPath("responseObject.message")
)

// ExtractOSSK8sAuditLog extracts commonlogk8saudit_contract.K8sAuditLogFieldSet from OSS audit log entries.
func ExtractOSSK8sAuditLog(reader *structured.NodeReader) (commonlogk8saudit_contract.K8sAuditLogFieldSet, error) {
	if mock, ok := structured.GetMock[commonlogk8saudit_contract.K8sAuditLogFieldSet](reader); ok {
		return mock, nil
	}
	if reader == nil || (!reader.Has(pathAuditID) && !reader.Has(pathObjectRef)) {
		return commonlogk8saudit_contract.K8sAuditLogFieldSet{}, nil
	}

	var result commonlogk8saudit_contract.K8sAuditLogFieldSet
	result.OperationID = reader.ReadStringOrDefault(pathAuditID, "")
	// Currently this won't support the long running operation. TODO: support long running operation
	result.IsFirst = true
	result.IsLast = true
	result.IsTruncated = reader.ReadStringOrDefault(pathAnnotationsTruncated, "") == "true" ||
		reader.ReadStringOrDefault(pathLabelsTruncated, "") == "true" ||
		reader.ReadBoolOrDefault(pathAnnotationsTruncated, false) ||
		reader.ReadBoolOrDefault(pathLabelsTruncated, false)
	apiGroup := reader.ReadStringOrDefault(pathObjectRefAPIGroup, "core")
	apiVersion := reader.ReadStringOrDefault(pathObjectRefAPIVersion, "unknown")
	kind := reader.ReadStringOrDefault(pathObjectRefResource, "unknown")
	namespace := reader.ReadStringOrDefault(pathObjectRefNamespace, "cluster-scope")
	name := reader.ReadStringOrDefault(pathObjectRefName, "unknown")
	subresource := reader.ReadStringOrDefault(pathObjectRefSubresource, "")
	verb := reader.ReadStringOrDefault(pathVerb, "")

	if name == "unknown" && verb == "create" {
		// the name may be generated from the server side.
		name = reader.ReadStringOrDefault(pathResponseObjectName, "unknown")
	}

	result.APIVersion = fmt.Sprintf("%s/%s", apiGroup, apiVersion)
	result.PluralKind = kind
	result.Namespace = namespace
	result.ResourceName = name
	result.SubresourceName = subresource
	result.ClusterName = "cluster"
	result.Verb = verbStringToVerb(verb)

	result.RequestURI = reader.ReadStringOrDefault(pathRequestURI, "")
	result.Principal = reader.ReadStringOrDefault(pathUserUsername, "unknown")
	result.StatusCode = reader.ReadIntOrDefault(pathResponseStatusCode, 0)
	result.StatusMessage = reader.ReadStringOrDefault(pathResponseStatusMessage, "")
	result.IsError = result.StatusCode < 200 || result.StatusCode >= 300
	result.Request, _ = reader.GetReader(pathRequestObject)
	result.Response, _ = reader.GetReader(pathResponseObject)
	return result, nil
}

func verbStringToVerb(verbStr string) *pb.Verb {
	switch verbStr {
	case "create":
		return commonlogk8saudit_contract.VerbCreate
	case "update":
		return commonlogk8saudit_contract.VerbUpdate
	case "patch":
		return commonlogk8saudit_contract.VerbPatch
	case "delete":
		return commonlogk8saudit_contract.VerbDelete
	case "deletecollection":
		return commonlogk8saudit_contract.VerbDeleteCollection
	default:
		return commonlogk8saudit_contract.VerbUnknown
	}
}

// OSSK8sEventFieldSet holds the structured data from a Kubernetes Event log.
type OSSK8sEventFieldSet struct {
	// APIVersion is the API version of the involved object.
	APIVersion string
	// ResourceKind is the kind of the involved object.
	ResourceKind string
	// Namespace is the namespace of the involved object.
	Namespace string
	// Resource is the name of the involved object.
	Resource string
	// Subresource is the subresource of the involved object.
	Subresource string
	// Reason is the short, machine-understandable string explaining why the event was triggered.
	Reason string
	// Message is the human-readable description of the status of this operation.
	Message string
}

// ResourceIdentity returns the ResourceIdentity representation of the involved object.
func (o *OSSK8sEventFieldSet) ResourceIdentity() *commonlogk8saudit_contract.ResourceIdentity {
	return &commonlogk8saudit_contract.ResourceIdentity{
		APIVersion:      o.APIVersion,
		Kind:            o.ResourceKind,
		Name:            o.Resource,
		Namespace:       o.Namespace,
		SubresourceName: o.Subresource,
	}
}

// ExtractOSSK8sEvent extracts event fields from `responseObject` of an Event log.
func ExtractOSSK8sEvent(reader *structured.NodeReader) (OSSK8sEventFieldSet, error) {
	if mock, ok := structured.GetMock[OSSK8sEventFieldSet](reader); ok {
		return mock, nil
	}
	var result OSSK8sEventFieldSet
	result.APIVersion = reader.ReadStringOrDefault(pathEventInvolvedAPIVersion, "core/v1")
	if !strings.Contains(result.APIVersion, "/") {
		result.APIVersion = "core/" + result.APIVersion
	}
	result.ResourceKind = strings.ToLower(reader.ReadStringOrDefault(pathEventInvolvedKind, "unknown"))
	result.Namespace = reader.ReadStringOrDefault(pathEventInvolvedNamespace, "cluster-scope")
	result.Resource = reader.ReadStringOrDefault(pathEventInvolvedName, "unknown")
	result.Subresource = reader.ReadStringOrDefault(pathEventInvolvedSubresource, "")
	result.Reason = reader.ReadStringOrDefault(pathEventReason, "???")
	result.Message = reader.ReadStringOrDefault(pathEventMessage, "")
	return result, nil
}
