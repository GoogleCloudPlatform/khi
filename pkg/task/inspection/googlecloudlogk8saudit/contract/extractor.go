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
	pathProtoPrincipal      = structured.CompileFieldPath("protoPayload.authenticationInfo.principalEmail")
	pathProtoStatusCode     = structured.CompileFieldPath("protoPayload.status.code")
	pathProtoStatusMessage  = structured.CompileFieldPath("protoPayload.status.message")
	pathProtoRequest        = structured.CompileFieldPath("protoPayload.request")
	pathProtoResponse       = structured.CompileFieldPath("protoPayload.response")
	pathLabels              = structured.CompileFieldPath("labels")
)

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

	type roundIndex struct{ round, index int }
	webhookResults := make(map[roundIndex]*commonlogk8saudit_contract.MutatingWebhookResult)

	getOrCreateResult := func(round, index int) *commonlogk8saudit_contract.MutatingWebhookResult {
		ri := roundIndex{round, index}
		if res, ok := webhookResults[ri]; ok {
			return res
		}
		res := &commonlogk8saudit_contract.MutatingWebhookResult{
			Round: round,
			Index: index,
		}
		webhookResults[ri] = res
		return res
	}

	labelsReader, _ := reader.GetReader(pathLabels)
	if labelsReader != nil {
		labelsReader.Children()(func(key structured.NodeChildrenKey, value structured.NodeReader) bool {
			keyStr := key.Key
			var prefix string
			switch {
			case strings.HasPrefix(keyStr, commonlogk8saudit_contract.MutatingWebhookMutationPrefix+"round_"):
				prefix = commonlogk8saudit_contract.MutatingWebhookMutationPrefix
			case strings.HasPrefix(keyStr, commonlogk8saudit_contract.MutatingWebhookPatchPrefix+"round_"):
				prefix = commonlogk8saudit_contract.MutatingWebhookPatchPrefix
			case strings.HasPrefix(keyStr, commonlogk8saudit_contract.MutatingWebhookFailedOpenPrefix+"round_"):
				prefix = commonlogk8saudit_contract.MutatingWebhookFailedOpenPrefix
			}

			if prefix != "" {
				suffix := strings.TrimPrefix(keyStr, prefix+"round_")
				parts := strings.Split(suffix, "_index_")
				if len(parts) == 2 {
					round, err1 := strconv.Atoi(parts[0])
					index, err2 := strconv.Atoi(parts[1])
					if err1 == nil && err2 == nil {
						res := getOrCreateResult(round, index)
						valStr, err := value.ReadString(structured.EmptyFieldPath)
						if err == nil {
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
					}
				}
			}
			return true
		})
	}

	for _, res := range webhookResults {
		result.MutatingWebhookResults = append(result.MutatingWebhookResults, res)
	}

	return result, nil
}

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
