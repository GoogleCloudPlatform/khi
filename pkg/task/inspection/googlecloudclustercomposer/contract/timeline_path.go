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

package googlecloudclustercomposer_contract

import (
	"context"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustComposerEnvironmentTimeline returns the timeline path for a Composer Environment.
func MustComposerEnvironmentTimeline(ctx context.Context, projectID, environmentName string) *khifilev6.TimelinePath {
	if projectID == "" {
		projectID = "unknown"
	}
	if environmentName == "" {
		environmentName = "unknown"
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	projectPath := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, projectID)
	return builder.TimelineAccumulator.GetPath(projectPath, khifilev6.PathSegment{
		Name: environmentName,
		Type: TimelineTypeComposerEnvironment,
	})
}

// MustAirflowDAGsRootTimeline returns the root timeline path for DAGs.
func MustAirflowDAGsRootTimeline(ctx context.Context, envPath *khifilev6.TimelinePath) *khifilev6.TimelinePath {
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(envPath, khifilev6.PathSegment{
		Name: "DAGs",
		Type: TimelineTypeDAGs,
	})
}

// MustAirflowDAGTimeline returns the timeline path for a specific Airflow DAG.
func MustAirflowDAGTimeline(ctx context.Context, envPath *khifilev6.TimelinePath, dagID string) *khifilev6.TimelinePath {
	if dagID == "" {
		dagID = "unknown"
	}
	dagsRoot := MustAirflowDAGsRootTimeline(ctx, envPath)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(dagsRoot, khifilev6.PathSegment{
		Name: dagID,
		Type: TimelineTypeAirflowDAG,
	})
}

// MustAirflowDAGRunTimeline returns the timeline path for a specific Airflow DAG run.
func MustAirflowDAGRunTimeline(ctx context.Context, envPath *khifilev6.TimelinePath, dagID, runID string) *khifilev6.TimelinePath {
	if dagID == "" {
		dagID = "unknown"
	}
	if runID == "" {
		runID = "unknown"
	}
	dagPath := MustAirflowDAGTimeline(ctx, envPath, dagID)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(dagPath, khifilev6.PathSegment{
		Name: runID,
		Type: TimelineTypeAirflowDAGRun,
	})
}

// MustAirflowTaskInstanceTimeline returns the timeline path for a specific Airflow TaskInstance under the given runPath.
func MustAirflowTaskInstanceTimeline(ctx context.Context, runPath *khifilev6.TimelinePath, taskName string) *khifilev6.TimelinePath {
	if taskName == "" {
		taskName = "unknown"
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(runPath, khifilev6.PathSegment{
		Name: taskName,
		Type: TimelineTypeAirflowTaskInstance,
	})
}

// MustAirflowComponentsRootTimeline returns the category root timeline path for Airflow components.
func MustAirflowComponentsRootTimeline(ctx context.Context, envPath *khifilev6.TimelinePath) *khifilev6.TimelinePath {
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(envPath, khifilev6.PathSegment{
		Name: "Components",
		Type: TimelineTypeComponents,
	})
}

// MustAirflowComponentTimeline returns the timeline path for a specific Airflow component.
func MustAirflowComponentTimeline(ctx context.Context, envPath *khifilev6.TimelinePath, componentType *pb.TimelineType, name string) *khifilev6.TimelinePath {
	if name == "" {
		name = "unknown"
	}
	compRoot := MustAirflowComponentsRootTimeline(ctx, envPath)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(compRoot, khifilev6.PathSegment{
		Name: name,
		Type: componentType,
	})
}

// MustAirflowWorkerTimeline returns the timeline path for an Airflow worker.
func MustAirflowWorkerTimeline(ctx context.Context, envPath *khifilev6.TimelinePath, workerHost string) *khifilev6.TimelinePath {
	if workerHost == "" {
		workerHost = "unknown"
	}
	return MustAirflowComponentTimeline(ctx, envPath, TimelineTypeAirflowWorker, workerHost)
}

// MustAirflowDAGProcessorManagerRootTimeline returns the root timeline path for DAG Processor Manager.
func MustAirflowDAGProcessorManagerRootTimeline(ctx context.Context, envPath *khifilev6.TimelinePath) *khifilev6.TimelinePath {
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(envPath, khifilev6.PathSegment{
		Name: "DAG Processor Manager",
		Type: TimelineTypeDAGProcessorManager,
	})
}

// MustAirflowDAGFileTimeline returns the timeline path for a parsed DAG file (with GCS prefix trimmed).
func MustAirflowDAGFileTimeline(ctx context.Context, envPath *khifilev6.TimelinePath, filePath string) *khifilev6.TimelinePath {
	if filePath == "" {
		filePath = "unknown"
	}
	dpmRoot := MustAirflowDAGProcessorManagerRootTimeline(ctx, envPath)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	trimmedPath := strings.TrimPrefix(filePath, "/home/airflow/gcs/dags/")
	if trimmedPath == "" {
		trimmedPath = "unknown"
	}
	return builder.TimelineAccumulator.GetPath(dpmRoot, khifilev6.PathSegment{
		Name: trimmedPath,
		Type: TimelineTypeDAGFile,
	})
}

// MustAirflowDAGProcessorManagerInstanceTimeline returns the timeline path for the manager instance that processed the file.
func MustAirflowDAGProcessorManagerInstanceTimeline(ctx context.Context, envPath *khifilev6.TimelinePath, filePath, instanceID string) *khifilev6.TimelinePath {
	if filePath == "" {
		filePath = "unknown"
	}
	if instanceID == "" {
		instanceID = "unknown"
	}
	fileRoot := MustAirflowDAGFileTimeline(ctx, envPath, filePath)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(fileRoot, khifilev6.PathSegment{
		Name: instanceID,
		Type: TimelineTypeDAGProcessorManagerInstance,
	})
}
