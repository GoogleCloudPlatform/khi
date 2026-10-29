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

package csm_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csm"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// csmControlPlaneDefaultResourceNames returns default Cloud Logging resource names for querying CSM control plane logs.
func csmControlPlaneDefaultResourceNames(fleetProjectID string) []string {
	if fleetProjectID == "" {
		return []string{}
	}
	return []string{fmt.Sprintf("projects/%s", fleetProjectID)}
}

// generateCSMControlPlaneStructuredQuery generates the structured query for CSM control plane logs.
func generateCSMControlPlaneStructuredQuery(fleetProjectID string) *logestimator.StructuredLogQuery {
	return &logestimator.StructuredLogQuery{
		Incomplete:    fleetProjectID == "",
		ResourceTypes: []string{"istio_control_plane"},
		Filters: []logestimator.LoggingMonitoringMatcher{
			logestimator.ResourceLabel("project_id", logestimator.Exact(fleetProjectID)),
		},
	}
}

// csmControlPlaneQuerySource implements gcpcommon.StructuredLogQuerySource for CSM control plane logs.
type csmControlPlaneQuerySource struct {
	fleetProjectID coretask.Input[string]
}

// DefaultResourceNames returns the default resource names for querying CSM control plane logs.
func (s *csmControlPlaneQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return csmControlPlaneDefaultResourceNames(s.fleetProjectID.Get(ctx)), nil
}

// Queries returns the structured log queries for CSM control plane logs.
func (s *csmControlPlaneQuerySource) Queries(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*logestimator.StructuredLogQuery, error) {
	return []*logestimator.StructuredLogQuery{
		generateCSMControlPlaneStructuredQuery(s.fleetProjectID.Get(ctx)),
	}, nil
}

// TimePartitionCount returns the number of time partitions for parallel log querying.
func (s *csmControlPlaneQuerySource) TimePartitionCount() int {
	return 10
}

var _ gcpcommon.StructuredLogQuerySource = (*csmControlPlaneQuerySource)(nil)

// listCSMControlPlaneLogEntriesTask queries CSM control plane logs from Cloud Logging.
var listCSMControlPlaneLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(
	csm.ListCSMControlPlaneLogEntriesTaskID,
	"CSM Control Plane logs",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &csmControlPlaneQuerySource{
			fleetProjectID: coretask.Use(b, csm.InputFleetProjectIDTaskID.Ref()),
		}
	},
)
