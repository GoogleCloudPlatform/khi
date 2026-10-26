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

package inspectionserver

import (
	"context"
	"testing"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// TypeID is the inspection type ID registered by NewServerWithInspection.
const TypeID = "test-type"

// NewServer creates an InspectionTaskServer whose data and temporary directories live under t.TempDir().
func NewServer(t testing.TB) *coreinspection.InspectionTaskServer {
	t.Helper()
	ioConfig, err := inspectioncore.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("failed to create test IOConfig: %v", err)
	}
	tempDir := t.TempDir()
	ioConfig.DataDestination = tempDir
	ioConfig.TemporaryFolder = tempDir
	server, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("failed to create inspection server: %v", err)
	}
	return server
}

// FeatureTask returns a constant task enabled by default as an inspection feature.
func FeatureTask() coretask.UntypedTask {
	dummyTaskID := taskid.NewDefaultImplementationID[any]("dummy-task")
	return coretask.DefineConstant[any](
		dummyTaskID,
		"success",
		coretask.WithLabelValue(inspectioncore.LabelKeyInspectionDefaultFeatureFlag, true),
		coretask.WithLabelValue(inspectioncore.LabelKeyInspectionFeatureFlag, true),
	)
}

// NewServerWithInspection registers TypeID with the given feature tasks and returns the server along with the ID of one completed inspection.
func NewServerWithInspection(t testing.TB, tasks ...coretask.UntypedTask) (*coreinspection.InspectionTaskServer, string) {
	t.Helper()
	server := NewServer(t)
	if err := server.AddInspectionType(coreinspection.InspectionType{
		Id:   TypeID,
		Name: "Test Type",
	}); err != nil {
		t.Fatalf("failed to add inspection type: %v", err)
	}
	for _, task := range tasks {
		if err := server.AddTask(task); err != nil {
			t.Fatalf("failed to add task: %v", err)
		}
	}
	return server, RunInspection(t, server)
}

// RunInspection creates one more inspection of TypeID on the server, runs it, and waits until it finishes.
func RunInspection(t testing.TB, server *coreinspection.InspectionTaskServer) string {
	t.Helper()
	inspectionID, err := server.CreateInspection(TypeID)
	if err != nil {
		t.Fatalf("failed to create inspection: %v", err)
	}
	runner := server.GetInspection(inspectionID)
	if err := runner.Run(context.Background(), &inspectioncore.InspectionRequest{Values: map[string]any{}}); err != nil {
		t.Fatalf("failed to run inspection: %v", err)
	}
	<-runner.Wait()
	return inspectionID
}
