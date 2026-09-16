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

package gcpcommon

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

var (
	caiPathAssetName       = structured.CompileFieldPath("asset.name")
	caiPathAssetType       = structured.CompileFieldPath("asset.assetType")
	caiPathDeleted         = structured.CompileFieldPath("deleted")
	caiPathWindowStartTime = structured.CompileFieldPath("window.startTime")
	caiPathWindowEndTime   = structured.CompileFieldPath("window.endTime")
	caiPathResourceData    = structured.CompileFieldPath("asset.resource.data")
)

// ExtractCAIAssetName extracts the full asset name from a CAI snapshot log's NodeReader.
func ExtractCAIAssetName(reader *structured.NodeReader) string {
	return reader.ReadStringOrDefault(caiPathAssetName, "")
}

// ExtractCAIAssetType extracts the asset type string from a CAI snapshot log's NodeReader.
func ExtractCAIAssetType(reader *structured.NodeReader) string {
	return reader.ReadStringOrDefault(caiPathAssetType, "")
}

// ExtractCAITimeWindow extracts validity start/end times and tombstone status from a CAI snapshot log's NodeReader.
func ExtractCAITimeWindow(reader *structured.NodeReader) (startTime, endTime time.Time, isDeleted bool) {
	return reader.ReadTimestampOrDefault(caiPathWindowStartTime, time.Time{}),
		reader.ReadTimestampOrDefault(caiPathWindowEndTime, time.Time{}),
		reader.ReadBoolOrDefault(caiPathDeleted, false)
}

// IsCAIAssetActiveAt reports whether the asset version covered by the given window was the current one at 'at'.
// A zero start time means the history did not report when the version became current, and a zero end
// time means the version is still current.
func IsCAIAssetActiveAt(windowStartTime, windowEndTime, at time.Time) bool {
	if !windowStartTime.IsZero() && windowStartTime.After(at) {
		return false
	}
	if !windowEndTime.IsZero() && !windowEndTime.After(at) {
		return false
	}
	return true
}

// ExtractCAIResourceBody extracts the resource payload node (`asset.resource.data`) from a CAI snapshot log's NodeReader,
// optionally reordering keys if keyOrder is provided.
func ExtractCAIResourceBody(reader *structured.NodeReader, keyOrder ...string) structured.Node {
	node, err := reader.GetNode(caiPathResourceData)
	if err != nil {
		return nil
	}
	if len(keyOrder) > 0 {
		return structured.WithKeyOrder(node, keyOrder...)
	}
	return node
}
