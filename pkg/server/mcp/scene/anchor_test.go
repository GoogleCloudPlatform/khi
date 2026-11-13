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

package scene

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/inspectionserver"
)

func TestMain(m *testing.M) {
	logger.InitGlobalKHILogger()
	os.Exit(m.Run())
}

func TestFormatShiftedClock(t *testing.T) {
	tm := time.Date(2026, 10, 1, 9, 30, 15, 0, time.UTC)
	testCases := []struct {
		name       string
		time       time.Time
		shiftHours float64
		want       string
	}{
		{
			name: "zero time returns empty",
			time: time.Time{},
			want: "",
		},
		{
			name:       "UTC offset 0",
			time:       tm,
			shiftHours: 0,
			want:       "09:30:15Z",
		},
		{
			name:       "positive whole-hour offset",
			time:       tm,
			shiftHours: 9,
			want:       "18:30:15 (UTC+9)",
		},
		{
			name:       "negative half-hour offset",
			time:       tm,
			shiftHours: -3.5,
			want:       "06:00:15 (UTC-3:30)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatShiftedClock(tc.time, tc.shiftHours)
			if got != tc.want {
				t.Errorf("formatShiftedClock() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatElapsedDelta(t *testing.T) {
	testCases := []struct {
		name   string
		d      time.Duration
		signed bool
		want   string
	}{
		{
			name:   "zero unsigned",
			d:      0,
			signed: false,
			want:   "0s",
		},
		{
			name:   "zero signed",
			d:      0,
			signed: true,
			want:   "+0s",
		},
		{
			name:   "sub-second ms unsigned",
			d:      450 * time.Millisecond,
			signed: false,
			want:   "450ms",
		},
		{
			name:   "seconds signed positive",
			d:      42 * time.Second,
			signed: true,
			want:   "+42s",
		},
		{
			name:   "seconds signed negative",
			d:      -15 * time.Second,
			signed: true,
			want:   "-15s",
		},
		{
			name:   "minutes and seconds",
			d:      2*time.Minute + 30*time.Second,
			signed: false,
			want:   "2m 30s",
		},
		{
			name:   "exact minutes",
			d:      5 * time.Minute,
			signed: false,
			want:   "5m",
		},
		{
			name:   "hours and minutes",
			d:      1*time.Hour + 15*time.Minute,
			signed: true,
			want:   "+1h 15m",
		},
		{
			name:   "exact hours",
			d:      3 * time.Hour,
			signed: false,
			want:   "3h",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatElapsedDelta(tc.d, tc.signed)
			if got != tc.want {
				t.Errorf("formatElapsedDelta() = %q, want %q", got, tc.want)
			}
		})
	}
}

type anchorFixture struct {
	wb            *workbench.Workbench
	inspectionID  string
	baseTime      time.Time
	nodeTid       uint32
	podTid        uint32
	appTid        uint32
	oomLogID      uint32
	nodeRev0LogID uint32
}

func setupAnchorFixture(t *testing.T) anchorFixture {
	t.Helper()
	baseTime := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	populateTaskID := taskid.NewDefaultImplementationID[struct{}]("populate-anchor-test")
	populateTask := inspectiontaskbase.DefineInspectionTask(
		populateTaskID,
		func(_ *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[struct{}] {
			return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (struct{}, error) {
				if taskMode == inspectioncore.TaskModeDryRun {
					return struct{}{}, nil
				}
				metadataSet := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
				if header, found := typedmap.Get(metadataSet, inspectionmetadata.HeaderMetadataKey); found && header != nil {
					header.StartTimeUnixSeconds = baseTime.Unix()
					header.EndTimeUnixSeconds = baseTime.Add(1 * time.Hour).Unix()
				}

				idGen := khictx.MustGetValue(ctx, inspectioncore.IDGenerator)
				builder := khictx.MustGetValue(ctx, inspectioncore.Builder)
				tlAcc := builder.TimelineAccumulator

				k8sPath := func(apiVersion, kind, namespace, name string) *khifilev6.TimelinePath {
					return tlAcc.GetPath(nil,
						khifilev6.PathSegment{Name: apiVersion, Type: inspectioncore.TimelineTypeAPIVersion},
						khifilev6.PathSegment{Name: kind, Type: inspectioncore.TimelineTypeKind},
						khifilev6.PathSegment{Name: namespace, Type: inspectioncore.TimelineTypeNamespace},
						khifilev6.PathSegment{Name: name, Type: inspectioncore.TimelineTypeResource},
					)
				}

				nodePath := k8sPath("core/v1", "node", "cluster-scope", "node-1")
				podPath := k8sPath("core/v1", "pod", "default", "pod-1")
				appPath := tlAcc.GetPath(podPath, khifilev6.PathSegment{Name: "app", Type: k8saudit.TimelineTypeContainer})

				recordRev := func(offset time.Duration, path *khifilev6.TimelinePath, logYAML, resourceYAML string, verb *pb.Verb, sev *pb.Severity) error {
					ts := baseTime.Add(offset)
					logBody, err := structured.FromYAML(logYAML)
					if err != nil {
						return err
					}
					resBody, err := structured.FromYAML(resourceYAML)
					if err != nil {
						return err
					}
					l := log.NewLogWithTimestamp(idGen, structured.NewNodeReader(logBody), ts)
					logCS, err := khifilev6.NewLogChangeSet(l)
					if err != nil {
						return err
					}
					logCS.SetTimestamp(ts)
					logCS.SetSeverity(sev)
					logCS.SetLogType(k8saudit.LogTypeAudit)
					if err := logCS.Flush(builder.LogAccumulator); err != nil {
						return err
					}
					tlCS := khifilev6.NewTimelineChangeSet(l)
					defer tlCS.Release()
					tlCS.AddRevision(path, &khifilev6.StagingRevision{
						ChangedTime:  ts,
						ResourceBody: resBody,
						VerbType:     verb,
						StateType:    k8saudit.RevisionStateK8sResourceExisting,
					})
					return tlCS.Flush(tlAcc, builder.LogAccumulator)
				}

				const nodeY = `apiVersion: v1
kind: Node
metadata:
  name: node-1
`
				const podY = `apiVersion: v1
kind: Pod
metadata:
  name: pod-1
`
				if err := recordRev(10*time.Minute, nodePath, nodeY, nodeY, k8saudit.VerbCreate, inspectioncore.SeverityInfo); err != nil {
					return struct{}{}, err
				}
				if err := recordRev(20*time.Minute, nodePath, nodeY, nodeY, k8saudit.VerbPatch, inspectioncore.SeverityWarning); err != nil {
					return struct{}{}, err
				}
				if err := recordRev(12*time.Minute, podPath, podY, podY, k8saudit.VerbCreate, inspectioncore.SeverityInfo); err != nil {
					return struct{}{}, err
				}

				oomTime := baseTime.Add(25 * time.Minute)
				logBody, err := structured.FromYAML("message: OOM\n")
				if err != nil {
					return struct{}{}, err
				}
				l := log.NewLogWithTimestamp(idGen, structured.NewNodeReader(logBody), oomTime)
				logCS, err := khifilev6.NewLogChangeSet(l)
				if err != nil {
					return struct{}{}, err
				}
				logCS.SetTimestamp(oomTime)
				logCS.SetSeverity(inspectioncore.SeverityError)
				logCS.SetLogType(k8saudit.LogTypeEvent)
				logCS.SetSummary("OOM")
				if err := logCS.Flush(builder.LogAccumulator); err != nil {
					return struct{}{}, err
				}
				tlCS := khifilev6.NewTimelineChangeSet(l)
				defer tlCS.Release()
				tlCS.AddEvent(podPath)
				tlCS.AddEvent(appPath)
				if err := tlCS.Flush(tlAcc, builder.LogAccumulator); err != nil {
					return struct{}{}, err
				}
				return struct{}{}, nil
			}
		},
		coretask.ProvidesTag(inspectiontaskbase.TagTimelineMapper),
		coretask.WithLabelValue(inspectioncore.LabelKeyInspectionDefaultFeatureFlag, true),
		coretask.WithLabelValue(inspectioncore.LabelKeyInspectionFeatureFlag, true),
	)

	srv, inspectionID := inspectionserver.NewServerWithInspection(t, populateTask)
	indexMgr := workbench.NewInspectionIndexManager(srv, t.TempDir())
	mgr := workbench.NewWorkbenchManager(srv, indexMgr, 5)
	t.Cleanup(indexMgr.Wait)

	ctx := context.Background()
	wb, err := mgr.Open(ctx, inspectionID, workbench.AccessorMCP, func(apiv1.OpenWorkbenchResponse_Stage, float64, string) error { return nil })
	if err != nil {
		t.Fatalf("mgr.Open() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if state, _, _, _ := wb.IndexStatus(); state == workbench.IndexStateBuilding {
			_ = wb.AwaitIndex(cleanupCtx)
		}
		wb.Close()
	})
	if err := wb.AwaitIndex(ctx); err != nil {
		t.Fatalf("wb.AwaitIndex() error = %v", err)
	}

	idx := wb.SearchIndex()
	var nodeTid, podTid, appTid, oomLogID uint32
	for _, tl := range idx.Timelines {
		switch tl.Name {
		case "node-1":
			nodeTid = tl.ID
		case "pod-1":
			podTid = tl.ID
		case "app":
			appTid = tl.ID
		}
	}
	for i := range idx.Logs {
		if idx.InternPool.ResolveStringFromID(idx.Logs[i].SummaryStringID) == "OOM" {
			oomLogID = idx.Logs[i].ID
			break
		}
	}
	nodeRev0Bounds, err := wb.ResolveRevisionBounds(nodeTid, 0)
	if err != nil {
		t.Fatalf("wb.ResolveRevisionBounds(nodeTid, 0) error = %v", err)
	}
	return anchorFixture{
		wb:            wb,
		inspectionID:  inspectionID,
		baseTime:      baseTime,
		nodeTid:       nodeTid,
		podTid:        podTid,
		appTid:        appTid,
		oomLogID:      oomLogID,
		nodeRev0LogID: nodeRev0Bounds.Revision.LogID,
	}
}

func TestResolveAnchor(t *testing.T) {
	fx := setupAnchorFixture(t)
	applied := fx.wb.ResolveFilter(workbench.Filter{})
	rawFilter := workbench.Filter{}

	revZero := 0
	fallbackTime := fx.baseTime.Add(5 * time.Minute)
	oomTime := fx.baseTime.Add(25 * time.Minute)
	nodeRev0Time := fx.baseTime.Add(10 * time.Minute)
	nodeRev1Time := fx.baseTime.Add(20 * time.Minute)
	invalidAnchorErr := strings.Join([]string{
		"Error: INVALID_ANCHOR",
		"",
		"- Specify at least one of `logId`, `revisionIndex`, or `time` in `test`.",
	}, "\n")

	testCases := []struct {
		name    string
		anchor  Anchor
		want    resolvedAnchorPoint
		wantErr string
	}{
		{
			name:   "resolves log anchor with auto-selected leaf timeline",
			anchor: Anchor{LogID: fx.oomLogID},
			want: resolvedAnchorPoint{
				time:       oomTime,
				timelineID: fx.appTid,
				logID:      fx.oomLogID,
			},
		},
		{
			name: "resolves revision anchor start edge",
			anchor: Anchor{
				TimelineID:    fx.nodeTid,
				RevisionIndex: &revZero,
				RevisionEdge:  RevisionEdgeStart,
			},
			want: resolvedAnchorPoint{
				time:          nodeRev0Time,
				timelineID:    fx.nodeTid,
				logID:         fx.nodeRev0LogID,
				revisionIndex: &revZero,
			},
		},
		{
			name: "resolves revision anchor end edge",
			anchor: Anchor{
				TimelineID:    fx.nodeTid,
				RevisionIndex: &revZero,
				RevisionEdge:  RevisionEdgeEnd,
			},
			want: resolvedAnchorPoint{
				time:          nodeRev1Time,
				timelineID:    fx.nodeTid,
				logID:         fx.nodeRev0LogID,
				revisionIndex: &revZero,
			},
		},
		{
			name: "prefers logId over revisionIndex and time when multiple fields are set",
			anchor: Anchor{
				TimelineID:    fx.nodeTid,
				LogID:         fx.oomLogID,
				RevisionIndex: &revZero,
				Time:          &fallbackTime,
			},
			want: resolvedAnchorPoint{
				time:       oomTime,
				timelineID: fx.nodeTid,
				logID:      fx.oomLogID,
			},
		},
		{
			name: "prefers revisionIndex over time when both are set",
			anchor: Anchor{
				TimelineID:    fx.nodeTid,
				RevisionIndex: &revZero,
				Time:          &fallbackTime,
			},
			want: resolvedAnchorPoint{
				time:          nodeRev0Time,
				timelineID:    fx.nodeTid,
				logID:         fx.nodeRev0LogID,
				revisionIndex: &revZero,
			},
		},
		{
			name:    "returns INVALID_ANCHOR when no fields are set",
			anchor:  Anchor{},
			wantErr: invalidAnchorErr,
		},
		{
			name:    "returns INVALID_ANCHOR when only zero time is set",
			anchor:  Anchor{Time: &time.Time{}},
			wantErr: invalidAnchorErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, errRes, err := resolveAnchor(fx.wb, fx.inspectionID, applied, rawFilter, "test", tc.anchor)
			if err != nil {
				t.Fatalf("resolveAnchor() unexpected error = %v", err)
			}
			if tc.wantErr != "" {
				if errRes == nil || !errRes.IsError || len(errRes.Content) == 0 {
					t.Fatalf("resolveAnchor() errRes = %+v, want error result", errRes)
				}
				tcText, ok := errRes.Content[0].(*mcpsdk.TextContent)
				if !ok {
					t.Fatalf("errRes.Content[0] = %T, want *mcpsdk.TextContent", errRes.Content[0])
				}
				if diff := cmp.Diff(tc.wantErr, tcText.Text); diff != "" {
					t.Errorf("resolveAnchor() error mismatch (-want +got):\n%s", diff)
				}
				return
			}
			if errRes != nil {
				t.Fatalf("resolveAnchor() unexpected tool error = %+v", errRes)
			}
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(resolvedAnchorPoint{})); diff != "" {
				t.Errorf("resolveAnchor() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveScopedTimelineIDs(t *testing.T) {
	fx := setupAnchorFixture(t)
	ctx := context.Background()
	autoInclude := roaring.NewBitmap()

	ids, errRes, err := resolveScopedTimelineIDs(ctx, fx.wb, fx.inspectionID, "test", `name == "pod-1"`, nil, true, autoInclude)
	if err != nil || errRes != nil {
		t.Fatalf("unexpected error: err=%v, errRes=%+v", err, errRes)
	}
	want := []uint32{fx.podTid, fx.appTid}
	if diff := cmp.Diff(want, ids); diff != "" {
		t.Errorf("resolveScopedTimelineIDs() mismatch (-want +got):\n%s", diff)
	}
	if !autoInclude.Contains(fx.podTid) || !autoInclude.Contains(fx.appTid) {
		t.Errorf("autoInclude missing ids: %v", autoInclude.ToArray())
	}
}
