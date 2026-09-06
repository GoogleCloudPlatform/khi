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
	"encoding/json"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

var (
	pathOperationID         = structured.CompileFieldPath("operation.id")
	pathOperationFirst      = structured.CompileFieldPath("operation.first")
	pathOperationLast       = structured.CompileFieldPath("operation.last")
	pathProtoResourceName   = structured.CompileFieldPath("protoPayload.resourceName")
	pathProtoMethodName     = structured.CompileFieldPath("protoPayload.methodName")
	pathResourceClusterName = structured.CompileFieldPath("resource.labels.cluster_name")
	pathLabelsDryRun        = structured.CompileFieldPath("labels.command\\.gke\\.io/dryRun")
	pathLabelsTruncated     = structured.CompileFieldPath("labels.audit\\.k8s\\.io/truncated")
	pathProtoPrincipal      = structured.CompileFieldPath("protoPayload.authenticationInfo.principalEmail")
	pathProtoStatusCode     = structured.CompileFieldPath("protoPayload.status.code")
	pathProtoStatusMessage  = structured.CompileFieldPath("protoPayload.status.message")
	pathProtoRequest        = structured.CompileFieldPath("protoPayload.request")
	pathProtoResponse       = structured.CompileFieldPath("protoPayload.response")
	pathLabels              = structured.CompileFieldPath("labels")
)

// ExtractGCPK8sAuditLogError extracts whether the log represents an error from a GCP Cloud Logging NodeReader.
func ExtractGCPK8sAuditLogError(reader *structured.NodeReader) (bool, error) {
	if mock, ok := structured.GetMock[commonlogk8saudit_contract.K8sAuditLogFieldSet](reader); ok {
		return mock.IsError, nil
	}
	if reader == nil {
		return false, nil
	}
	return reader.ReadIntOrDefault(pathProtoStatusCode, 0) != 0, nil
}

// ExtractGCPK8sAuditLog extracts Kubernetes audit log data from a GCP Cloud Logging NodeReader.
func ExtractGCPK8sAuditLog(reader *structured.NodeReader) (commonlogk8saudit_contract.K8sAuditLogFieldSet, error) {
	if mock, ok := structured.GetMock[commonlogk8saudit_contract.K8sAuditLogFieldSet](reader); ok {
		return mock, nil
	}
	var result commonlogk8saudit_contract.K8sAuditLogFieldSet
	if reader == nil {
		return result, nil
	}

	result.OperationID = reader.ReadStringOrDefault(pathOperationID, "")
	result.IsFirst = reader.ReadBoolOrDefault(pathOperationFirst, false)
	result.IsLast = reader.ReadBoolOrDefault(pathOperationLast, false)
	resourceName := reader.ReadStringOrDefault(pathProtoResourceName, "")
	methodName := reader.ReadStringOrDefault(pathProtoMethodName, "")
	result.ClusterName = reader.ReadStringOrDefault(pathResourceClusterName, "unknown")
	result.RequestURI = resourceName
	result.IsDryRun = reader.ReadStringOrDefault(pathLabelsDryRun, "") != ""
	result.IsTruncated = reader.ReadStringOrDefault(pathLabelsTruncated, "") == "true" || reader.ReadBoolOrDefault(pathLabelsTruncated, false)

	apiVersion, pluralKind, namespace, name, subResourceName, verb := parseKubernetesOperation(resourceName, methodName)
	result.APIVersion = apiVersion
	result.PluralKind = pluralKind
	result.Namespace = namespace
	result.ResourceName = name
	result.SubresourceName = subResourceName
	result.Verb = verb

	result.Principal = reader.ReadStringOrDefault(pathProtoPrincipal, "")
	result.StatusCode = reader.ReadIntOrDefault(pathProtoStatusCode, 0)
	result.StatusMessage = reader.ReadStringOrDefault(pathProtoStatusMessage, "")
	result.IsError = result.StatusCode != 0
	result.Request, _ = reader.GetReader(pathProtoRequest)
	result.Response, _ = reader.GetReader(pathProtoResponse)
	labelsReader, _ := reader.GetReader(pathLabels)
	result.MutatingWebhookResults = extractMutatingWebhookResults(labelsReader)

	return result, nil
}

// mutatingWebhookKey identifies a mutating webhook result by its round and execution index.
type mutatingWebhookKey struct {
	round int
	index int
}

// extractMutatingWebhookResults parses mutating webhook execution results recorded in the log labels.
func extractMutatingWebhookResults(labelsReader *structured.NodeReader) []*commonlogk8saudit_contract.MutatingWebhookResult {
	if labelsReader == nil {
		return nil
	}

	var webhookResults map[mutatingWebhookKey]*commonlogk8saudit_contract.MutatingWebhookResult

	labelsReader.Children()(func(childKey structured.NodeChildrenKey, childVal structured.NodeReader) bool {
		prefix, round, index, ok := parseWebhookLabelKey(childKey.Key)
		if !ok {
			return true
		}

		valStr, err := childVal.ReadString(structured.EmptyFieldPath)
		if err != nil {
			return true
		}

		if webhookResults == nil {
			webhookResults = make(map[mutatingWebhookKey]*commonlogk8saudit_contract.MutatingWebhookResult)
		}
		key := mutatingWebhookKey{round: round, index: index}
		res, exists := webhookResults[key]
		if !exists {
			res = &commonlogk8saudit_contract.MutatingWebhookResult{
				Round: round,
				Index: index,
			}
			webhookResults[key] = res
		}

		populateWebhookResult(res, prefix, valStr)
		return true
	})

	if len(webhookResults) == 0 {
		return nil
	}
	out := make([]*commonlogk8saudit_contract.MutatingWebhookResult, 0, len(webhookResults))
	for _, res := range webhookResults {
		out = append(out, res)
	}
	return out
}

