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

package csm_impl

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csm"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// generateCSMTrafficDirectorStructuredQuery generates a structured query for CSM Traffic Director logs.
func generateCSMTrafficDirectorStructuredQuery(fleetProjectID string, clusterIdentifiers []string, isDryRun bool) *logestimator.StructuredLogQuery {
	if isDryRun {
		clusterIdentifiers = []string{"dummy"}
	}
	if len(clusterIdentifiers) == 0 {
		return nil
	}

	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(fleetProjectID)),
		logestimator.LogID(logestimator.OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
	}

	switch {
	case isDryRun:
		filters = append(filters, logestimator.CustomFilter(`protoPayload.resourceName:"gsmrsvd-dummy" -- The actual resource name selector will be generated from other logs in the middle of the pipeline.`))
	case len(clusterIdentifiers) == 1:
		filters = append(filters, logestimator.CustomFilter(fmt.Sprintf(`protoPayload.resourceName:"gsmrsvd-%s"`, clusterIdentifiers[0])))
	default:
		quotedIdentifiers := make([]string, len(clusterIdentifiers))
		for i, id := range clusterIdentifiers {
			quotedIdentifiers[i] = fmt.Sprintf(`"gsmrsvd-%s"`, id)
		}
		filters = append(filters, logestimator.CustomFilter(fmt.Sprintf(`protoPayload.resourceName:(%s)`, strings.Join(quotedIdentifiers, " OR "))))
	}

	return &logestimator.StructuredLogQuery{
		Incomplete: fleetProjectID == "",
		Filters:    filters,
	}
}

// csmTrafficDirectorQuerySource builds the Cloud Logging query for the audit logs of the Traffic Director resources that CSM created for the cluster.
type csmTrafficDirectorQuerySource struct {
	fleetProjectID     coretask.Input[string]
	clusterIdentifiers coretask.Input[[]string]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *csmTrafficDirectorQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	fleetProjectID := s.fleetProjectID.Get(ctx)
	return []string{fmt.Sprintf("projects/%s", fleetProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *csmTrafficDirectorQuerySource) Queries(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*logestimator.StructuredLogQuery, error) {
	fleetProjectID := s.fleetProjectID.Get(ctx)
	clusterIdentifiers := s.clusterIdentifiers.Get(ctx)
	isDryRun := taskMode == inspectioncore.TaskModeDryRun

	sq := generateCSMTrafficDirectorStructuredQuery(fleetProjectID, clusterIdentifiers, isDryRun)
	if sq == nil {
		if !isDryRun {
			slog.InfoContext(ctx, "No CSM BackendServices found in inventory. Skipping Traffic Director log query.")
		}
		return nil, nil
	}

	return []*logestimator.StructuredLogQuery{sq}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *csmTrafficDirectorQuerySource) TimePartitionCount() int {
	return 1
}

var _ gcpcommon.StructuredLogQuerySource = (*csmTrafficDirectorQuerySource)(nil)

// listCSMTrafficDirectorLogEntriesTask queries the audit logs of the Traffic Director resources that CSM created for the cluster from Cloud Logging.
var listCSMTrafficDirectorLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(
	csm.ListCSMTrafficDirectorLogEntriesTaskID,
	"CSM Traffic Director logs",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &csmTrafficDirectorQuerySource{
			fleetProjectID:     coretask.Use(b, csm.InputFleetProjectIDTaskID.Ref()),
			clusterIdentifiers: coretask.Use(b, csm.CSMClusterIdentifierTaskID.Ref()),
		}
	},
)
