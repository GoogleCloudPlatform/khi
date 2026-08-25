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

package apiv1

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/generated"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func setupTestInspectionServer(
	t *testing.T,
	cycleDuration time.Duration,
	updateInterval time.Duration,
) (*httptest.Server, apiv1connect.InspectionServiceClient, *coreinspection.InspectionTaskServer) {
	t.Helper()
	logger.InitGlobalKHILogger()
	oldStore := upload.DefaultUploadFileStore
	upload.DefaultUploadFileStore = upload.NewUploadFileStore(upload.NewLocalUploadFileStoreProvider(t.TempDir()))
	t.Cleanup(func() {
		upload.DefaultUploadFileStore = oldStore
	})
	ioConfig, err := inspectioncore_contract.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("NewIOConfigForTest failed: %v", err)
	}
	server, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	err = generated.RegisterAllInspectionTasks(server)
	if err != nil {
		t.Fatalf("RegisterAllInspectionTasks failed: %v", err)
	}
	style.LockRegistry()

	serverImpl := NewInspectionServiceServerWithIntervals(server, cycleDuration, updateInterval)
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewInspectionServiceHandler(serverImpl)
	mux.Handle(path, handler)

	ts := httptest.NewServer(mux)
	client := apiv1connect.NewInspectionServiceClient(ts.Client(), ts.URL)
	return ts, client, server
}

func TestInspectionServiceServer_GetInspectionTypes(t *testing.T) {
	testCases := []struct {
		name         string
		targetTypeId string
		wantFound    bool
	}{
		{
			name:         "registers and returns GKE inspection type",
			targetTypeId: "gcp-gke",
			wantFound:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, _ := setupTestInspectionServer(t, 30*time.Second, 1*time.Second)
			defer ts.Close()

			res, err := client.GetInspectionTypes(context.Background(), connect.NewRequest(&apiv1.GetInspectionTypesRequest{}))
			if err != nil {
				t.Fatalf("GetInspectionTypes() unexpected error: %v", err)
			}

			found := false
			for _, typ := range res.Msg.GetTypes() {
				if typ.GetId() == tc.targetTypeId {
					found = true
					break
				}
			}

			if found != tc.wantFound {
				t.Errorf("inspection type %s found = %v, want %v", tc.targetTypeId, found, tc.wantFound)
			}
		})
	}
}

func TestInspectionServiceServer_CreateAndUpdateInspection(t *testing.T) {
	testCases := []struct {
		name         string
		typeId       string
		updatedName  string
		wantName     string
		wantFilename string
	}{
		{
			name:         "creates inspection and updates name",
			typeId:       "gcp-gke",
			updatedName:  "My Custom Inspection",
			wantName:     "My Custom Inspection",
			wantFilename: "My Custom Inspection.khi",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, server := setupTestInspectionServer(t, 30*time.Second, 1*time.Second)
			defer ts.Close()

			createRes, err := client.CreateInspection(context.Background(), connect.NewRequest(&apiv1.CreateInspectionRequest{
				InspectionTypeId: proto.String(tc.typeId),
			}))
			if err != nil {
				t.Fatalf("CreateInspection() unexpected error: %v", err)
			}
			if createRes.Msg.GetInspectionId() == "" {
				t.Fatal("CreateInspection() returned empty inspection ID")
			}

			filePath := filepath.Join(t.TempDir(), "result.khi")
			if err := os.WriteFile(filePath, []byte("test data"), 0644); err != nil {
				t.Fatalf("WriteFile failed: %v", err)
			}
			store := inspectioncore_contract.NewFileSystemInspectionResultRepository(filePath)
			metadata := typedmap.NewTypedMap()
			header := &inspectionmetadata.HeaderMetadata{
				InspectionType: tc.typeId,
				InspectionName: "Initial Name",
			}
			typedmap.Set(metadata, inspectionmetadata.HeaderMetadataKey, header)
			server.RegisterImportedInspection("imported-update-1", store, metadata.AsReadonly())

			_, err = client.UpdateInspection(context.Background(), connect.NewRequest(&apiv1.UpdateInspectionRequest{
				InspectionId: proto.String("imported-update-1"),
				Name:         proto.String(tc.updatedName),
			}))
			if err != nil {
				t.Fatalf("UpdateInspection() unexpected error: %v", err)
			}

			runner := server.GetInspection("imported-update-1")
			if runner == nil {
				t.Fatalf("GetInspection(imported-update-1) returned nil")
			}
			md, err := runner.GetCurrentMetadata()
			if err != nil {
				t.Fatalf("GetCurrentMetadata() failed: %v", err)
			}
			gotHeader, found := typedmap.Get(md, inspectionmetadata.HeaderMetadataKey)
			if !found || gotHeader == nil {
				t.Fatal("HeaderMetadata not found")
			}

			if gotHeader.InspectionName != tc.wantName {
				t.Errorf("InspectionName = %q, want %q", gotHeader.InspectionName, tc.wantName)
			}
			if gotHeader.SuggestedFileName != tc.wantFilename {
				t.Errorf("SuggestedFileName = %q, want %q", gotHeader.SuggestedFileName, tc.wantFilename)
			}
		})
	}
}

