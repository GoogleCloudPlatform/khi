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

package googlecloudcaik8s_impl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"google.golang.org/protobuf/encoding/protojson"
)

// gkeResourceIdentity holds identifying details extracted from a GKE asset name.
type gkeResourceIdentity struct {
	ClusterName  string
	NodePoolName string
}

// IsCluster returns true if the asset identity corresponds to a cluster.
func (i gkeResourceIdentity) IsCluster() bool {
	return i.ClusterName != "" && i.NodePoolName == ""
}

// IsNodePool returns true if the asset identity corresponds to a node pool.
func (i gkeResourceIdentity) IsNodePool() bool {
	return i.NodePoolName != ""
}

// parseGKEAssetName parses a GKE full resource name into a gkeResourceIdentity.
func parseGKEAssetName(name string) gkeResourceIdentity {
	parts := strings.Split(name, "/")
	var clusterName, nodePoolName string
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "clusters" {
			clusterName = parts[i+1]
		} else if parts[i] == "nodePools" {
			nodePoolName = parts[i+1]
		}
	}
	return gkeResourceIdentity{
		ClusterName:  clusterName,
		NodePoolName: nodePoolName,
	}
}

// snapshotToGKERawLog converts a GKEResourceSnapshot to a raw Log entity.
func snapshotToGKERawLog(idGen *id.Generator, s *googlecloudcaik8s_contract.GKEResourceSnapshot) (*log.Log, error) {
	jsonBytes, err := protojson.Marshal(s.TemporalAsset)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal temporal asset to JSON: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal temporal asset JSON: %w", err)
	}
	node, nodeErr := structured.FromGoValue(m, &structured.AlphabeticalGoMapKeyOrderProvider{})
	if nodeErr != nil {
		return nil, fmt.Errorf("failed to convert temporal asset map to structured node: %w", nodeErr)
	}

	reader := structured.NewNodeReader(node)
	return log.NewLogWithTimestamp(idGen, reader, s.StartTime), nil
}

// GKERawLogTask converts fetched GKE resource snapshots into raw logs.
var GKERawLogTask = inspectiontaskbase.NewInspectionTask(
	googlecloudcaik8s_contract.GKERawLogTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudcaik8s_contract.GKEResourceFetcherTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]*log.Log, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return []*log.Log{}, nil
		}
		snapshots := coretask.GetTaskResult(ctx, googlecloudcaik8s_contract.GKEResourceFetcherTaskID.Ref())
		idGen := khictx.MustGetValue(ctx, inspectioncore_contract.IDGenerator)

		logs := make([]*log.Log, 0, len(snapshots))
		for _, s := range snapshots {
			l, err := snapshotToGKERawLog(idGen, s)
			if err != nil {
				return nil, err
			}
			logs = append(logs, l)
		}
		return logs, nil
	},
)

// caiGKEResourceLogIngester implements LogIngester for CAI GKE resource snapshots.
type caiGKEResourceLogIngester struct{}

var _ inspectiontaskbase.LogIngester = (*caiGKEResourceLogIngester)(nil)

// RawLogTask returns the task reference providing raw CAI GKE logs.
func (i *caiGKEResourceLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudcaik8s_contract.GKERawLogTaskID.Ref()
}

// Dependencies returns additional task dependencies for log ingestion.
func (i *caiGKEResourceLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog populates the metadata into LogChangeSet.
func (i *caiGKEResourceLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	assetName := l.NodeReader.ReadStringOrDefault(pathAssetName, "")
	identity := parseGKEAssetName(assetName)

	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetTimestamp(l.Timestamp)
	cs.SetLogType(googlecloudcaik8s_contract.LogTypeCAIResourceSnapshot)
	cs.SetSeverity(inspectioncore_contract.SeverityInfo)

	switch {
	case identity.IsNodePool():
		cs.SetSummary(fmt.Sprintf("CAI resource snapshot: NodePool/%s", identity.NodePoolName))
	case identity.IsCluster():
		cs.SetSummary(fmt.Sprintf("CAI resource snapshot: Cluster/%s", identity.ClusterName))
	default:
		cs.SetSummary("CAI resource snapshot: GKE resource")
	}

	return cs, nil
}

// GKELogIngesterTask is the task that ingests CAI GKE resource snapshot log metadata.
var GKELogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	googlecloudcaik8s_contract.GKELogIngesterTaskID,
	&caiGKEResourceLogIngester{},
)

// GKELogGrouperTask groups CAI GKE resource snapshot logs by their resource identity.
var GKELogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudcaik8s_contract.GKELogGrouperTaskID,
	googlecloudcaik8s_contract.GKERawLogTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		assetName := l.NodeReader.ReadStringOrDefault(pathAssetName, "")
		identity := parseGKEAssetName(assetName)
		if identity.IsNodePool() {
			return fmt.Sprintf("nodepool/%s/%s", identity.ClusterName, identity.NodePoolName)
		} else if identity.IsCluster() {
			return fmt.Sprintf("cluster/%s", identity.ClusterName)
		}
		return "unknown"
	},
)
