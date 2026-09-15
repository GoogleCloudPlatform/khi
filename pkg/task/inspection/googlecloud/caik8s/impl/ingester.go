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
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"google.golang.org/protobuf/encoding/protojson"
)

// snapshotToRawLog converts a ClusterResourceSnapshot to a raw Log entity.
func snapshotToRawLog(idGen *id.Generator, s *caik8s.ClusterResourceSnapshot) (*log.Log, error) {
	jsonBytes, err := protojson.Marshal(s.TemporalAsset)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal temporal asset to JSON: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal temporal asset JSON: %w", err)
	}
	restoreManifestTypeMeta(m)
	restoreManifestMetadata(m)
	node, nodeErr := structured.FromGoValue(m, &structured.AlphabeticalGoMapKeyOrderProvider{})
	if nodeErr != nil {
		return nil, fmt.Errorf("failed to convert temporal asset map to structured node: %w", nodeErr)
	}

	reader := structured.NewNodeReader(node)
	return log.NewLogWithTimestamp(idGen, reader, s.StartTime), nil
}

// RawLogTask converts fetched cluster resource snapshots into raw logs.
var RawLogTask = inspectiontaskbase.NewInspectionTask(
	caik8s.RawLogTaskID,
	[]coretask.Dependency{
		caik8s.ClusterResourceFetcherTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*log.Log, error) {
		if taskMode == inspectioncore.TaskModeDryRun {
			return []*log.Log{}, nil
		}
		snapshots := coretask.GetTaskResult(ctx, caik8s.ClusterResourceFetcherTaskID.Ref())
		idGen := khictx.MustGetValue(ctx, inspectioncore.IDGenerator)

		logs := make([]*log.Log, 0, len(snapshots))
		for _, s := range snapshots {
			l, err := snapshotToRawLog(idGen, s)
			if err != nil {
				return nil, err
			}
			logs = append(logs, l)
		}
		return logs, nil
	},
)

// caiClusterResourceLogIngester implements LogIngester for CAI resource snapshots.
type caiClusterResourceLogIngester struct{}

var _ inspectiontaskbase.LogIngester = (*caiClusterResourceLogIngester)(nil)

// RawLogTask returns the task reference providing raw CAI logs.
func (i *caiClusterResourceLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return caik8s.RawLogTaskID.Ref()
}

// Dependencies returns additional task dependencies for log ingestion.
func (i *caiClusterResourceLogIngester) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{}
}

// ProcessLog populates the metadata into LogChangeSet.
func (i *caiClusterResourceLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	identity := extractResourceIdentityFromLog(l.NodeReader)

	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetTimestamp(l.Timestamp)
	cs.SetLogType(caik8s.LogTypeCAIResourceSnapshot)
	cs.SetSeverity(inspectioncore.SeverityInfo)
	cs.SetSummary(fmt.Sprintf("CAI resource snapshot: %s/%s", identity.Kind, identity.Name))

	return cs, nil
}

// LogIngesterTask is the task that ingests CAI cluster resource snapshot log metadata.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	caik8s.LogIngesterTaskID,
	&caiClusterResourceLogIngester{},
)

// LogGrouperTask groups CAI resource snapshot logs by their resource identity.
// The grouper cannot report errors, so logs without a recognized identity fall into a single bucket.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	caik8s.LogGrouperTaskID,
	caik8s.RawLogTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		identity := extractResourceIdentityFromLog(l.NodeReader)
		if identity.Kind == "" || identity.Name == "" {
			return "unknown"
		}
		return identity.String()
	},
)
