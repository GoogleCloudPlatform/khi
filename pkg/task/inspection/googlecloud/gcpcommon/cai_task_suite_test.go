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

package gcpcommon

import (
	"context"
	"errors"
	"testing"
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSnapshotToCAIRawLog(t *testing.T) {
	startTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	dataStruct, err := structpb.NewStruct(map[string]any{
		"name": "my-resource",
	})
	if err != nil {
		t.Fatalf("failed to create structpb: %v", err)
	}

	snapshot := &CAIAssetSnapshot{
		TemporalAsset: &assetpb.TemporalAsset{
			Asset: &assetpb.Asset{
				Name:      "//compute.googleapis.com/projects/p/zones/z/instances/i",
				AssetType: "compute.googleapis.com/Instance",
				Resource: &assetpb.Resource{
					Data: dataStruct,
				},
			},
			Window: &assetpb.TimeWindow{
				StartTime: timestamppb.New(startTime),
			},
		},
	}

	testCases := []struct {
		name             string
		preprocessRawMap func(m map[string]any)
		wantCustomVal    string
	}{
		{
			name:             "without preprocess callback",
			preprocessRawMap: nil,
			wantCustomVal:    "",
		},
		{
			name: "with preprocess callback",
			preprocessRawMap: func(m map[string]any) {
				m["customField"] = "injected"
			},
			wantCustomVal: "injected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idGen := id.NewGenerator()
			gotLog, err := SnapshotToCAIRawLog(idGen, snapshot, tc.preprocessRawMap)
			if err != nil {
				t.Fatalf("SnapshotToCAIRawLog() unexpected error: %v", err)
			}
			if !gotLog.Timestamp.Equal(startTime) {
				t.Errorf("SnapshotToCAIRawLog() timestamp = %v, want %v", gotLog.Timestamp, startTime)
			}
			customVal := gotLog.NodeReader.ReadStringOrDefault(structured.CompileFieldPath("customField"), "")
			if customVal != tc.wantCustomVal {
				t.Errorf("SnapshotToCAIRawLog() customField = %q, want %q", customVal, tc.wantCustomVal)
			}
		})
	}
}

func makeCAITestLog(t *testing.T, assetName string, start, end time.Time, deleted bool, version string) *log.Log {
	t.Helper()
	m := map[string]any{
		"asset": map[string]any{
			"name":      assetName,
			"assetType": "test/Type",
			"resource": map[string]any{
				"data": map[string]any{
					"version": version,
				},
			},
		},
		"deleted": deleted,
		"window":  map[string]any{},
	}
	windowMap := m["window"].(map[string]any)
	if !start.IsZero() {
		windowMap["startTime"] = start.Format(time.RFC3339)
	}
	if !end.IsZero() {
		windowMap["endTime"] = end.Format(time.RFC3339)
	}
	reader := newTestNodeReader(t, m)
	return log.NewLogWithTimestamp(id.NewGenerator(), reader, start)
}

func TestExtractCAIActiveAssetStates(t *testing.T) {
	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

	testCases := []struct {
		name             string
		logs             []*log.Log
		wantIdentity     string
		wantVersion      string
		wantObservedTime time.Time
		wantCount        int
	}{
		{
			name: "chronological order keeps newest active snapshot",
			logs: []*log.Log{
				makeCAITestLog(t, "res-1", t1, time.Time{}, false, "v1"),
				makeCAITestLog(t, "res-1", t2, time.Time{}, false, "v2"),
				makeCAITestLog(t, "res-2", t1, t2, false, "expired"),
				makeCAITestLog(t, "res-3", t2, time.Time{}, true, "del"),
			},
			wantCount:        1,
			wantIdentity:     "res-1",
			wantVersion:      "v2",
			wantObservedTime: t2,
		},
		{
			name: "out-of-order logs still select snapshot with latest observed time",
			logs: []*log.Log{
				makeCAITestLog(t, "res-1", t2, time.Time{}, false, "v2"),
				makeCAITestLog(t, "res-1", t1, time.Time{}, false, "v1"),
			},
			wantCount:        1,
			wantIdentity:     "res-1",
			wantVersion:      "v2",
			wantObservedTime: t2,
		},
		{
			name: "normalizes zero window start time to queryStartTime and wins deduplication over older snapshot",
			logs: []*log.Log{
				makeCAITestLog(t, "res-1", t1, time.Time{}, false, "v1"),
				makeCAITestLog(t, "res-1", time.Time{}, time.Time{}, false, "v-zero"),
			},
			wantCount:        1,
			wantIdentity:     "res-1",
			wantVersion:      "v-zero",
			wantObservedTime: queryStartTime,
		},
		{
			name: "skips logs where identity or body extraction fails",
			logs: []*log.Log{
				makeCAITestLog(t, "", t2, time.Time{}, false, "v2"),
			},
			wantCount: 0,
		},
	}

	extractIdentity := func(reader *structured.NodeReader) (string, bool) {
		name := ExtractCAIAssetName(reader)
		return name, name != ""
	}
	identityKey := func(id string) string { return id }
	extractBody := func(reader *structured.NodeReader) structured.Node {
		return ExtractCAIResourceBody(reader)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			states := ExtractCAIActiveAssetStates(tc.logs, queryStartTime, extractIdentity, identityKey, extractBody)
			if len(states) != tc.wantCount {
				t.Fatalf("ExtractCAIActiveAssetStates() count = %d, want %d", len(states), tc.wantCount)
			}
			if tc.wantCount > 0 {
				if states[0].Identity != tc.wantIdentity {
					t.Errorf("state[0].Identity = %q, want %q", states[0].Identity, tc.wantIdentity)
				}
				if !states[0].ObservedTime.Equal(tc.wantObservedTime) {
					t.Errorf("state[0].ObservedTime = %v, want %v", states[0].ObservedTime, tc.wantObservedTime)
				}
				bodyReader := structured.NewNodeReader(states[0].ResourceBody)
				gotVersion := bodyReader.ReadStringOrDefault(structured.CompileFieldPath("version"), "")
				if gotVersion != tc.wantVersion {
					t.Errorf("state[0] version = %q, want %q", gotVersion, tc.wantVersion)
				}
			}
		})
	}
}