func TestInspectionServiceServer_InspectionFeatures(t *testing.T) {
	testCases := []struct {
		name         string
		typeId       string
		featureState bool
	}{
		{
			name:         "retrieves features and toggles state",
			typeId:       "gcp-gke",
			featureState: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, _ := setupTestInspectionServer(t, 30*time.Second, 1*time.Second)
			defer ts.Close()

			createRes, err := client.CreateInspection(context.Background(), connect.NewRequest(&apiv1.CreateInspectionRequest{
				InspectionTypeId: proto.String(tc.typeId),
			}))
			if err != nil {
				t.Fatalf("CreateInspection() unexpected error: %v", err)
			}
			inspectionID := createRes.Msg.GetInspectionId()

			featRes, err := client.GetInspectionFeatures(context.Background(), connect.NewRequest(&apiv1.GetInspectionFeaturesRequest{
				InspectionId: proto.String(inspectionID),
			}))
			if err != nil {
				t.Fatalf("GetInspectionFeatures() unexpected error: %v", err)
			}
			features := featRes.Msg.GetFeatures()
			if len(features) == 0 {
				t.Fatal("GetInspectionFeatures() returned no features")
			}
			targetFeatureID := features[0].GetId()

			_, err = client.UpdateInspectionFeatures(context.Background(), connect.NewRequest(&apiv1.UpdateInspectionFeaturesRequest{
				InspectionId: proto.String(inspectionID),
				FeatureStates: map[string]bool{
					targetFeatureID: tc.featureState,
				},
			}))
			if err != nil {
				t.Fatalf("UpdateInspectionFeatures() unexpected error: %v", err)
			}

			featResAfter, err := client.GetInspectionFeatures(context.Background(), connect.NewRequest(&apiv1.GetInspectionFeaturesRequest{
				InspectionId: proto.String(inspectionID),
			}))
			if err != nil {
				t.Fatalf("GetInspectionFeatures() after update unexpected error: %v", err)
			}

			var found *apiv1.InspectionFeature
			for _, f := range featResAfter.Msg.GetFeatures() {
				if f.GetId() == targetFeatureID {
					found = f
					break
				}
			}
			if found == nil {
				t.Fatalf("feature %s not found in features list", targetFeatureID)
			}
			if found.GetEnabled() != tc.featureState {
				t.Errorf("feature enabled = %v, want %v", found.GetEnabled(), tc.featureState)
			}
		})
	}
}

