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

package gcpcommon

import (
	"sync"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSetErrorMetadataForFetchLogError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc             string
		err              error
		wantErrorMessage *inspectionmetadata.ErrorMessage
	}{
		{
			desc: "unauthenticated error",
			err:  status.Error(codes.Unauthenticated, "permission denied"),
			wantErrorMessage: &inspectionmetadata.ErrorMessage{
				ErrorId: 0,
				Message: "rpc error: code = Unauthenticated desc = permission denied",
			},
		},
		{
			desc: "non-grpc error",
			err:  khierrors.ErrInvalidInput,
			wantErrorMessage: &inspectionmetadata.ErrorMessage{
				ErrorId: 0,
				Message: "invalid input",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			setErrorMetadataForFetchLogError(ctx, tt.err)

			metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
			errorMessageSet, found := typedmap.Get(metadata, inspectionmetadata.ErrorMessageSetMetadataKey)
			if !found {
				t.Fatalf("error message set metadata not found")
			}
			if diff := cmp.Diff(tt.wantErrorMessage, errorMessageSet.ErrorMessages[0]); diff != "" {
				t.Errorf("setErrorMetadataForFetchLogError() mismatch (-want +got):\n%s", diff)
			}

		})
	}
}

func TestGroupResourceNamesByContainer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		resourceNames []string
		want          []*resourceContainerLogQueryGroup
		wantErr       bool
	}{
		{
			name: "valid project-based resource names",
			resourceNames: []string{
				"projects/project-1/locations/us-central1/buckets/bucket-1/views/view-1",
				"projects/project-2/locations/us-west1/buckets/bucket-2/views/view-2",
				"projects/project-1/locations/asia-northeast1/buckets/bucket-3/views/view-3",
			},
			want: []*resourceContainerLogQueryGroup{
				{
					container:     googlecloud.Project("project-1"),
					resourceNames: []string{"projects/project-1/locations/us-central1/buckets/bucket-1/views/view-1", "projects/project-1/locations/asia-northeast1/buckets/bucket-3/views/view-3"},
				},
				{
					container:     googlecloud.Project("project-2"),
					resourceNames: []string{"projects/project-2/locations/us-west1/buckets/bucket-2/views/view-2"},
				},
			},
		},
		{
			name: "unsupported resource name format",
			resourceNames: []string{
				"folders/12345",
			},
			wantErr: true,
		},
		{
			name:          "empty resource names",
			resourceNames: []string{},
			want:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := groupResourceNamesByContainer(tt.resourceNames)
			if (err != nil) != tt.wantErr {
				t.Errorf("groupResourceNamesByContainer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(resourceContainerLogQueryGroup{}), cmpopts.AcyclicTransformer("container", func(c googlecloud.ResourceContainer) string { return c.Identifier() })); diff != "" {
				t.Errorf("groupResourceNamesByContainer() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDivideGroupByMaximumResourceName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                    string
		groups                  []*resourceContainerLogQueryGroup
		maxResourceNamePerGroup int
		want                    []*resourceContainerLogQueryGroup
	}{
		{
			name: "group smaller than max",
			groups: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2"}},
			},
			maxResourceNamePerGroup: 3,
			want: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2"}},
			},
		},
		{
			name: "group equal to max",
			groups: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2", "r3"}},
			},
			maxResourceNamePerGroup: 3,
			want: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2", "r3"}},
			},
		},
		{
			name: "group needs multiple splits",
			groups: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2", "r3", "r4", "r5", "r6", "r7"}},
			},
			maxResourceNamePerGroup: 3,
			want: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2", "r3"}},
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r4", "r5", "r6"}},
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r7"}},
			},
		},
		{
			name: "multiple groups, some need splitting",
			groups: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"p1r1", "p1r2", "p1r3", "p1r4"}},
				{container: googlecloud.Project("project-2"), resourceNames: []string{"p2r1", "p2r2"}},
			},
			maxResourceNamePerGroup: 2,
			want: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"p1r1", "p1r2"}},
				{container: googlecloud.Project("project-1"), resourceNames: []string{"p1r3", "p1r4"}},
				{container: googlecloud.Project("project-2"), resourceNames: []string{"p2r1", "p2r2"}},
			},
		},
		{
			name:                    "empty input",
			groups:                  nil,
			maxResourceNamePerGroup: 5,
			want:                    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := divideGroupByMaximumResourceName(tt.groups, tt.maxResourceNamePerGroup)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(resourceContainerLogQueryGroup{}), cmpopts.AcyclicTransformer("container", func(c googlecloud.ResourceContainer) string { return c.Identifier() })); diff != "" {
				t.Errorf("divideGroupByMaximumResourceName() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMonitorProgress(t *testing.T) {
	t.Parallel()

	progressDest := inspectionmetadata.NewTaskProgressMetadata("test-task")
	ctx := progress.WithContext(t.Context(), progressDest)
	tracker := progress.NewRatioTracker(ctx, progress.WithUnit("logs"))
	source := make(chan LogFetchProgress)
	baseLogCount := 100
	listCallIndex := 0
	totalListCalls := 2

	var wg sync.WaitGroup
	monitorProgress(ctx, &wg, source, tracker, baseLogCount, listCallIndex, totalListCalls)

	source <- LogFetchProgress{
		LogCount: 50,
		Progress: 0.5,
	}
	close(source)
	wg.Wait()

	snap := progressDest.Snapshot()
	wantRatio := float32(0+0.5) / float32(2) // 0.25
	if snap.Ratio != wantRatio {
		t.Errorf("snap.Ratio = %v, want %v", snap.Ratio, wantRatio)
	}

	wantMsg := "[1/2] 150 logs"
	if snap.Message != wantMsg {
		t.Errorf("snap.Message = %q, want %q", snap.Message, wantMsg)
	}
}