func newTestCAITaskSuiteConfig(targetTimeline *khifilev6.TimelinePath) CAITaskSuiteConfig[string] {
	taskIDs := NewCAITaskIDSet("cloud.google.com/cai/test/")
	return CAITaskSuiteConfig[string]{
		TaskIDs: taskIDs,
		ExtractIdentity: func(reader *structured.NodeReader) (string, bool) {
			name := ExtractCAIAssetName(reader)
			return name, name != ""
		},
		IdentityGroupKey: func(id string) string {
			if id == "empty-key" {
				return ""
			}
			return "group:" + id
		},
		FormatLogSummary: func(id string) string {
			return "summary:" + id
		},
		MapInitialRevision: func(_ context.Context, _ *log.Log, identity string, observedTime time.Time) (CAIInitialSnapshotRevisionSpec, bool, error) {
			if identity == "skip-me" {
				return CAIInitialSnapshotRevisionSpec{}, true, nil
			}
			return CAIInitialSnapshotRevisionSpec{
				TargetTimeline:    targetTimeline,
				ObservedTime:      observedTime,
				CreationStateType: k8saudit.RevisionStateK8sResourceExistingLogNotFound,
				SnapshotStateType: k8saudit.RevisionStateK8sClusterExistingLogNotFound,
			}, false, nil
		},
	}
}