func TestInspectionServiceServer_GetAndWatchInspections(t *testing.T) {
	testCases := []struct {
		name          string
		cycleDuration time.Duration
		interval      time.Duration
	}{
		{
			name:          "streams initial snapshot then expires cycle",
			cycleDuration: 50 * time.Millisecond,
			interval:      10 * time.Millisecond,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, server := setupTestInspectionServer(t, tc.cycleDuration, tc.interval)
			defer ts.Close()

			filePath := filepath.Join(t.TempDir(), "result.khi")
			if err := os.WriteFile(filePath, []byte("test data"), 0644); err != nil {
				t.Fatalf("WriteFile failed: %v", err)
			}
			store := inspectioncore_contract.NewFileSystemInspectionResultRepository(filePath)
			metadata := typedmap.NewTypedMap()
			header := &inspectionmetadata.HeaderMetadata{
				InspectionType: "gcp-gke",
				InspectionName: "Imported Run",
			}
			typedmap.Set(metadata, inspectionmetadata.HeaderMetadataKey, header)
			server.RegisterImportedInspection("imported-1", store, metadata.AsReadonly())

			// Test GetInspections (snapshot)
			getRes, err := client.GetInspections(context.Background(), connect.NewRequest(&apiv1.GetInspectionsRequest{}))
			if err != nil {
				t.Fatalf("GetInspections() unexpected error: %v", err)
			}
			if len(getRes.Msg.GetInspections()) != 1 {
				t.Fatalf("GetInspections() count = %d, want 1", len(getRes.Msg.GetInspections()))
			}
			if getRes.Msg.GetInspections()[0].GetId() != "imported-1" {
				t.Errorf("inspection ID = %q, want %q", getRes.Msg.GetInspections()[0].GetId(), "imported-1")
			}

			// Test WatchInspections (stream)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			stream, err := client.WatchInspections(ctx, connect.NewRequest(&apiv1.WatchInspectionsRequest{}))
			if err != nil {
				t.Fatalf("WatchInspections() unexpected error: %v", err)
			}

			receivedCount := 0
			var firstItem *apiv1.InspectionListItem
			for stream.Receive() {
				receivedCount++
				if firstItem == nil && len(stream.Msg().GetInspections()) > 0 {
					firstItem = stream.Msg().GetInspections()[0]
				}
			}

			if receivedCount < 1 {
				t.Errorf("received count = %d, want at least 1", receivedCount)
			}
			if firstItem == nil || firstItem.GetId() != "imported-1" {
				t.Errorf("stream item = %v, want ID imported-1", firstItem)
			}
			if stream.Err() != nil {
				t.Errorf("stream.Err() = %v, want nil on cycle completion", stream.Err())
			}
		})
	}
}

func TestInspectionServiceServer_GetInspectionDataChunk(t *testing.T) {
	testCases := []struct {
		name          string
		data          []byte
		offset        int64
		maxSize       int64
		wantData      []byte
		wantTotalSize int64
		wantErrCode   connect.Code
	}{
		{
			name:          "reads full data chunk",
			data:          []byte("hello world 1234567890"),
			offset:        0,
			maxSize:       1024,
			wantData:      []byte("hello world 1234567890"),
			wantTotalSize: 22,
		},
		{
			name:          "reads partial data chunk with offset",
			data:          []byte("hello world 1234567890"),
			offset:        6,
			maxSize:       5,
			wantData:      []byte("world"),
			wantTotalSize: 22,
		},
		{
			name:        "returns invalid argument error for negative offset",
			data:        []byte("hello world"),
			offset:      -1,
			maxSize:     5,
			wantErrCode: connect.CodeInvalidArgument,
		},
		{
			name:        "returns invalid argument error when offset exceeds file size",
			data:        []byte("hello world"),
			offset:      100,
			maxSize:     5,
			wantErrCode: connect.CodeInvalidArgument,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, server := setupTestInspectionServer(t, 30*time.Second, 1*time.Second)
			defer ts.Close()

			filePath := filepath.Join(t.TempDir(), "result.khi")
			if err := os.WriteFile(filePath, tc.data, 0644); err != nil {
				t.Fatalf("WriteFile failed: %v", err)
			}
			store := inspectioncore_contract.NewFileSystemInspectionResultRepository(filePath)
			metadata := typedmap.NewTypedMap()
			server.RegisterImportedInspection("data-test-1", store, metadata.AsReadonly())

			res, err := client.GetInspectionDataChunk(context.Background(), connect.NewRequest(&apiv1.GetInspectionDataChunkRequest{
				InspectionId: proto.String("data-test-1"),
				OffsetBytes:  proto.Int64(tc.offset),
				MaxSizeBytes: proto.Int64(tc.maxSize),
			}))
			if tc.wantErrCode != 0 {
				if err == nil {
					t.Fatalf("GetInspectionDataChunk() expected error, got nil")
				}
				if connect.CodeOf(err) != tc.wantErrCode {
					t.Errorf("GetInspectionDataChunk() code = %v, want %v", connect.CodeOf(err), tc.wantErrCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetInspectionDataChunk() unexpected error: %v", err)
			}

			if res.Msg.GetTotalFileSizeBytes() != tc.wantTotalSize {
				t.Errorf("TotalFileSizeBytes = %d, want %d", res.Msg.GetTotalFileSizeBytes(), tc.wantTotalSize)
			}
			if !bytes.Equal(res.Msg.GetData(), tc.wantData) {
				t.Errorf("Data = %q, want %q", string(res.Msg.GetData()), string(tc.wantData))
			}
		})
	}
}

func TestInspectionServiceServer_DryRunInspection(t *testing.T) {
	testCases := []struct {
		name   string
		typeId string
	}{
		{
			name:   "performs dry run and returns form fields and plan",
			typeId: "gcp-gke",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, _ := setupTestInspectionServer(t, 30*time.Second, 1*time.Second)
			defer ts.Close()

			createRes, err := client.CreateInspection(context.Background(), connect.NewRequest(&apiv1.CreateInspectionRequest{
				InspectionTypeId: proto.String(tc.typeId),
			}))
			if err != nil {
				t.Fatalf("CreateInspection() unexpected error: %v", err)
			}
			inspectionID := createRes.Msg.GetInspectionId()

			dryRunRes, err := client.DryRunInspection(context.Background(), connect.NewRequest(&apiv1.DryRunInspectionRequest{
				InspectionId: proto.String(inspectionID),
				Parameters:   &apiv1.InspectionParameters{},
			}))
			if err != nil {
				t.Fatalf("DryRunInspection() unexpected error: %v", err)
			}

			if dryRunRes.Msg.GetPlan() == nil || dryRunRes.Msg.GetPlan().GetTaskGraph() == "" {
				t.Errorf("DryRunInspection() plan task graph is empty: %v", dryRunRes.Msg.GetPlan())
			}
		})
	}
}

