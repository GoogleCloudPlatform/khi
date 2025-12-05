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

package ossclusterk8s_contract

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

type OSSK8sAuditLogFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (o *OSSK8sAuditLogFieldSetReader) FieldSetKind() string {
	return (&commonlogk8sauditv2_contract.K8sAuditLogFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (o *OSSK8sAuditLogFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result commonlogk8sauditv2_contract.K8sAuditLogFieldSet
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

	result.K8sOperation = &model.KubernetesObjectOperation{
		APIVersion:      fmt.Sprintf("%s/%s", apiGroup, apiVersion),
		PluralKind:      kind,
		Namespace:       namespace,
		Name:            name,
		SubResourceName: subresource,
		Verb:            verbStringToEnum(verb),
	}

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
		return nil, fmt.Errorf("failed to read timestmap from given log")
	}
	result.Severity = enum.SeverityUnknown // TODO: handle OSS k8s audit log severity properly
	return result, nil
}

var _ log.FieldSetReader = (*OSSK8sAuditLogCommonFieldSetReader)(nil)

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
		return enum.RevisionVerbDeleteCollection
	default:
		// Add verbs for get/list/watch
		return enum.RevisionVerbUnknown
	}
}