func TestCAITaskSuite_FetcherTask(t *testing.T) {
	testCases := []struct {
		name          string
		mode          inspectioncore.InspectionTaskModeType
		skip          bool
		targetErr     error
		discoverErr   error
		mockServer    *mockCAIAssetServer
		wantSnapshots int
		wantErr       bool
	}{
		{
			name:          "DryRun mode returns empty slice immediately",
			mode:          inspectioncore.TaskModeDryRun,
			skip:          false,
			wantSnapshots: 0,
			wantErr:       false,
		},
		{
			name:          "Run mode with skip=true returns empty slice immediately",
			mode:          inspectioncore.TaskModeRun,
			skip:          true,
			wantSnapshots: 0,
			wantErr:       false,
		},
		{
			name:          "Run mode propagates ResolveSearchTarget error",
			mode:          inspectioncore.TaskModeRun,
			targetErr:     errors.New("resolve target error"),
			wantSnapshots: 0,
			wantErr:       true,
		},
		{
			name:          "Run mode suppresses Discover error and returns empty slice",
			mode:          inspectioncore.TaskModeRun,
			discoverErr:   errors.New("discover error"),
			wantSnapshots: 0,
			wantErr:       false,
		},
		{
			name: "Run mode fetches snapshots using configured mock server",
			mode: inspectioncore.TaskModeRun,
			mockServer: &mockCAIAssetServer{
				batchAssets: []*assetpb.TemporalAsset{
					{
						Asset: &assetpb.Asset{
							Name: "//compute.googleapis.com/projects/p1/zones/z/instances/i-1",
						},
					},
				},
			},
			wantSnapshots: 1,
			wantErr:       false,
		},
		{
			name: "Run mode suppresses fetch failure error and returns empty slice",
			mode: inspectioncore.TaskModeRun,
			mockServer: &mockCAIAssetServer{
				batchErr: errors.New("simulated fetch failure"),
			},
			wantSnapshots: 0,
			wantErr:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var factory *googlecloud.ClientFactory
			if tc.mockServer != nil {
				factory = setupMockCAIServer(t, tc.mockServer)
			}
			cfg := newTestCAITaskSuiteConfig(&khifilev6.TimelinePath{ID: 1})
			cfg.ResolveSearchTarget = func(_ context.Context, _ inspectioncore.InspectionTaskModeType) (string, CAIAssetSearchTarget, bool, error) {
				if tc.targetErr != nil {
					return "", CAIAssetSearchTarget{}, false, tc.targetErr
				}
				return "p1", CAIAssetSearchTarget{
					Scope: "projects/p1",
					Discover: func(_ context.Context, _ CAIFetcher) ([]string, error) {
						if tc.discoverErr != nil {
							return nil, tc.discoverErr
						}
						return []string{"//compute.googleapis.com/projects/p1/zones/z/instances/i-1"}, nil
					},
				}, tc.skip, nil
			}
			suite := NewCAITaskSuite(cfg)

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			snapshots, _, err := inspectiontest.RunInspectionTask(ctx, suite.FetcherTask, tc.mode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(APIClientFactoryTaskID.Ref(), factory),
				tasktest.NewTaskDependencyValuePair(APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
				tasktest.NewTaskDependencyValuePair(InputStartTimeTaskID.Ref(), time.Now()),
				tasktest.NewTaskDependencyValuePair(InputEndTimeTaskID.Ref(), time.Now()),
			)
			if (err != nil) != tc.wantErr {
				t.Fatalf("FetcherTask error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && len(snapshots) != tc.wantSnapshots {
				t.Errorf("len(snapshots) = %d, want %d", len(snapshots), tc.wantSnapshots)
			}
		})
	}
}

func TestCAITaskSuite_LogGrouper(t *testing.T) {
	suite := NewCAITaskSuite(newTestCAITaskSuiteConfig(&khifilev6.TimelinePath{ID: 1}))
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	testCases := []struct {
		name      string
		assetName string
		wantGroup string
	}{
		{
			name:      "groups recognized identity by group key",
			assetName: "pod-1",
			wantGroup: "group:pod-1",
		},
		{
			name:      "groups unrecognized identity under unknown",
			assetName: "",
			wantGroup: "unknown",
		},
		{
			name:      "groups empty group key under unknown",
			assetName: "empty-key",
			wantGroup: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logs := []*log.Log{makeCAITestLog(t, tc.assetName, t1, time.Time{}, false, "v1")}
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			grouped, _, err := inspectiontest.RunInspectionTask(ctx, suite.LogGrouperTask, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(suite.LogGrouperTask.Dependencies()[0].(taskid.TaskReference[[]*log.Log]), logs))
			if err != nil {
				t.Fatalf("LogGrouperTask error: %v", err)
			}
			if len(grouped[tc.wantGroup].Logs) != 1 {
				t.Errorf("grouped[%q].Logs len = %d, want 1", tc.wantGroup, len(grouped[tc.wantGroup].Logs))
			}
		})
	}
}

func TestCAITaskSuite_Ingester(t *testing.T) {
	cfg := newTestCAITaskSuiteConfig(&khifilev6.TimelinePath{ID: 1})
	ingester := &caiLogIngester[string]{
		rawLogRef:        cfg.TaskIDs.RawLog.Ref(),
		extractIdentity:  cfg.ExtractIdentity,
		formatLogSummary: cfg.FormatLogSummary,
	}
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	testCases := []struct {
		name        string
		assetName   string
		wantSummary string
	}{
		{
			name:        "ingests log with identity and summary",
			assetName:   "pod-1",
			wantSummary: "summary:pod-1",
		},
		{
			name:        "ingests log without identity using unknown fallback summary",
			assetName:   "",
			wantSummary: "CAI resource snapshot: unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := makeCAITestLog(t, tc.assetName, t1, time.Time{}, false, "v1")
			cs, err := ingester.ProcessLog(t.Context(), l)
			if err != nil {
				t.Fatalf("ProcessLog() error: %v", err)
			}
			testchangeset.AssertLog(t, cs).
				HasTimestamp(t1).
				HasLogType(LogTypeCAIResourceSnapshot).
				HasSeverity(inspectioncore.SeverityInfo).
				HasSummary(tc.wantSummary)
		})
	}
}

