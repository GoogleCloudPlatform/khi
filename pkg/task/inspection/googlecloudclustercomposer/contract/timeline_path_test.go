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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestMustComposerEnvironmentTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	testCases := []struct {
		name            string
		projectID       string
		environmentName string
		wantProject     string
		wantEnv         string
	}{
		{
			name:            "valid input",
			projectID:       "my-project",
			environmentName: "my-env",
			wantProject:     "my-project",
			wantEnv:         "my-env",
		},
		{
			name:            "empty project",
			projectID:       "",
			environmentName: "my-env",
			wantProject:     "unknown",
			wantEnv:         "my-env",
		},
		{
			name:            "empty env",
			projectID:       "my-project",
			environmentName: "",
			wantProject:     "my-project",
			wantEnv:         "unknown",
		},
		{
			name:            "both empty",
			projectID:       "",
			environmentName: "",
			wantProject:     "unknown",
			wantEnv:         "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MustComposerEnvironmentTimeline(ctx, tc.projectID, tc.environmentName)
			if got == nil {
				t.Fatal("expected timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantEnv, got.Name.Resolve()); diff != "" {
				t.Errorf("MustComposerEnvironmentTimeline() environmentName mismatch (-want +got):\n%s", diff)
			}
			if got.Parent == nil {
				t.Fatal("expected parent timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantProject, got.Parent.Name.Resolve()); diff != "" {
				t.Errorf("MustComposerEnvironmentTimeline() projectID mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMustAirflowDAGTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
	envPath := MustComposerEnvironmentTimeline(ctx, "my-project", "my-env")

	testCases := []struct {
		name    string
		dagID   string
		wantDAG string
	}{
		{
			name:    "valid input",
			dagID:   "my-dag",
			wantDAG: "my-dag",
		},
		{
			name:    "empty dag",
			dagID:   "",
			wantDAG: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MustAirflowDAGTimeline(ctx, envPath, tc.dagID)
			if got == nil {
				t.Fatal("expected timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantDAG, got.Name.Resolve()); diff != "" {
				t.Errorf("MustAirflowDAGTimeline() dagID mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMustAirflowDAGRunTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
	envPath := MustComposerEnvironmentTimeline(ctx, "my-project", "my-env")

	testCases := []struct {
		name    string
		dagID   string
		runID   string
		wantDAG string
		wantRun string
	}{
		{
			name:    "valid input",
			dagID:   "my-dag",
			runID:   "my-run",
			wantDAG: "my-dag",
			wantRun: "my-run",
		},
		{
			name:    "empty dag",
			dagID:   "",
			runID:   "my-run",
			wantDAG: "unknown",
			wantRun: "my-run",
		},
		{
			name:    "empty run",
			dagID:   "my-dag",
			runID:   "",
			wantDAG: "my-dag",
			wantRun: "unknown",
		},
		{
			name:    "both empty",
			dagID:   "",
			runID:   "",
			wantDAG: "unknown",
			wantRun: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MustAirflowDAGRunTimeline(ctx, envPath, tc.dagID, tc.runID)
			if got == nil {
				t.Fatal("expected timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantRun, got.Name.Resolve()); diff != "" {
				t.Errorf("MustAirflowDAGRunTimeline() runID mismatch (-want +got):\n%s", diff)
			}
			if got.Parent == nil {
				t.Fatal("expected parent timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantDAG, got.Parent.Name.Resolve()); diff != "" {
				t.Errorf("MustAirflowDAGRunTimeline() dagID mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMustAirflowTaskInstanceTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
	envPath := MustComposerEnvironmentTimeline(ctx, "my-project", "my-env")
	runPath := MustAirflowDAGRunTimeline(ctx, envPath, "my-dag", "my-run")

	testCases := []struct {
		name     string
		taskName string
		wantTask string
	}{
		{
			name:     "valid input",
			taskName: "my-task",
			wantTask: "my-task",
		},
		{
			name:     "empty task name",
			taskName: "",
			wantTask: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MustAirflowTaskInstanceTimeline(ctx, runPath, tc.taskName)
			if got == nil {
				t.Fatal("expected timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantTask, got.Name.Resolve()); diff != "" {
				t.Errorf("MustAirflowTaskInstanceTimeline() taskName mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMustAirflowComponentTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
	envPath := MustComposerEnvironmentTimeline(ctx, "my-project", "my-env")

	testCases := []struct {
		name          string
		componentName string
		wantName      string
	}{
		{
			name:          "valid input",
			componentName: "scheduler",
			wantName:      "scheduler",
		},
		{
			name:          "empty component name",
			componentName: "",
			wantName:      "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MustAirflowComponentTimeline(ctx, envPath, TimelineTypeAirflowScheduler, tc.componentName)
			if got == nil {
				t.Fatal("expected timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantName, got.Name.Resolve()); diff != "" {
				t.Errorf("MustAirflowComponentTimeline() componentName mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMustAirflowWorkerTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
	envPath := MustComposerEnvironmentTimeline(ctx, "my-project", "my-env")

	testCases := []struct {
		name       string
		workerHost string
		wantHost   string
	}{
		{
			name:       "valid input",
			workerHost: "worker-1",
			wantHost:   "worker-1",
		},
		{
			name:       "empty worker host",
			workerHost: "",
			wantHost:   "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MustAirflowWorkerTimeline(ctx, envPath, tc.workerHost)
			if got == nil {
				t.Fatal("expected timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantHost, got.Name.Resolve()); diff != "" {
				t.Errorf("MustAirflowWorkerTimeline() workerHost mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMustAirflowDAGFileTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
	envPath := MustComposerEnvironmentTimeline(ctx, "my-project", "my-env")

	testCases := []struct {
		name     string
		filePath string
		wantFile string
	}{
		{
			name:     "valid path with prefix",
			filePath: "/home/airflow/gcs/dags/my_dag.py",
			wantFile: "my_dag.py",
		},
		{
			name:     "valid path without prefix",
			filePath: "my_dag.py",
			wantFile: "my_dag.py",
		},
		{
			name:     "empty file path",
			filePath: "",
			wantFile: "unknown",
		},
		{
			name:     "file path is exactly the prefix",
			filePath: "/home/airflow/gcs/dags/",
			wantFile: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MustAirflowDAGFileTimeline(ctx, envPath, tc.filePath)
			if got == nil {
				t.Fatal("expected timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantFile, got.Name.Resolve()); diff != "" {
				t.Errorf("MustAirflowDAGFileTimeline() filePath mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMustAirflowDAGProcessorManagerInstanceTimeline(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
	envPath := MustComposerEnvironmentTimeline(ctx, "my-project", "my-env")

	testCases := []struct {
		name         string
		filePath     string
		instanceID   string
		wantFile     string
		wantInstance string
	}{
		{
			name:         "valid input",
			filePath:     "/home/airflow/gcs/dags/my_dag.py",
			instanceID:   "inst-123",
			wantFile:     "my_dag.py",
			wantInstance: "inst-123",
		},
		{
			name:         "empty file path",
			filePath:     "",
			instanceID:   "inst-123",
			wantFile:     "unknown",
			wantInstance: "inst-123",
		},
		{
			name:         "empty instance ID",
			filePath:     "/home/airflow/gcs/dags/my_dag.py",
			instanceID:   "",
			wantFile:     "my_dag.py",
			wantInstance: "unknown",
		},
		{
			name:         "both empty",
			filePath:     "",
			instanceID:   "",
			wantFile:     "unknown",
			wantInstance: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MustAirflowDAGProcessorManagerInstanceTimeline(ctx, envPath, tc.filePath, tc.instanceID)
			if got == nil {
				t.Fatal("expected timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantInstance, got.Name.Resolve()); diff != "" {
				t.Errorf("MustAirflowDAGProcessorManagerInstanceTimeline() instanceID mismatch (-want +got):\n%s", diff)
			}
			if got.Parent == nil {
				t.Fatal("expected parent timeline path to be not nil")
			}
			if diff := cmp.Diff(tc.wantFile, got.Parent.Name.Resolve()); diff != "" {
				t.Errorf("MustAirflowDAGProcessorManagerInstanceTimeline() filePath mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
