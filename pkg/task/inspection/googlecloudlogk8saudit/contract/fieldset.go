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

package googlecloudlogk8saudit_contract

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

type GCPK8sAuditLogFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (g *GCPK8sAuditLogFieldSetReader) FieldSetKind() string {
	return (&commonlogk8saudit_contract.K8sAuditLogFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (g *GCPK8sAuditLogFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result commonlogk8saudit_contract.K8sAuditLogFieldSet
	result.OperationID = reader.ReadStringOrDefault("operation.id", "")
	result.IsFirst = reader.ReadBoolOrDefault("operation.first", false)
	result.IsLast = reader.ReadBoolOrDefault("operation.last", false)
	resourceName := reader.ReadStringOrDefault("protoPayload.resourceName", "")
	methodName := reader.ReadStringOrDefault("protoPayload.methodName", "")
	result.ClusterName = reader.ReadStringOrDefault("resource.labels.cluster_name", "unknown")
	result.RequestURI = resourceName

	apiVersion, pluralKind, namespace, name, subResourceName, verb := parseKubernetesOperation(resourceName, methodName)
	result.APIVersion = apiVersion
	result.PluralKind = pluralKind
	result.Namespace = namespace
	result.ResourceName = name
	result.SubresourceName = subResourceName
	result.Verb = verb

	result.Principal = reader.ReadStringOrDefault("protoPayload.authenticationInfo.principalEmail", "")
	result.StatusCode = reader.ReadIntOrDefault("protoPayload.status.code", 0)
	result.StatusMessage = reader.ReadStringOrDefault("protoPayload.status.message", "")
	result.IsError = result.StatusCode != 0
	result.Request, _ = reader.GetReader("protoPayload.request")
	result.Response, _ = reader.GetReader("protoPayload.response")
	return &result, nil
}

var _ log.FieldSetReader = (*GCPK8sAuditLogFieldSetReader)(nil)

// parseKubernetesOperation parses the resourceName and methodName from a GCP audit log
// to determine the details of a Kubernetes API operation, returning split fields.
func parseKubernetesOperation(resourceName string, methodName string) (apiVersion, pluralKind, namespace, name, subResourceName string, verb *pb.Verb) {
	resourceNameFragments := strings.Split(resourceName, "/")
	methodNameFragments := strings.Split(methodName, ".")
	verbStr := methodNameFragments[len(methodNameFragments)-1]
	switch verbStr {
	case "create":
		verb = commonlogk8saudit_contract.VerbCreate
	case "update":
		verb = commonlogk8saudit_contract.VerbUpdate
	case "delete":
		verb = commonlogk8saudit_contract.VerbDelete
	case "deletecollection":
		verb = commonlogk8saudit_contract.VerbDeleteCollection
	case "patch":
		verb = commonlogk8saudit_contract.VerbPatch
	default:
		verb = commonlogk8saudit_contract.VerbUnknown
	}
	// Example methodName field formats:
	// namespaced resource: core/v1/namespaces/kube-system/pods/event-exporter-gke-787cd5d885-dqf4b
	// namespaced resource with subresource: core/v1/namespaces/kube-system/pods/event-exporter-gke-787cd5d885-dqf4b/status
	// cluster scoped resource:  core/v1/nodes/gke-p0-gke-basic-1-default-8a2ac49b-19tq
	// cluster scoped resource with subresource: core/v1/nodes/gke-p0-gke-basic-1-default-8a2ac49b-19tq/status
	// namespace resource: core/v1/namespaces/kube-system
	// namespace resource with subresource: core/v1/namespaces/kube-system/finalize
	switch {
	case len(methodNameFragments) > 4 && methodNameFragments[4] == "namespaces": // This log is to modify "Namespace" resource itself
		namespace = "cluster-scope"
		if len(resourceNameFragments) > 3 {
			name = resourceNameFragments[3]
		}
		pluralKind = "namespaces"
		if len(resourceNameFragments) > 4 {
			subResourceName = resourceNameFragments[4]
		}
	case len(resourceNameFragments) >= 5 && resourceNameFragments[2] == "namespaces":
		if len(resourceNameFragments) > 3 {
			namespace = resourceNameFragments[3]
		}
		if len(resourceNameFragments) > 4 {
			pluralKind = resourceNameFragments[4]
		}
		if len(resourceNameFragments) > 5 {
			name = resourceNameFragments[5]
		}
		if len(resourceNameFragments) > 6 {
			subResourceName = resourceNameFragments[6]
		}
	case len(resourceNameFragments) >= 3:
		namespace = "cluster-scope"
		if len(resourceNameFragments) > 3 {
			name = resourceNameFragments[3]
		}
		pluralKind = resourceNameFragments[2]
		if len(resourceNameFragments) > 4 {
			subResourceName = resourceNameFragments[4]
		}
	}
	if len(resourceNameFragments) >= 2 {
		apiVersion = resourceNameFragments[0] + "/" + resourceNameFragments[1]
	}
	return
}
