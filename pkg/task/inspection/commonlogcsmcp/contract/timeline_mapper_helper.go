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

package commonlogcsmcp_contract

import (
	"context"
	"time"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// TimelineState tracks connection states across logs in a group.
type TimelineState struct {
	ObservedConnections map[string]bool
}

// MapPodAndConnectionTimelines maps xDS logs to client Pod and Connection timelines and tracks connection states.
func MapPodAndConnectionTimelines(
	ctx context.Context,
	cs *khifilev6.TimelineChangeSet,
	clusterName string,
	msg string,
	changedTime time.Time,
	pods []PodIdentifier,
	state *TimelineState,
) *TimelineState {
	if !IsXDSLog(msg) || clusterName == "" || len(pods) == 0 {
		return state
	}

	clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
	kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiPath, "pod")

	for _, pod := range pods {
		clientNsPath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, pod.Namespace)
		clientPodPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, clientNsPath, pod.Name)
		csmcpPodLogPath := MustCSMCPPodLogTimeline(ctx, clientPodPath)

		cs.AddEvent(csmcpPodLogPath)

		if pod.ConnectionID == "" || (!IsConnectionLog(msg) && !IsDisconnectionLog(msg)) {
			continue
		}

		if state == nil {
			state = &TimelineState{
				ObservedConnections: make(map[string]bool),
			}
		}

		connPath := MustCSMCPConnectionTimeline(ctx, clientPodPath, pod.ConnectionID)

		if IsConnectionLog(msg) {
			state.ObservedConnections[pod.ConnectionKey()] = true
			cs.AddRevision(connPath, &khifilev6.StagingRevision{
				VerbType:     inspectioncore.VerbUnknown,
				StateType:    RevisionStateCSMCPConnectionConnected,
				ResourceBody: nil,
				Principal:    "csm-cp",
				ChangedTime:  changedTime,
			})
			continue
		}

		if !state.ObservedConnections[pod.ConnectionKey()] {
			cs.AddRevision(connPath, &khifilev6.StagingRevision{
				VerbType:     inspectioncore.VerbUnknown,
				StateType:    RevisionStateCSMCPConnectionConnectedLogNotFound,
				ResourceBody: nil,
				Principal:    "csm-cp",
				ChangedTime:  time.Unix(0, 0),
			})
			state.ObservedConnections[pod.ConnectionKey()] = true
		}
		cs.AddRevision(connPath, &khifilev6.StagingRevision{
			VerbType:     inspectioncore.VerbUnknown,
			StateType:    RevisionStateCSMCPConnectionTerminated,
			ResourceBody: nil,
			Principal:    "csm-cp",
			ChangedTime:  changedTime,
		})
	}
	return state
}
