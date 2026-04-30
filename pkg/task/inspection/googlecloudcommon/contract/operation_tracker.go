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

package googlecloudcommon_contract

import "sync"

// GCPOperationStateTracker tracks the state of long-running operations.
type GCPOperationStateTracker struct {
	pendingRequests sync.Map // map[string]string
}

// NewGCPOperationStateTracker creates a new GCPOperationStateTracker.
func NewGCPOperationStateTracker() *GCPOperationStateTracker {
	return &GCPOperationStateTracker{}
}

// TrackAndGetManifest tracks the operation and returns the manifest string if the resource state should be updated.
func (t *GCPOperationStateTracker) TrackAndGetManifest(audit *GCPAuditLogFieldSet) (manifest string, shouldUpdate bool) {
	if audit.ImmediateOperation() {
		if audit.Response != nil {
			manifest, _ = audit.ResponseString()
		} else if audit.Request != nil {
			manifest, _ = audit.RequestString()
		}
		return manifest, true
	}

	if audit.Starting() {
		if audit.Request != nil {
			req, _ := audit.RequestString()
			t.pendingRequests.Store(audit.OperationID, req)
		}
		return "", false
	}

	if audit.Ending() {
		if audit.Response != nil {
			manifest, _ = audit.ResponseString()
		}
		if manifest == "" {
			if v, ok := t.pendingRequests.Load(audit.OperationID); ok {
				manifest = v.(string)
			}
		}
		t.pendingRequests.Delete(audit.OperationID)
		return manifest, true
	}

	return "", false
}
