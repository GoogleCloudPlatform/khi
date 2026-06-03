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
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

type OSSK8sAuditLogFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (o *OSSK8sAuditLogFieldSetReader) FieldSetKind() string {
	return (&commonlogk8saudit_contract.K8sAuditLogFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (o *OSSK8sAuditLogFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result commonlogk8saudit_contract.K8sAuditLogFieldSet
	result.OperationID = reader.ReadStringOrDefault("auditID", "")
	// Currently this won't support the long running operation. TODO: support long runnning operation
	result.IsFirst = true
	result.IsLast = true
	apiGroup := reader.ReadStringOrDefault("objectRef.apiGroup", "core")
	apiVersion := reader.ReadStringOrDefault("objectRef.apiVersion", "unknown")
	kind := reader.ReadStringOrDefault("objectRef.resource", "unknown")
	namespace := reader.ReadStringOrDefault("objectRef.namespace", "cluster-scope")
	name := reader.ReadStringOrDefault("objectRef.name", "unknown")
	subresource := reader.ReadStringOrDefault("objectRef.subresource", "")
	verb := reader.ReadStringOrDefault("verb", "")

	if name == "unknown" && verb == "create" {
		// the name may be generated from the server side.
		name = reader.ReadStringOrDefault("responseObject.metadata.name", "unknown")
	}

	result.APIVersion = fmt.Sprintf("%s/%s", apiGroup, apiVersion)
	result.PluralKind = kind
	result.Namespace = namespace
	result.ResourceName = name
	result.SubresourceName = subresource
	result.ClusterName = "cluster"
	result.Verb = verbStringToVerb(verb)

	result.RequestURI = reader.ReadStringOrDefault("requestURI", "")
	result.Principal = reader.ReadStringOrDefault("user.username", "unknown")
	result.StatusCode = reader.ReadIntOrDefault("responseStatus.code", 0)
	result.StatusMessage = reader.ReadStringOrDefault("responseStatus.message", "")
	result.IsError = result.StatusCode < 200 || result.StatusCode >= 300
	result.Request, _ = reader.GetReader("requestObject")
	result.Response, _ = reader.GetReader("responseObject")
	return &result, nil
}

var _ log.FieldSetReader = (*OSSK8sAuditLogFieldSetReader)(nil)

// OSSK8sAuditLogCommonFieldSetReader implements log.FieldSetReader for log.CommonFieldSet{}.
type OSSK8sAuditLogCommonFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (o *OSSK8sAuditLogCommonFieldSetReader) FieldSetKind() string {
	return (&log.CommonFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (o *OSSK8sAuditLogCommonFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var err error
	result := &log.CommonFieldSet{}
	result.DisplayID = reader.ReadStringOrDefault("auditID", "unknown")
	result.Timestamp, err = reader.ReadTimestamp("stageTimestamp")
	if err != nil {
		return nil, fmt.Errorf("failed to read timestamp from given log")
	}
	result.Severity = enum.SeverityUnknown // TODO: handle OSS k8s audit log severity properly
	return result, nil
}

var _ log.FieldSetReader = (*OSSK8sAuditLogCommonFieldSetReader)(nil)

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

// Kind returns the kind of this FieldSet.
func (o *OSSK8sEventFieldSet) Kind() string {
	return "ossclusterk8s.khi.google.com/EventFieldSet"
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

var _ log.FieldSet = (*OSSK8sEventFieldSet)(nil)

// OSSK8sEventFieldSetReader extracts OSSK8sEventFieldSet from a raw log.
type OSSK8sEventFieldSetReader struct{}

// FieldSetKind returns the Kind of the field set this reader constructs.
func (o *OSSK8sEventFieldSetReader) FieldSetKind() string {
	return (&OSSK8sEventFieldSet{}).Kind()
}

// Read extracts event fields from `responseObject` of the Event log.
func (o *OSSK8sEventFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result OSSK8sEventFieldSet
	result.APIVersion = reader.ReadStringOrDefault("responseObject.involvedObject.apiVersion", "core/v1")
	if !strings.Contains(result.APIVersion, "/") {
		result.APIVersion = "core/" + result.APIVersion
	}
	result.ResourceKind = strings.ToLower(reader.ReadStringOrDefault("responseObject.involvedObject.kind", "unknown"))
	result.Namespace = reader.ReadStringOrDefault("responseObject.involvedObject.namespace", "cluster-scope")
	result.Resource = reader.ReadStringOrDefault("responseObject.involvedObject.name", "unknown")
	result.Subresource = reader.ReadStringOrDefault("responseObject.involvedObject.subresource", "")
	result.Reason = reader.ReadStringOrDefault("responseObject.reason", "???")
	result.Message = reader.ReadStringOrDefault("responseObject.message", "")
	return &result, nil
}

var _ log.FieldSetReader = (*OSSK8sEventFieldSetReader)(nil)
