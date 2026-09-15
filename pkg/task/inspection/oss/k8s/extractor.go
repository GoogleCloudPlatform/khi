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

package ossk8s

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
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

	pathKind                     = structured.CompileFieldPath("kind")
	pathResponseObjectKind       = structured.CompileFieldPath("responseObject.kind")
	pathEventInvolvedAPIVersion  = structured.CompileFieldPath("responseObject.involvedObject.apiVersion")
	pathEventInvolvedKind        = structured.CompileFieldPath("responseObject.involvedObject.kind")
	pathEventInvolvedNamespace   = structured.CompileFieldPath("responseObject.involvedObject.namespace")
	pathEventInvolvedName        = structured.CompileFieldPath("responseObject.involvedObject.name")
	pathEventInvolvedSubresource = structured.CompileFieldPath("responseObject.involvedObject.subresource")
	pathEventReason              = structured.CompileFieldPath("responseObject.reason")
	pathEventMessage             = structured.CompileFieldPath("responseObject.message")
)

// ExtractOSSK8sIsEventAuditLog extracts whether the log is an OSS K8s audit log for an Event resource.
func ExtractOSSK8sIsEventAuditLog(reader *structured.NodeReader) (bool, error) {
	if _, ok := structured.GetMock[OSSK8sEventFieldSet](reader); ok {
		return true, nil
	}
	if reader == nil {
		return false, nil
	}
	return reader.ReadStringOrDefault(pathKind, "") == "Event" && reader.ReadStringOrDefault(pathResponseObjectKind, "") == "Event", nil
}

// ExtractOSSK8sIsNonEventAuditLog extracts whether the log is an OSS K8s audit log for a non-event resource.
func ExtractOSSK8sIsNonEventAuditLog(reader *structured.NodeReader) (bool, error) {
	if _, ok := structured.GetMock[*k8saudit.K8sAuditLogFieldSet](reader); ok {
		return true, nil
	}
	if reader == nil {
		return false, nil
	}
	verb := reader.ReadStringOrDefault(pathVerb, "")
	if reader.ReadStringOrDefault(pathKind, "") == "Event" && reader.ReadStringOrDefault(pathResponseObjectKind, "") != "Event" && reader.Has(pathObjectRef) {
		if verb == "" || verb == "get" || verb == "watch" || verb == "list" {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

// ExtractOSSK8sAuditLogError extracts whether an OSS audit log is an error.
func ExtractOSSK8sAuditLogError(reader *structured.NodeReader) (bool, error) {
	if mock, ok := structured.GetMock[*k8saudit.K8sAuditLogFieldSet](reader); ok {
		return mock.IsError, nil
	}
	if cached, ok := structured.GetCache(reader, k8saudit.K8sAuditLogCacheKey); ok {
		return cached.IsError, nil
	}
	if !reader.Has(pathAuditID) && !reader.Has(pathObjectRef) {
		return false, nil
	}
	statusCode := reader.ReadIntOrDefault(pathResponseStatusCode, 0)
	return statusCode < 200 || statusCode >= 300, nil
}

// ExtractOSSK8sAuditLog extracts k8saudit.K8sAuditLogFieldSet from OSS audit log entries.
func ExtractOSSK8sAuditLog(reader *structured.NodeReader) (*k8saudit.K8sAuditLogFieldSet, error) {
	if cached, ok := structured.GetCache(reader, k8saudit.K8sAuditLogCacheKey); ok {
		return cached, nil
	}
	if mock, ok := structured.GetMock[*k8saudit.K8sAuditLogFieldSet](reader); ok {
		return mock, nil
	}
	if !reader.Has(pathAuditID) && !reader.Has(pathObjectRef) {
		return &k8saudit.K8sAuditLogFieldSet{}, nil
	}

	result := &k8saudit.K8sAuditLogFieldSet{}
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

	structured.SetCache(reader, k8saudit.K8sAuditLogCacheKey, result)
	return result, nil
}

func verbStringToVerb(verbStr string) *pb.Verb {
	switch verbStr {
	case "create":
		return k8saudit.VerbCreate
	case "update":
		return k8saudit.VerbUpdate
	case "patch":
		return k8saudit.VerbPatch
	case "delete":
		return k8saudit.VerbDelete
	case "deletecollection":
		return k8saudit.VerbDeleteCollection
	default:
		return k8saudit.VerbUnknown
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
func (o *OSSK8sEventFieldSet) ResourceIdentity() *k8saudit.ResourceIdentity {
	return &k8saudit.ResourceIdentity{
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