func TestInspectionServiceServer_CancelInspection(t *testing.T) {
	testCases := []struct {
		name         string
		typeId       string
		startFirst   bool
		wantCode     connect.Code
		wantErrorMsg string
	}{
		{
			name:         "returns failed precondition when cancelling unstarted task",
			typeId:       "gcp-gke",
			startFirst:   false,
			wantCode:     connect.CodeFailedPrecondition,
			wantErrorMsg: "this task is not yet started",
		},
		{
			name:         "cancels started inspection run successfully",
			typeId:       "gcp-gke",
			startFirst:   true,
			wantCode:     0,
			wantErrorMsg: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, _ := setupTestInspectionServer(t, 30*time.Second, 1*time.Second)
			defer ts.Close()

			createRes, err := client.CreateInspection(context.Background(), connect.NewRequest(&apiv1.CreateInspectionRequest{
				InspectionTypeId: proto.String(tc.typeId),
			}))
			if err != nil {
				t.Fatalf("CreateInspection() unexpected error: %v", err)
			}
			inspectionID := createRes.Msg.GetInspectionId()

			if tc.startFirst {
				_, err = client.RunInspection(context.Background(), connect.NewRequest(&apiv1.RunInspectionRequest{
					InspectionId: proto.String(inspectionID),
					Parameters:   &apiv1.InspectionParameters{},
				}))
				if err != nil {
					t.Fatalf("RunInspection() unexpected error: %v", err)
				}
			}

			_, err = client.CancelInspection(context.Background(), connect.NewRequest(&apiv1.CancelInspectionRequest{
				InspectionId: proto.String(inspectionID),
			}))
			if tc.wantCode != 0 {
				if err == nil {
					t.Fatal("CancelInspection() expected error, got nil")
				}

				connErr, ok := err.(*connect.Error)
				if !ok {
					t.Fatalf("expected connect.Error, got %T: %v", err, err)
				}
				if connErr.Code() != tc.wantCode {
					t.Errorf("CancelInspection() error code = %v, want %v", connErr.Code(), tc.wantCode)
				}
				if diff := cmp.Diff(tc.wantErrorMsg, connErr.Message()); diff != "" {
					t.Errorf("CancelInspection() error message mismatch (-want +got):\n%s", diff)
				}
			} else if err != nil {
				t.Fatalf("CancelInspection() unexpected error: %v", err)
			}
		})
	}
}

