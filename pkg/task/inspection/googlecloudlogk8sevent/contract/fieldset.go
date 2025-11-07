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
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

type KubernetesEventFieldSet struct {
	ClusterName  string
	APIVersion   string
	ResourceKind string
	Namespace    string
	Resource     string
	Reason       string
	Message      string
}

func (k *KubernetesEventFieldSet) ResourcePath() resourcepath.ResourcePath {
	if k.Resource == "" {
		return resourcepath.Cluster(k.ClusterName)
	}
	return resourcepath.NameLayerGeneralItem(k.APIVersion, k.ResourceKind, k.Namespace, k.Resource)
}

// Kind implements log.FieldSet.
func (k *KubernetesEventFieldSet) Kind() string {
	return "k8s_event"
}

var _ log.FieldSet = (*KubernetesEventFieldSet)(nil)

type GCPKubernetesEventFieldSetReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (g *GCPKubernetesEventFieldSetReader) FieldSetKind() string {
	return (&KubernetesEventFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (g *GCPKubernetesEventFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result KubernetesEventFieldSet
	result.ClusterName = reader.ReadStringOrDefault("resource.labels.cluster_name", "unknown")
	// Event exporter ingests cluster scoped logs without jsonPayload at the beginning
	if reader.Has("textPayload") {
		result.Message = reader.ReadStringOrDefault("textPayload", "")
		return &result, nil
	}
	kind, err := reader.ReadString("jsonPayload.kind")
	if err != nil {
		return nil, err
	}
	if kind != "Event" {
		return nil, fmt.Errorf("skipping unknown kind: %q: %w", kind, khierrors.ErrInvalidInput)
	}
	result.APIVersion = reader.ReadStringOrDefault("jsonPayload.involvedObject.apiVersion", "v1")
	if !strings.Contains(result.APIVersion, "/") {
		result.APIVersion = "core/" + result.APIVersion
	}
	result.ResourceKind = strings.ToLower(reader.ReadStringOrDefault("jsonPayload.involvedObject.kind", ""))
	result.Namespace = reader.ReadStringOrDefault("jsonPayload.involvedObject.namespace", "cluster-scope")
	result.Resource = reader.ReadStringOrDefault("jsonPayload.involvedObject.name", "")
	result.Reason = reader.ReadStringOrDefault("jsonPayload.reason", "")
	result.Message = reader.ReadStringOrDefault("jsonPayload.message", "")
	if result.Message == "" {
		result.Message = reader.ReadStringOrDefault("jsonPayload.action", "")
	}
	return &result, nil

}
