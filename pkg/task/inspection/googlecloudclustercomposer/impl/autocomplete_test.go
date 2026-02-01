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

package googlecloudclustercomposer_impl

import (
	"context"
	"net"
	"sort"
	"testing"
	"time"

	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/api/option"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type mockMetricServer struct {
	monitoringpb.UnimplementedMetricServiceServer
	listTimeSeriesFunc func(context.Context, *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error)
}

func (m *mockMetricServer) ListTimeSeries(ctx context.Context, req *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error) {
	if m.listTimeSeriesFunc != nil {
		return m.listTimeSeriesFunc(ctx, req)
	}
	return &monitoringpb.ListTimeSeriesResponse{}, nil
}

func TestAutocompleteComposerEnvironmentIdentityTask(t *testing.T) {
	// Setup gRPC mock server
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	mockServer := &mockMetricServer{
		listTimeSeriesFunc: func(ctx context.Context, req *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error) {
			return &monitoringpb.ListTimeSeriesResponse{
				TimeSeries: []*monitoringpb.TimeSeries{
					{
						Metric: &metricpb.Metric{
							Type: "composer.googleapis.com/environment/healthy",
						},
						Resource: &monitoredres.MonitoredResource{
							Type: "cloud_composer_environment",
							Labels: map[string]string{
								"project_id":       "test-project",
								"environment_name": "test-env-1",
								"location":         "us-central1",
							},
						},
					},
					{
						Metric: &metricpb.Metric{
							Type: "composer.googleapis.com/environment/healthy",
						},
						Resource: &monitoredres.MonitoredResource{
							Type: "cloud_composer_environment",
							Labels: map[string]string{
								"project_id":       "test-project",
								"environment_name": "test-env-2",
								"location":         "europe-west1",
							},
						},
					},
				},
			}, nil
		},
	}
	monitoringpb.RegisterMetricServiceServer(s, mockServer)
	go func() {
		if err := s.Serve(lis); err != nil {
			// Server might be closed
		}
	}()
	defer s.Stop()

	// Use real ClientFactory configured to point to mock server
	factory, err := googlecloud.NewClientFactory(func(f *googlecloud.ClientFactory) error {
		f.MonitoringMetricClientOptions = append(f.MonitoringMetricClientOptions, func(opts []option.ClientOption, _ googlecloud.ResourceContainer) ([]option.ClientOption, error) {
			return append(opts,
				option.WithEndpoint(lis.Addr().String()),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
				option.WithoutAuthentication(),
			), nil
		})
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create client factory: %v", err)
	}

	injector := googlecloud.NewCallOptionInjector()

	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
	inspectionTask := AutocompleteComposerEnvironmentIdentityTask

	result, _, err := inspectiontest.RunInspectionTask(ctx, inspectionTask, inspectioncore_contract.TaskModeDryRun, nil,
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputProjectIdTaskID.Ref(), "test-project"),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputStartTimeTaskID.Ref(), startTime),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputEndTimeTaskID.Ref(), endTime),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientFactoryTaskID.Ref(), factory),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(), injector),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{
		{
			ProjectID:       "test-project",
			Location:        "us-central1",
			EnvironmentName: "test-env-1",
		},
		{
			ProjectID:       "test-project",
			Location:        "europe-west1",
			EnvironmentName: "test-env-2",
		},
	}

	if diff := cmp.Diff(expected, result.Values, cmpopts.SortSlices(func(a, b googlecloudclustercomposer_contract.ComposerEnvironmentIdentity) bool {
		return a.EnvironmentName < b.EnvironmentName
	})); diff != "" {
		t.Errorf("unexpected result (-want +got):\n%s", diff)
	}
}

func TestAutocompleteLocationForComposerEnvironmentTask(t *testing.T) {
	testCases := []struct {
		desc      string
		projectID string
		envName   string
		input     *inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]
		want      *inspectioncore_contract.AutocompleteResult[string]
	}{
		{
			desc:      "project id is empty",
			projectID: "",
			input: &inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
				Values: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{},
				Error:  "",
				Hint:   "",
			},
			want: &inspectioncore_contract.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   "Locations are suggested after the project ID is provided.",
			},
		},
		{
			desc:      "identities with error",
			projectID: "foo-project",
			envName:   "foo-env",
			input: &inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
				Values: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{},
				Error:  "some error",
				Hint:   "some hint",
			},
			want: &inspectioncore_contract.AutocompleteResult[string]{
				Values: []string{},
				Error:  "some error",
				Hint:   "some hint",
			},
		},
		{
			desc:      "environment name is empty",
			projectID: "foo-project",
			envName:   "",
			input: &inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
				Values: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{
					{ProjectID: "foo-project", Location: "us-central1", EnvironmentName: "env1"},
				},
				Error: "",
				Hint:  "",
			},
			want: &inspectioncore_contract.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   "Locations are suggested after the environment name is provided.",
			},
		},
		{
			desc:      "filter by environment name",
			projectID: "foo-project",
			envName:   "env1",
			input: &inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
				Values: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{
					{ProjectID: "foo-project", Location: "us-central1", EnvironmentName: "env1"},
					{ProjectID: "foo-project", Location: "asia-northeast1", EnvironmentName: "env3"},
				},
				Error: "",
				Hint:  "",
			},
			want: &inspectioncore_contract.AutocompleteResult[string]{
				Values: []string{"us-central1"},
				Error:  "",
				Hint:   "",
			},
		},
		{
			desc:      "filter by environment name mismatch",
			projectID: "foo-project",
			envName:   "env", // Partial match but not exact
			input: &inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
				Values: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{
					{ProjectID: "foo-project", Location: "us-central1", EnvironmentName: "env1"},
				},
				Error: "",
				Hint:  "",
			},
			want: &inspectioncore_contract.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())

			projectIDInput := tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputProjectIdTaskID.Ref(), tc.projectID)
			envNameInput := tasktest.NewTaskDependencyValuePair(googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(), tc.envName)
			startTimeInput := tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputStartTimeTaskID.Ref(), time.Now())
			endTimeInput := tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputEndTimeTaskID.Ref(), time.Now())
			identitiesInput := tasktest.NewTaskDependencyValuePair(googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID.Ref(), tc.input)

			result, _, err := inspectiontest.RunInspectionTask(ctx, AutocompleteLocationForComposerEnvironmentTask, inspectioncore_contract.TaskModeDryRun, map[string]any{}, projectIDInput, envNameInput, startTimeInput, endTimeInput, identitiesInput)
			if err != nil {
				t.Fatalf("failed to run inspection task: %v", err)
			}

			// Sort values for deterministic comparison because map iteration is random
			if len(result.Values) > 0 {
				sort.Strings(result.Values)
			}
			if len(tc.want.Values) > 0 {
				sort.Strings(tc.want.Values)
			}

			if diff := cmp.Diff(tc.want, result); diff != "" {
				t.Errorf("result of AutocompleteLocationForComposerEnvironmentTask mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
