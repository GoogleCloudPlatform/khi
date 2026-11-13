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

package k8sevent

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
	pathResourceUID  = structured.CompileFieldPath("jsonPayload.involvedObject.uid")
	pathFieldPath    = structured.CompileFieldPath("jsonPayload.involvedObject.fieldPath")
	pathReason       = structured.CompileFieldPath("jsonPayload.reason")
	pathMessage      = structured.CompileFieldPath("jsonPayload.message")
	pathAction       = structured.CompileFieldPath("jsonPayload.action")
	pathSourceHost   = structured.CompileFieldPath("jsonPayload.source.host")
)

// KubernetesEventFieldSet contains the parsed fields for a Kubernetes event log.
type KubernetesEventFieldSet struct {
	// ProjectID is the GCP project ID from the resource labels.
	ProjectID string
	// ClusterName is the GKE cluster name from the resource labels.
	ClusterName string
	// APIVersion is the normalized API group/version of the involved Kubernetes object.
	APIVersion string
	// ResourceKind is the lowercase kind of the involved Kubernetes object.
	ResourceKind string
	// Namespace is the namespace of the involved object, or "cluster-scope" if non-namespaced.
	Namespace string
	// Resource is the name of the involved Kubernetes object.
	Resource string
	// ResourceUID is the UID of the involved Kubernetes object.
	ResourceUID string
	// FieldPath is the subobject path within the involved object (e.g., container field path).
	FieldPath string
	// Reason is the short machine-understandable reason for the event.
	Reason string
	// Message is the human-readable description of the event, falling back to action or textPayload.
	Message string
	// SourceHost is the hostname of the node reporting the event.
	SourceHost string
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
	result.ResourceUID = reader.ReadStringOrDefault(pathResourceUID, "")
	result.FieldPath = reader.ReadStringOrDefault(pathFieldPath, "")
	result.Reason = reader.ReadStringOrDefault(pathReason, "")
	result.Message = reader.ReadStringOrDefault(pathMessage, "")
	if result.Message == "" {
		result.Message = reader.ReadStringOrDefault(pathAction, "")
	}
	result.SourceHost = reader.ReadStringOrDefault(pathSourceHost, "")
	return result, nil
}
