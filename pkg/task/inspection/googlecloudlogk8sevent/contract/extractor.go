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

package googlecloudlogk8sevent_contract

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

var (
	pathProjectID    = structured.CompileFieldPath("resource.labels.project_id")
	pathClusterName  = structured.CompileFieldPath("resource.labels.cluster_name")
	pathTextPayload  = structured.CompileFieldPath("textPayload")
	pathKind         = structured.CompileFieldPath("jsonPayload.kind")
	pathAPIVersion   = structured.CompileFieldPath("jsonPayload.involvedObject.apiVersion")
	pathInvolvedKind = structured.CompileFieldPath("jsonPayload.involvedObject.kind")
	pathNamespace    = structured.CompileFieldPath("jsonPayload.involvedObject.namespace")
	pathResourceName = structured.CompileFieldPath("jsonPayload.involvedObject.name")
	pathReason       = structured.CompileFieldPath("jsonPayload.reason")
	pathMessage      = structured.CompileFieldPath("jsonPayload.message")
	pathAction       = structured.CompileFieldPath("jsonPayload.action")
)

// KubernetesEventFieldSet contains the parsed fields for a Kubernetes event log.
type KubernetesEventFieldSet struct {
	ProjectID    string
	ClusterName  string
	APIVersion   string
	ResourceKind string
	Namespace    string
	Resource     string
	Reason       string
	Message      string
}

// ExtractKubernetesEvent extracts KubernetesEventFieldSet from a NodeReader.
func ExtractKubernetesEvent(reader *structured.NodeReader) (KubernetesEventFieldSet, error) {
	if mock, ok := structured.GetMock[KubernetesEventFieldSet](reader); ok {
		return mock, nil
	}
	var result KubernetesEventFieldSet
	result.ProjectID = reader.ReadStringOrDefault(pathProjectID, "unknown")
	result.ClusterName = reader.ReadStringOrDefault(pathClusterName, "unknown")
	// Event exporter ingests cluster scoped logs without jsonPayload at the beginning
	if reader.Has(pathTextPayload) {
		result.Message = reader.ReadStringOrDefault(pathTextPayload, "")
		return result, nil
	}
	kind, err := reader.ReadString(pathKind)
	if err != nil {
		return KubernetesEventFieldSet{}, err
	}
	if kind != "Event" {
		return KubernetesEventFieldSet{}, fmt.Errorf("skipping unknown kind: %q: %w", kind, khierrors.ErrInvalidInput)
	}
	result.APIVersion = reader.ReadStringOrDefault(pathAPIVersion, "v1")
	if !strings.Contains(result.APIVersion, "/") {
		result.APIVersion = "core/" + result.APIVersion
	}
	result.ResourceKind = strings.ToLower(reader.ReadStringOrDefault(pathInvolvedKind, ""))
	result.Namespace = reader.ReadStringOrDefault(pathNamespace, "cluster-scope")
	result.Resource = reader.ReadStringOrDefault(pathResourceName, "")
	result.Reason = reader.ReadStringOrDefault(pathReason, "")
	result.Message = reader.ReadStringOrDefault(pathMessage, "")
	if result.Message == "" {
		result.Message = reader.ReadStringOrDefault(pathAction, "")
	}
	return result, nil
}
