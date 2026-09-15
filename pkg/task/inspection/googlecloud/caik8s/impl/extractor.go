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
)

var (
	pathAssetName                             = structured.CompileFieldPath("asset.name")
	pathAssetType                             = structured.CompileFieldPath("asset.assetType")
	pathDeleted                               = structured.CompileFieldPath("deleted")
	pathWindowStartTime                       = structured.CompileFieldPath("window.startTime")
	pathWindowEndTime                         = structured.CompileFieldPath("window.endTime")
	pathResourceData                          = structured.CompileFieldPath("asset.resource.data")
	pathResourceDataAPIVersion                = structured.CompileFieldPath("asset.resource.data.apiVersion")
	pathResourceDataKind                      = structured.CompileFieldPath("asset.resource.data.kind")
	pathResourceDataMetadataName              = structured.CompileFieldPath("asset.resource.data.metadata.name")
	pathResourceDataMetadataNamespace         = structured.CompileFieldPath("asset.resource.data.metadata.namespace")
	pathResourceDataMetadataCreationTimestamp = structured.CompileFieldPath("asset.resource.data.metadata.creationTimestamp")
)

// extractResourceIdentityFromLog extracts the Kubernetes ResourceIdentity from the log's NodeReader.
func extractResourceIdentityFromLog(reader *structured.NodeReader) *k8saudit.ResourceIdentity {
	return resolveResourceIdentity(
		reader.ReadStringOrDefault(pathAssetName, ""),
		reader.ReadStringOrDefault(pathAssetType, ""),
		reader.ReadStringOrDefault(pathResourceDataAPIVersion, ""),
		reader.ReadStringOrDefault(pathResourceDataKind, ""),
		reader.ReadStringOrDefault(pathResourceDataMetadataName, ""),
		reader.ReadStringOrDefault(pathResourceDataMetadataNamespace, ""),
	)
}

// extractTimeWindow extracts validity start/end times and tombstone status from the log's NodeReader.
func extractTimeWindow(reader *structured.NodeReader) (startTime, endTime time.Time, isDeleted bool) {
	return reader.ReadTimestampOrDefault(pathWindowStartTime, time.Time{}),
		reader.ReadTimestampOrDefault(pathWindowEndTime, time.Time{}),
		reader.ReadBoolOrDefault(pathDeleted, false)
}

// extractCreationTimestamp extracts metadata.creationTimestamp of the manifest, reporting whether the
// manifest carried a parsable value.
func extractCreationTimestamp(reader *structured.NodeReader) (time.Time, bool) {
	creationTime := reader.ReadTimestampOrDefault(pathResourceDataMetadataCreationTimestamp, time.Time{})
	return creationTime, !creationTime.IsZero()
}

// isActiveAt reports whether the asset version covered by the given window was the current one at 'at'.
// A zero start time means the history did not report when the version became current, and a zero end
// time means the version is still current.
func isActiveAt(windowStartTime, windowEndTime, at time.Time) bool {
	if !windowStartTime.IsZero() && windowStartTime.After(at) {
		return false
	}
	if !windowEndTime.IsZero() && !windowEndTime.After(at) {
		return false
	}
	return true
}

// extractResourceBody extracts the Kubernetes resource manifest Node for timeline staging.
func extractResourceBody(reader *structured.NodeReader) structured.Node {
	node, err := reader.GetNode(pathResourceData)
	if err != nil {
		return nil
	}
	return structured.WithKeyOrder(node, k8s.K8sManifestKeyOrder...)
}
