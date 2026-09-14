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

package googlecloudcaik8s_impl

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
)

// ConvertTemporalAssetToClusterResourceSnapshot converts a CAI TemporalAsset into a ClusterResourceSnapshot.
func ConvertTemporalAssetToClusterResourceSnapshot(ta *assetpb.TemporalAsset) (*googlecloudcaik8s_contract.ClusterResourceSnapshot, error) {
	if ta == nil || ta.Asset == nil {
		return nil, errors.New("temporal asset and underlying asset must not be nil")
	}

	var startTime time.Time
	if ta.Window != nil && ta.Window.StartTime != nil {
		startTime = ta.Window.StartTime.AsTime()
	}

	return &googlecloudcaik8s_contract.ClusterResourceSnapshot{
		TemporalAsset: ta,
		StartTime:     startTime,
	}, nil
}

// resolveResourceIdentity resolves and normalizes a ResourceIdentity from manifest fields with asset fallbacks.
func resolveResourceIdentity(assetName, assetType, manifestAPIVersion, manifestKind, manifestName, manifestNamespace string) *commonlogk8saudit_contract.ResourceIdentity {
	parsedNamespace, parsedName := parseAssetName(assetName)
	name := manifestName
	if name == "" {
		name = parsedName
	}
	namespace := manifestNamespace
	if namespace == "" {
		namespace = parsedNamespace
	}
	kind := manifestKind
	if kind == "" && assetType != "" {
		kind = parseKindFromAssetType(assetType)
	}
	kind = strings.ToLower(kind)
	if kind == "namespace" {
		namespace = ""
	}

	return &commonlogk8saudit_contract.ResourceIdentity{
		APIVersion: normalizeAPIVersion(manifestAPIVersion, assetType),
		Kind:       kind,
		Name:       name,
		Namespace:  namespace,
	}
}

// parseAssetName extracts namespace and resource name from a CAI asset name.
func parseAssetName(assetName string) (namespace, name string) {
	k8sPart := assetName
	if idx := strings.Index(assetName, "/k8s/"); idx != -1 {
		k8sPart = assetName[idx+len("/k8s/"):]
	}
	parts := strings.Split(k8sPart, "/")
	for i := 0; i < len(parts); i++ {
		if parts[i] == "namespaces" && i+1 < len(parts) {
			if i+3 <= len(parts) {
				return parts[i+1], parts[len(parts)-1]
			}
			if i+1 == len(parts)-1 {
				return "", parts[i+1]
			}
		}
	}
	return "", parts[len(parts)-1]
}

// parseKindFromAssetType extracts the Kind name from a CAI asset type.
func parseKindFromAssetType(assetType string) string {
	parts := strings.Split(assetType, "/")
	return parts[len(parts)-1]
}

// normalizeAPIVersion standardizes the apiVersion string, ensuring group prefixes for core resources.
func normalizeAPIVersion(apiVersion, assetType string) string {
	if apiVersion != "" {
		if !strings.Contains(apiVersion, "/") {
			return "core/" + apiVersion
		}
		return apiVersion
	}
	group := ""
	if assetType != "" {
		parts := strings.Split(assetType, "/")
		if len(parts) > 1 {
			group = parts[0]
		}
	}
	if group == "k8s.io" || group == "core" || group == "" {
		return "core/v1"
	}
	return group + "/v1"
}

// resolveManifestAPIVersion constructs the canonical Kubernetes manifest apiVersion string.
func resolveManifestAPIVersion(version, assetType string) string {
	group := ""
	if parts := strings.Split(assetType, "/"); len(parts) > 1 {
		group = parts[0]
	}
	switch group {
	case "apps.k8s.io":
		group = "apps"
	case "batch.k8s.io":
		group = "batch"
	case "policy.k8s.io":
		group = "policy"
	case "autoscaling.k8s.io":
		group = "autoscaling"
	}

	if version == "" {
		version = "v1"
	}

	if group == "" || group == "k8s.io" || group == "core" {
		return version
	}
	if strings.Contains(version, "/") {
		return version
	}
	return group + "/" + version
}

// restoreManifestTypeMeta populates apiVersion and kind into the resource data map if they are missing.
func restoreManifestTypeMeta(m map[string]any) {
	assetMap, ok := m["asset"].(map[string]any)
	if !ok {
		return
	}
	assetType, _ := assetMap["assetType"].(string)
	resourceMap, ok := assetMap["resource"].(map[string]any)
	if !ok {
		return
	}
	version, _ := resourceMap["version"].(string)
	dataMap, ok := resourceMap["data"].(map[string]any)
	if !ok || dataMap == nil {
		return
	}

	if _, hasKind := dataMap["kind"]; !hasKind && assetType != "" {
		kind := parseKindFromAssetType(assetType)
		if kind != "" {
			dataMap["kind"] = kind
		}
	}

	if _, hasAPIVersion := dataMap["apiVersion"]; !hasAPIVersion && assetType != "" {
		resolvedAPIVersion := resolveManifestAPIVersion(version, assetType)
		if resolvedAPIVersion != "" {
			dataMap["apiVersion"] = resolvedAPIVersion
		}
	}
}

// extractResourceData extracts the resource data map from the unmarshaled temporal asset map.
func extractResourceData(m map[string]any) map[string]any {
	assetMap, ok := m["asset"].(map[string]any)
	if !ok {
		return nil
	}
	resourceMap, ok := assetMap["resource"].(map[string]any)
	if !ok {
		return nil
	}
	dataMap, ok := resourceMap["data"].(map[string]any)
	if !ok {
		return nil
	}
	return dataMap
}

// restoreManifestMetadata decodes encoded metadata fields and removes empty default strings.
func restoreManifestMetadata(m map[string]any) {
	resourceData := extractResourceData(m)
	if resourceData == nil {
		return
	}
	metadata, ok := resourceData["metadata"].(map[string]any)
	if !ok || metadata == nil {
		return
	}

	for _, field := range []string{"clusterName", "selfLink", "generateName", "namespace"} {
		if strVal, ok := metadata[field].(string); ok && strVal == "" {
			delete(metadata, field)
		}
	}

	managedFields, ok := metadata["managedFields"].([]any)
	if !ok {
		return
	}
	for _, entry := range managedFields {
		if entryMap, ok := entry.(map[string]any); ok {
			restoreManagedFieldsEntry(entryMap)
		}
	}
}

// restoreManagedFieldsEntry decodes fieldsV1 JSON and removes empty default strings in a managedFields entry.
func restoreManagedFieldsEntry(entry map[string]any) {
	if strVal, ok := entry["subresource"].(string); ok && strVal == "" {
		delete(entry, "subresource")
	}
	fieldsV1, ok := entry["fieldsV1"].(map[string]any)
	if !ok {
		return
	}
	rawStr, ok := fieldsV1["Raw"].(string)
	if !ok || rawStr == "" {
		return
	}
	decodedBytes, err := base64.StdEncoding.DecodeString(rawStr)
	if err != nil {
		return
	}
	var decodedFields map[string]any
	if err := json.Unmarshal(decodedBytes, &decodedFields); err == nil {
		entry["fieldsV1"] = decodedFields
	}
}