// parseWebhookLabelKey extracts the admission webhook prefix, round, and index from a label key.
func parseWebhookLabelKey(key string) (prefix string, round int, index int, ok bool) {
	switch {
	case strings.HasPrefix(key, commonlogk8saudit_contract.MutatingWebhookMutationPrefix+"round_"):
		prefix = commonlogk8saudit_contract.MutatingWebhookMutationPrefix
	case strings.HasPrefix(key, commonlogk8saudit_contract.MutatingWebhookPatchPrefix+"round_"):
		prefix = commonlogk8saudit_contract.MutatingWebhookPatchPrefix
	case strings.HasPrefix(key, commonlogk8saudit_contract.MutatingWebhookFailedOpenPrefix+"round_"):
		prefix = commonlogk8saudit_contract.MutatingWebhookFailedOpenPrefix
	default:
		return "", 0, 0, false
	}

	suffix := strings.TrimPrefix(key, prefix+"round_")
	roundStr, indexStr, found := strings.Cut(suffix, "_index_")
	if !found {
		return "", 0, 0, false
	}
	r, err1 := strconv.Atoi(roundStr)
	idx, err2 := strconv.Atoi(indexStr)
	if err1 != nil || err2 != nil {
		return "", 0, 0, false
	}
	return prefix, r, idx, true
}

// populateWebhookResult populates the webhook result from a JSON payload or string value.
func populateWebhookResult(res *commonlogk8saudit_contract.MutatingWebhookResult, prefix string, valStr string) {
	switch prefix {
	case commonlogk8saudit_contract.MutatingWebhookMutationPrefix:
		var mutationInfo commonlogk8saudit_contract.MutatingWebhookMutationInfo
		if json.Unmarshal([]byte(valStr), &mutationInfo) == nil {
			res.Configuration = mutationInfo.Configuration
			res.Webhook = mutationInfo.Webhook
			res.Mutated = mutationInfo.Mutated
		}
	case commonlogk8saudit_contract.MutatingWebhookPatchPrefix:
		var patchInfo commonlogk8saudit_contract.MutatingWebhookPatchInfo
		if json.Unmarshal([]byte(valStr), &patchInfo) == nil {
			res.Patch = patchInfo.Patch
			if res.Configuration == "" {
				res.Configuration = patchInfo.Configuration
			}
			if res.Webhook == "" {
				res.Webhook = patchInfo.Webhook
			}
		}
	case commonlogk8saudit_contract.MutatingWebhookFailedOpenPrefix:
		res.FailedOpen = true
		if res.Webhook == "" {
			res.Webhook = valStr
		}
	}
}

// parseVerb parses the Kubernetes operation verb from methodName.
func parseVerb(methodName string) *pb.Verb {
	verbStr := methodName
	if lastDot := strings.LastIndexByte(methodName, '.'); lastDot >= 0 {
		verbStr = methodName[lastDot+1:]
	}
	switch verbStr {
	case "create":
		return commonlogk8saudit_contract.VerbCreate
	case "update":
		return commonlogk8saudit_contract.VerbUpdate
	case "delete":
		return commonlogk8saudit_contract.VerbDelete
	case "deletecollection":
		return commonlogk8saudit_contract.VerbDeleteCollection
	case "patch":
		return commonlogk8saudit_contract.VerbPatch
	default:
		return commonlogk8saudit_contract.VerbUnknown
	}
}

// isNamespaceOperation checks if the method modifies the "Namespace" resource itself.
// In GCP audit logs, such methods have "namespaces" as their 5th dot-separated segment (e.g., "io.k8s.core.v1.namespaces.create").
func isNamespaceOperation(methodName string) bool {
	idx := 0
	for frag := range strings.SplitSeq(methodName, ".") {
		if idx == 4 {
			return frag == "namespaces"
		}
		idx++
	}
	return false
}

// parseResourceTarget parses the resource target fields from resourceName.
func parseResourceTarget(resourceName string, isNamespaceOp bool) (apiVersion, pluralKind, namespace, name, subResourceName string) {
	var frags [8]string
	numFrags := 0
	for frag := range strings.SplitSeq(resourceName, "/") {
		if numFrags < len(frags) {
			frags[numFrags] = frag
		}
		numFrags++
	}

	switch {
	case isNamespaceOp:
		namespace = "cluster-scope"
		pluralKind = "namespaces"
		if numFrags > 3 {
			name = frags[3]
		}
		if numFrags > 4 {
			subResourceName = frags[4]
		}
	case numFrags >= 5 && frags[2] == "namespaces":
		if numFrags > 3 {
			namespace = frags[3]
		}
		if numFrags > 4 {
			pluralKind = frags[4]
		}
		if numFrags > 5 {
			name = frags[5]
		}
		if numFrags > 6 {
			subResourceName = frags[6]
		}
	case numFrags >= 3:
		namespace = "cluster-scope"
		pluralKind = frags[2]
		if numFrags > 3 {
			name = frags[3]
		}
		if numFrags > 4 {
			subResourceName = frags[4]
		}
	}

	if numFrags >= 2 {
		apiVersion = resourceName[:len(frags[0])+1+len(frags[1])]
	}
	return
}

// parseKubernetesOperation parses the resourceName and methodName from a GCP audit log
// to determine the details of a Kubernetes API operation, returning split fields.
func parseKubernetesOperation(resourceName string, methodName string) (apiVersion, pluralKind, namespace, name, subResourceName string, verb *pb.Verb) {
	verb = parseVerb(methodName)
	isNamespaceOp := isNamespaceOperation(methodName)
	apiVersion, pluralKind, namespace, name, subResourceName = parseResourceTarget(resourceName, isNamespaceOp)
	return
}