func TestCAITaskSuite_Mapper(t *testing.T) {
	targetTimeline := &khifilev6.TimelinePath{ID: 1}
	cfg := newTestCAITaskSuiteConfig(targetTimeline)
	mapper := &caiTimelineMapper[string]{
		logIngesterRef:     cfg.TaskIDs.LogIngester.Ref(),
		groupedLogRef:      cfg.TaskIDs.LogGrouper.Ref(),
		dependencies:       append([]coretask.Dependency{InputStartTimeTaskID.Ref()}, cfg.MapperDependencies...),
		extractIdentity:    cfg.ExtractIdentity,
		mapInitialRevision: cfg.MapInitialRevision,
	}
	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

	testCases := []struct {
		name         string
		log          *log.Log
		wantNil      bool
		assertResult func(t *testing.T, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name:    "stages initial revision for active snapshot",
			log:     makeCAITestLog(t, "pod-1", t1, time.Time{}, false, "v1"),
			wantNil: false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(targetTimeline, &khifilev6.StagingRevision{
						ChangedTime: t1,
						Principal:   "N/A",
						VerbType:    k8saudit.VerbCreate,
						StateType:   k8saudit.RevisionStateK8sClusterExistingLogNotFound,
					})
			},
		},
		{
			name:    "skips deleted tombstone snapshot",
			log:     makeCAITestLog(t, "pod-1", t1, time.Time{}, true, "v1"),
			wantNil: true,
		},
		{
			name:    "skips snapshot with inactive time window",
			log:     makeCAITestLog(t, "pod-1", t1.Add(-2*time.Hour), t1.Add(-1*time.Hour), false, "v1"),
			wantNil: true,
		},
		{
			name:    "skips log with unrecognized identity",
			log:     makeCAITestLog(t, "", t1, time.Time{}, false, "v1"),
			wantNil: true,
		},
		{
			name:    "stages initial revision using queryStartTime when windowStartTime is zero",
			log:     makeCAITestLog(t, "pod-1", time.Time{}, time.Time{}, false, "v1"),
			wantNil: false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(targetTimeline, &khifilev6.StagingRevision{
						ChangedTime: queryStartTime,
						Principal:   "N/A",
						VerbType:    k8saudit.VerbCreate,
						StateType:   k8saudit.RevisionStateK8sClusterExistingLogNotFound,
					})
			},
		},
		{
			name:    "skips log when MapInitialRevision returns skip=true",
			log:     makeCAITestLog(t, "skip-me", t1, time.Time{}, false, "v1"),
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			ctx = tasktest.WithTaskResult(ctx, InputStartTimeTaskID.Ref(), queryStartTime)
			cs, _, err := mapper.ProcessLogByGroup(ctx, tc.log, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() error: %v", err)
			}
			if tc.wantNil {
				if cs != nil {
					t.Errorf("ProcessLogByGroup() = %v, want nil", cs)
				}
				return
			}
			if cs == nil {
				t.Fatal("ProcessLogByGroup() = nil, want non-nil changeset")
			}
			tc.assertResult(t, cs)
		})
	}
}

func TestNewCAIInitialResourceStateProviderTask(t *testing.T) {
	taskID := taskid.NewDefaultImplementationID[[]string]("cloud.google.com/cai/test/provider")
	suiteTaskIDs := NewCAITaskIDSet("cloud.google.com/cai/test/")

	providerTask := NewCAIInitialResourceStateProviderTask[string, []string](
		taskID,
		suiteTaskIDs,
		func(reader *structured.NodeReader) (string, bool) {
			name := ExtractCAIAssetName(reader)
			return name, name != ""
		},
		func(id string) string { return id },
		func(reader *structured.NodeReader) structured.Node { return ExtractCAIResourceBody(reader) },
		func(states []CAIActiveAssetState[string]) []string {
			if states == nil {
				return nil
			}
			var res []string
			for _, s := range states {
				res = append(res, s.Identity)
			}
			return res
		},
	)

	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	logs := []*log.Log{makeCAITestLog(t, "pod-1", t1, time.Time{}, false, "v1")}

	testCases := []struct {
		name     string
		taskMode inspectioncore.InspectionTaskModeType
		want     []string
	}{
		{
			name:     "dryrun mode returns nil",
			taskMode: inspectioncore.TaskModeDryRun,
			want:     nil,
		},
		{
			name:     "run mode builds provider from active asset states",
			taskMode: inspectioncore.TaskModeRun,
			want:     []string{"pod-1"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.RunInspectionTask(ctx, providerTask, tc.taskMode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(suiteTaskIDs.RawLog.Ref(), logs),
				tasktest.NewTaskDependencyValuePair(InputStartTimeTaskID.Ref(), queryStartTime),
			)
			if err != nil {
				t.Fatalf("RunInspectionTask() error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("provider output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
