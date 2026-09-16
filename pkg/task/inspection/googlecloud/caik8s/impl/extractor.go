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

package caik8s_impl

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

var (
	pathResourceDataAPIVersion                = structured.CompileFieldPath("asset.resource.data.apiVersion")
	pathResourceDataKind                      = structured.CompileFieldPath("asset.resource.data.kind")
	pathResourceDataMetadataName              = structured.CompileFieldPath("asset.resource.data.metadata.name")
	pathResourceDataMetadataNamespace         = structured.CompileFieldPath("asset.resource.data.metadata.namespace")
	pathResourceDataMetadataCreationTimestamp = structured.CompileFieldPath("asset.resource.data.metadata.creationTimestamp")
)

// extractK8sIdentity extracts the Kubernetes ResourceIdentity from the log's NodeReader
// and returns false if Kind or Name is empty.
func extractK8sIdentity(reader *structured.NodeReader) (*k8saudit.ResourceIdentity, bool) {
	identity := resolveResourceIdentity(
		gcpcommon.ExtractCAIAssetName(reader),
		gcpcommon.ExtractCAIAssetType(reader),
		reader.ReadStringOrDefault(pathResourceDataAPIVersion, ""),
		reader.ReadStringOrDefault(pathResourceDataKind, ""),
		reader.ReadStringOrDefault(pathResourceDataMetadataName, ""),
		reader.ReadStringOrDefault(pathResourceDataMetadataNamespace, ""),
	)
	if identity.Kind == "" || identity.Name == "" {
		return identity, false
	}
	return identity, true
}

// extractCreationTimestamp extracts metadata.creationTimestamp of the manifest.
func extractCreationTimestamp(reader *structured.NodeReader) time.Time {
	return reader.ReadTimestampOrDefault(pathResourceDataMetadataCreationTimestamp, time.Time{})
}

// extractK8sResourceBody extracts the Kubernetes resource manifest Node for timeline staging.
func extractK8sResourceBody(reader *structured.NodeReader) structured.Node {
	return gcpcommon.ExtractCAIResourceBody(reader, k8s.K8sManifestKeyOrder...)
}

// extractGKEResourceBody extracts the GKE Cluster or NodePool resource payload Node.
func extractGKEResourceBody(reader *structured.NodeReader) structured.Node {
	return gcpcommon.ExtractCAIResourceBody(reader)
}