func TestInspectionServiceServer_GetInspectionMetadata(t *testing.T) {
	testCases := []struct {
		name       string
		header     *inspectionmetadata.HeaderMetadata
		plan       *inspectionmetadata.InspectionPlanMetadata
		wantHeader *apiv1.InspectionHeader
		wantPlan   *apiv1.InspectionPlan
	}{
		{
			name: "returns inspection metadata",
			header: &inspectionmetadata.HeaderMetadata{
				InspectionType:    "gcp-gke",
				InspectionName:    "Test Run",
				SuggestedFileName: "Test Run.khi",
				FileSize:          1234,
			},
			plan: &inspectionmetadata.InspectionPlanMetadata{
				TaskGraph: "graph TD; A-->B;",
			},
			wantHeader: &apiv1.InspectionHeader{
				InspectionType:         proto.String("gcp-gke"),
				InspectionName:         proto.String("Test Run"),
				InspectionTypeIconPath: proto.String(""),
				StartTimeUnixSeconds:   proto.Int64(0),
				EndTimeUnixSeconds:     proto.Int64(0),
				InspectTimeUnixSeconds: proto.Int64(0),
				SuggestedFilename:      proto.String("Test Run.khi"),
				FileSize:               proto.Int64(1234),
			},
			wantPlan: &apiv1.InspectionPlan{
				TaskGraph: proto.String("graph TD; A-->B;"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, server := setupTestInspectionServer(t, 30*time.Second, 1*time.Second)
			defer ts.Close()

			filePath := filepath.Join(t.TempDir(), "result.khi")
			_ = os.WriteFile(filePath, []byte("data"), 0644)
			store := inspectioncore_contract.NewFileSystemInspectionResultRepository(filePath)
			metadata := typedmap.NewTypedMap()
			typedmap.Set(metadata, inspectionmetadata.HeaderMetadataKey, tc.header)
			typedmap.Set(metadata, inspectionmetadata.InspectionPlanMetadataKey, tc.plan)
			server.RegisterImportedInspection("metadata-test-1", store, metadata.AsReadonly())

			res, err := client.GetInspectionMetadata(context.Background(), connect.NewRequest(&apiv1.GetInspectionMetadataRequest{
				InspectionId: proto.String("metadata-test-1"),
			}))
			if err != nil {
				t.Fatalf("GetInspectionMetadata() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.wantHeader, res.Msg.GetHeader(), protocmp.Transform()); diff != "" {
				t.Errorf("header mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantPlan, res.Msg.GetPlan(), protocmp.Transform()); diff != "" {
				t.Errorf("plan mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInspectionServiceServer_RunInspection(t *testing.T) {
	testCases := []struct {
		name   string
		typeId string
	}{
		{
			name:   "initiates inspection run successfully and does not cancel on request completion",
			typeId: "gcp-gke",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, server := setupTestInspectionServer(t, 30*time.Second, 1*time.Second)
			defer ts.Close()

			createRes, err := client.CreateInspection(context.Background(), connect.NewRequest(&apiv1.CreateInspectionRequest{
				InspectionTypeId: proto.String(tc.typeId),
			}))
			if err != nil {
				t.Fatalf("CreateInspection() unexpected error: %v", err)
			}
			inspectionID := createRes.Msg.GetInspectionId()

			reqCtx, reqCancel := context.WithCancel(context.Background())
			_, err = client.RunInspection(reqCtx, connect.NewRequest(&apiv1.RunInspectionRequest{
				InspectionId: proto.String(inspectionID),
				Parameters:   &apiv1.InspectionParameters{},
			}))
			reqCancel() // Cancel caller context immediately to simulate HTTP request lifecycle completion.
			if err != nil {
				t.Fatalf("RunInspection() unexpected error: %v", err)
			}

			task := server.GetInspection(inspectionID)
			if task == nil {
				t.Fatalf("inspection %s was not found", inspectionID)
			}
			<-task.Wait()

			md, err := task.GetCurrentMetadata()
			if err != nil {
				t.Fatalf("GetCurrentMetadata() unexpected error: %v", err)
			}
			progress, found := typedmap.Get(md, inspectionmetadata.ProgressMetadataKey)
			if !found {
				t.Fatalf("progress metadata not found")
			}
			if diff := cmp.Diff(inspectionmetadata.TaskPhaseCancelled == progress.Phase, false); diff != "" {
				t.Errorf("task cancellation status mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
