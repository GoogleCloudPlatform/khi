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

package googlecloudcaik8s_contract

import (
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
)

// ClusterResourceSnapshot represents a Kubernetes resource captured from CAI.
type ClusterResourceSnapshot struct {
	// TemporalAsset holds the raw temporal asset response received from Cloud Asset Inventory.
	// This payload is used directly as the log body.
	TemporalAsset *assetpb.TemporalAsset

	// StartTime is the beginning of this temporal snapshot's validity window.
	StartTime time.Time
}
