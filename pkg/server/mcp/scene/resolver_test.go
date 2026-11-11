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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
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
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type resolverFixture struct {
	wb             *workbench.Workbench
	inspectionID   string
	baseTime       time.Time
	nodeTimelineID uint32
	podTimelineID  uint32
	appTimelineID  uint32
	allPodTreeIDs  []uint32
	oomEventLogID  uint32
}

func setupResolverFixture(t *testing.T) resolverFixture {
	t.Helper()

	baseTime := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	populateTaskID := taskid.NewDefaultImplementationID[struct{}]("populate-scene-resolver-test")
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

				nodePath := k8sPath("core/v1", "node", "cluster-scope", "gke-node-1")
				podPath := k8sPath("core/v1", "pod", "default", "web-1")
				appPath := tlAcc.GetPath(podPath, khifilev6.PathSegment{Name: "app", Type: k8saudit.TimelineTypeContainer})

				recordRev := func(offset time.Duration, path *khifilev6.TimelinePath, logYAML, resourceYAML, summary string, state *pb.RevisionState, verb *pb.Verb, sev *pb.Severity) error {
					ts := baseTime.Add(offset)
					logBodyNode, err := structured.FromYAML(logYAML)
					if err != nil {
						return err
					}
					resourceBodyNode, err := structured.FromYAML(resourceYAML)
					if err != nil {
						return err
					}
					l := log.NewLogWithTimestamp(idGen, structured.NewNodeReader(logBodyNode), ts)
					logCS, err := khifilev6.NewLogChangeSet(l)
					if err != nil {
						return err
					}
					logCS.SetTimestamp(ts)
					logCS.SetSeverity(sev)
					logCS.SetLogType(k8saudit.LogTypeAudit)
					logCS.SetSummary(summary)
					if err := logCS.Flush(builder.LogAccumulator); err != nil {
						return err
					}

					tlCS := khifilev6.NewTimelineChangeSet(l)
					defer tlCS.Release()
					tlCS.AddRevision(path, &khifilev6.StagingRevision{
						ChangedTime:  ts,
						ResourceBody: resourceBodyNode,
						Principal:    "kubelet",
						VerbType:     verb,
						StateType:    state,
					})
					return tlCS.Flush(tlAcc, builder.LogAccumulator)
				}

				const nodeRev0 = `apiVersion: v1
kind: Node
metadata:
  managedFields:
  - manager: kubelet
    operation: Update
  name: gke-node-1
status:
  ready: "True"
  reason: KubeletReady
`
				const nodeRev1 = `apiVersion: v1
kind: Node
metadata:
  managedFields:
  - manager: kubelet
    operation: Apply
  name: gke-node-1
status:
  ready: "False"
  reason: KubeletNotReady
`
				const nodeRev2 = `apiVersion: v1
kind: Node
metadata:
  name: gke-node-1
status:
  ready: "True"
  reason: KubeletReady
`
				const podAudit0 = `insertId: audit-pod-web-1
protoPayload:
  methodName: io.k8s.core.v1.pods.create
`
				const podRev0 = `apiVersion: v1
kind: Pod
metadata:
  name: web-1
status:
  phase: Running
`

				if err := recordRev(10*time.Minute, nodePath, nodeRev0, nodeRev0, "Node Ready=True", k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbCreate, inspectioncore.SeverityInfo); err != nil {
					return struct{}{}, err
				}
				if err := recordRev(24*time.Minute, nodePath, nodeRev1, nodeRev1, "Node Ready=False", k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbPatch, inspectioncore.SeverityWarning); err != nil {
					return struct{}{}, err
				}
				if err := recordRev(34*time.Minute, nodePath, nodeRev2, nodeRev2, "Node recovered", k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbPatch, inspectioncore.SeverityInfo); err != nil {
					return struct{}{}, err
				}
				if err := recordRev(12*time.Minute, podPath, podAudit0, podRev0, "Pod web-1 running", k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbCreate, inspectioncore.SeverityInfo); err != nil {
					return struct{}{}, err
				}

				oomTime := baseTime.Add(25 * time.Minute)
				logNode, err := structured.FromYAML(`insertId: evt-oom
message: "OOMKilled container app\nexitCode: 137"
`)
				if err != nil {
					return struct{}{}, err
				}
				l := log.NewLogWithTimestamp(idGen, structured.NewNodeReader(logNode), oomTime)
				logCS, err := khifilev6.NewLogChangeSet(l)
				if err != nil {
					return struct{}{}, err
				}
				logCS.SetTimestamp(oomTime)
				logCS.SetSeverity(inspectioncore.SeverityError)
				logCS.SetLogType(k8saudit.LogTypeEvent)
				logCS.SetSummary("Container app OOMKilled")
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
		t.Fatalf("mgr.Open() unexpected error = %v", err)
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
		t.Fatalf("wb.AwaitIndex() unexpected error = %v", err)
	}

	idx := wb.SearchIndex()
	var nodeID, podID, appID uint32
	for _, tl := range idx.Timelines {
		switch tl.Name {
		case "gke-node-1":
			nodeID = tl.ID
		case "web-1":
			podID = tl.ID
		case "app":
			appID = tl.ID
		}
	}
	var oomLogID uint32
	for i := range idx.Logs {
		if idx.InternPool.ResolveStringFromID(idx.Logs[i].SummaryStringID) == "Container app OOMKilled" {
			oomLogID = idx.Logs[i].ID
			break
		}
	}
	if nodeID == 0 || podID == 0 || appID == 0 || oomLogID == 0 {
		t.Fatalf("missing fixture IDs: node=%d pod=%d app=%d oomLog=%d", nodeID, podID, appID, oomLogID)
	}

	return resolverFixture{
		wb:             wb,
		inspectionID:   inspectionID,
		baseTime:       baseTime,
		nodeTimelineID: nodeID,
		podTimelineID:  podID,
		appTimelineID:  appID,
		allPodTreeIDs:  []uint32{podID, appID},
		oomEventLogID:  oomLogID,
	}
}

func TestSliceWorkbenchIntegration(t *testing.T) {
	fx := setupResolverFixture(t)

	t.Run("SliceResourceDiff returns ErrTimelineNotFound for unknown timeline", func(t *testing.T) {
		_, err := SliceResourceDiff(fx.wb, 999999, 0, DiffSliceOptions{Paths: []string{"$.metadata.name"}})
		if !errors.Is(err, workbench.ErrTimelineNotFound) {
			t.Fatalf("SliceResourceDiff() error = %v, want %v", err, workbench.ErrTimelineNotFound)
		}
	})

	t.Run("SliceLogDetail returns ErrLogNotFound for unknown log", func(t *testing.T) {
		_, err := SliceLogDetail(fx.wb, 999999, LogSliceOptions{Windows: []SliceWindow{{Path: "$.message"}}})
		if !errors.Is(err, workbench.ErrLogNotFound) {
			t.Fatalf("SliceLogDetail() error = %v, want %v", err, workbench.ErrLogNotFound)
		}
	})

	t.Run("SliceResourceManifest returns ErrTimelineNotFound for unknown timeline", func(t *testing.T) {
		_, err := SliceResourceManifest(fx.wb, 999999, workbench.RevisionAtIndex(0), ManifestSliceOptions{Paths: []string{"$.metadata.name"}})
		if !errors.Is(err, workbench.ErrTimelineNotFound) {
			t.Fatalf("SliceResourceManifest() error = %v, want %v", err, workbench.ErrTimelineNotFound)
		}
	})
}

func TestResolve_AllLayersAndAutoInclude(t *testing.T) {
	fx := setupResolverFixture(t)
	ctx := context.Background()

	alertTime := fx.baseTime.Add(24*time.Minute + 30*time.Second)
	revOne := 1

	spec := Spec{
		InspectionID: fx.inspectionID,
		Filter: workbench.Filter{
			TimelineQuery: `name == "web-1"`,
		},
		Annotations: &Annotations{
			DimUnlistedRows: boolRef(true),
			Palette: []PaletteEntry{
				{ID: "trigger", Color: "amber"},
				{ID: "victim", Color: "red"},
			},
			Markers: []MarkerAnnotation{
				{
					Anchor: Anchor{Time: &alertTime},
					Label:  "PagerDuty alert fired",
					Badge:  "ALERT",
					Style:  MarkerStyleReference,
				},
				{
					Anchor: Anchor{LogID: fx.oomEventLogID},
					Label:  "Container OOMKilled",
				},
			},
			Spans: []SpanAnnotation{
				{
					Label:     "Node NotReady window",
					PaletteID: "trigger",
					Pattern:   SpanPatternHatched,
					Start:     Anchor{TimelineID: fx.nodeTimelineID, RevisionIndex: &revOne, RevisionEdge: RevisionEdgeStart},
					End:       Anchor{TimelineID: fx.nodeTimelineID, RevisionIndex: &revOne, RevisionEdge: RevisionEdgeEnd},
				},
			},
			RowHighlights: []RowHighlight{
				{
					TimelineQuery: `name == "web-1"`,
					RoleLabel:     "VICTIM · Pod",
					PaletteID:     "victim",
				},
			},
			DiffExcerpts: []DiffExcerpt{
				{
					Revision:  RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1},
					Paths:     []string{"$.status.reason"},
					Note:      "Node transitioned to KubeletNotReady",
					PaletteID: "trigger",
				},
			},
			LogExcerpts: []LogExcerpt{
				{
					LogID:   fx.oomEventLogID,
					Windows: []SliceWindow{{Query: "exitCode: 137", BeforeLines: 0, AfterLines: 0}},
					Note:    "Exit code 137 recorded",
				},
			},
			Connectors: []ConnectorAnnotation{
				{
					From:      Anchor{TimelineID: fx.nodeTimelineID, RevisionIndex: &revOne},
					To:        Anchor{LogID: fx.oomEventLogID},
					Badge:     "1",
					PaletteID: "victim",
				},
			},
			Callouts: []CalloutAnnotation{
				{
					Anchor: Anchor{LogID: fx.oomEventLogID},
					Label:  "Leaf container selected automatically",
					Dashed: true,
				},
			},
		},
	}

	nodeBounds, err := fx.wb.ResolveRevisionBounds(fx.nodeTimelineID, 1)
	if err != nil {
		t.Fatalf("ResolveRevisionBounds(node, 1) error = %v", err)
	}
	nodeRev1LogID := nodeBounds.Revision.LogID

	podBounds, err := fx.wb.ResolveRevisionBounds(fx.podTimelineID, 0)
	if err != nil {
		t.Fatalf("ResolveRevisionBounds(pod, 0) error = %v", err)
	}
	podRev0LogID := podBounds.Revision.LogID

	fullOverlay := &Overlay{
		AutoIncludeTimelineIDs: []uint32{fx.nodeTimelineID, fx.podTimelineID, fx.appTimelineID},
		Palette: []ResolvedPaletteColors{
			{
				ID:      "trigger",
				Main:    "#b45309",
				OnMain:  "#fafafa",
				Soft:    "rgba(180, 83, 9, 0.13)",
				Surface: "#f8eee6",
			},
			{
				ID:      "victim",
				Main:    "#d32f2f",
				OnMain:  "#fafafa",
				Soft:    "rgba(211, 47, 47, 0.13)",
				Surface: "#fbeaea",
			},
			{
				ID:      "default-reference",
				Main:    "#5e35b1",
				OnMain:  "#fafafa",
				Soft:    "rgba(94, 53, 177, 0.13)",
				Surface: "#efebf7",
			},
		},
		DimUnlistedRows: true,
		Markers: []ResolvedMarker{
			{
				ID:        "marker-1",
				Badge:     "ALERT",
				Label:     "PagerDuty alert fired",
				Style:     MarkerStyleReference,
				PaletteID: "default-reference",
				TimeMs:    alertTime.UnixMilli(),
				TimeLabel: "09:24:30Z",
			},
			{
				ID:         "marker-2",
				Badge:      "E2",
				Label:      "Container OOMKilled",
				Style:      MarkerStylePin,
				PaletteID:  "victim",
				TimeMs:     fx.baseTime.Add(25 * time.Minute).UnixMilli(),
				TimeLabel:  "09:25:00Z",
				TimelineID: fx.appTimelineID,
				LogID:      fx.oomEventLogID,
			},
		},
		Spans: []ResolvedSpan{
			{
				ID:            "span-1",
				Label:         "Node NotReady window",
				StartTimeMs:   fx.baseTime.Add(24 * time.Minute).UnixMilli(),
				EndTimeMs:     fx.baseTime.Add(34 * time.Minute).UnixMilli(),
				DurationLabel: "10m",
				Pattern:       SpanPatternHatched,
				PaletteID:     "trigger",
			},
		},
		RowHighlights: []ResolvedRowHighlight{
			{
				ID:          "row-highlight-1",
				RoleLabel:   "VICTIM · Pod",
				TimelineIDs: fx.allPodTreeIDs,
				PaletteID:   "victim",
			},
		},
		Excerpts: []ResolvedExcerptCard{
			{
				ID:           "excerpt-1",
				Kind:         ExcerptKindDiff,
				Badge:        "E1",
				PaletteID:    "trigger",
				TimeMs:       fx.baseTime.Add(24 * time.Minute).UnixMilli(),
				TimeLabel:    "09:24:00Z",
				TimelineID:   fx.nodeTimelineID,
				ResourcePath: "node · gke-node-1",
				Title:        "Patch · Resource exists",
				Subtitle:     "+2 / -2 · 7 diff lines",
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 diff lines omitted"},
					{
						Kind:       ExcerptLineKindDeleted,
						OldLineNum: 7,
						Text:       "  reason: KubeletReady",
						Parts: []LinePart{
							{Text: "  "},
							{Text: "reason: KubeletReady", Matched: true},
						},
					},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 7,
						Text:       "  reason: KubeletNotReady",
						Parts: []LinePart{
							{Text: "  "},
							{Text: "reason: KubeletNotReady", Matched: true},
						},
					},
				},
				Note:          "Node transitioned to KubeletNotReady",
				LogTimestamp:  "2026-10-01T09:24:00Z",
				LogID:         nodeRev1LogID,
				RevisionIndex: &revOne,
			},
			{
				ID:           "excerpt-2",
				Kind:         ExcerptKindLog,
				Badge:        "E2",
				PaletteID:    "victim",
				TimeMs:       fx.baseTime.Add(25 * time.Minute).UnixMilli(),
				TimeLabel:    "09:25:00Z",
				TimelineID:   fx.appTimelineID,
				ResourcePath: "web-1/app · container",
				Title:        "k8s event · ERROR",
				Subtitle:     "4 lines",
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindSummary, Text: "Container app OOMKilled"},
					{Kind: ExcerptLineKindOmitted, Omitted: 3, Text: "⋮ 3 lines omitted"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 4,
						Text:       "  exitCode: 137",
						Parts: []LinePart{
							{Text: "  "},
							{Text: "exitCode: 137", Matched: true},
						},
					},
				},
				Note:         "Exit code 137 recorded",
				LogTimestamp: "2026-10-01T09:25:00Z",
				InsertID:     "evt-oom",
				LogID:        fx.oomEventLogID,
			},
		},
		Connectors: []ResolvedConnector{
			{
				ID:             "connector-1",
				Badge:          "1",
				Label:          "+1m",
				Style:          ConnectorStyleCurve,
				PaletteID:      "victim",
				FromTimeMs:     fx.baseTime.Add(24 * time.Minute).UnixMilli(),
				FromTimelineID: fx.nodeTimelineID,
				ToTimeMs:       fx.baseTime.Add(25 * time.Minute).UnixMilli(),
				ToTimelineID:   fx.appTimelineID,
			},
		},
		Callouts: []ResolvedCallout{
			{
				ID:         "callout-1",
				Label:      "Leaf container selected automatically",
				Placement:  CalloutPlacementAuto,
				Dashed:     true,
				PaletteID:  "victim",
				TimeMs:     fx.baseTime.Add(25 * time.Minute).UnixMilli(),
				TimelineID: fx.appTimelineID,
				LogID:      fx.oomEventLogID,
			},
		},
	}

	revZero := 0
	auditDiffOverlay := &Overlay{
		AutoIncludeTimelineIDs: []uint32{fx.podTimelineID},
		Palette: []ResolvedPaletteColors{
			{
				ID:      "default",
				Main:    "#1976d2",
				OnMain:  "#fafafa",
				Soft:    "rgba(25, 118, 210, 0.13)",
				Surface: "#e8f1fb",
			},
		},
		Excerpts: []ResolvedExcerptCard{
			{
				ID:           "excerpt-1",
				Kind:         ExcerptKindDiff,
				Badge:        "E1",
				PaletteID:    "default",
				TimeMs:       fx.baseTime.Add(12 * time.Minute).UnixMilli(),
				TimeLabel:    "09:12:00Z",
				TimelineID:   fx.podTimelineID,
				ResourcePath: "pod · default/web-1",
				Title:        "Create · Resource exists",
				Subtitle:     "+6 / -0 · 6 diff lines",
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 2, Text: "⋮ 2 diff lines omitted"},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 3,
						Text:       "metadata:",
						Parts:      []LinePart{{Text: "metadata:", Matched: true}},
					},
					{Kind: ExcerptLineKindAdded, NewLineNum: 4, Text: "  name: web-1"},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 5,
						Text:       "status:",
						Parts:      []LinePart{{Text: "status:", Matched: true}},
					},
					{Kind: ExcerptLineKindAdded, NewLineNum: 6, Text: "  phase: Running"},
				},
				LogTimestamp:  "2026-10-01T09:12:00Z",
				InsertID:      "audit-pod-web-1",
				LogID:         podRev0LogID,
				RevisionIndex: &revZero,
			},
		},
	}

	type resolveResult struct {
		Overlay    *Overlay
		DiffBadges []string
		LogBadges  []string
	}

	testCases := []struct {
		name string
		spec Spec
		want resolveResult
	}{
		{
			name: "empty annotations returns nil overlay",
			spec: Spec{InspectionID: fx.inspectionID},
			want: resolveResult{},
		},
		{
			name: "resolves all layers and auto-includes timelines",
			spec: spec,
			want: resolveResult{
				Overlay:    fullOverlay,
				DiffBadges: []string{"E1"},
				LogBadges:  []string{"E2"},
			},
		},
		{
			name: "diff excerpt resolves audit log insertId and timestamp without internal rev prefixes",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					DiffExcerpts: []DiffExcerpt{
						{
							Revision: RevisionRef{TimelineID: fx.podTimelineID, RevisionIndex: 0},
							Paths:    []string{"$.metadata", "$.status"},
						},
					},
				},
			},
			want: resolveResult{
				Overlay:    auditDiffOverlay,
				DiffBadges: []string{"E1"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			specCopy := tc.spec.Clone()
			applied := fx.wb.ResolveFilter(specCopy.Filter)
			gotOverlay, errRes, err := Resolve(ctx, fx.wb, fx.inspectionID, &specCopy, applied)
			if err != nil {
				t.Fatalf("Resolve() unexpected error = %v", err)
			}
			if errRes != nil {
				t.Fatalf("Resolve() unexpected tool error = %+v", errRes)
			}
			got := resolveResult{
				Overlay: gotOverlay,
			}
			if specCopy.Annotations != nil {
				for _, d := range specCopy.Annotations.DiffExcerpts {
					got.DiffBadges = append(got.DiffBadges, d.Badge)
				}
				for _, l := range specCopy.Annotations.LogExcerpts {
					got.LogBadges = append(got.LogBadges, l.Badge)
				}
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolve_ManifestExcerptsAndBadgeSync(t *testing.T) {
	fx := setupResolverFixture(t)
	ctx := context.Background()

	nodeRev0Bounds, err := fx.wb.ResolveRevisionBounds(fx.nodeTimelineID, 0)
	if err != nil {
		t.Fatalf("ResolveRevisionBounds(node, 0) error = %v", err)
	}
	nodeRev1Bounds, err := fx.wb.ResolveRevisionBounds(fx.nodeTimelineID, 1)
	if err != nil {
		t.Fatalf("ResolveRevisionBounds(node, 1) error = %v", err)
	}
	podRev0Bounds, err := fx.wb.ResolveRevisionBounds(fx.podTimelineID, 0)
	if err != nil {
		t.Fatalf("ResolveRevisionBounds(pod, 0) error = %v", err)
	}

	revZero := 0
	revOne := 1
	zoomStart := fx.baseTime.Add(20 * time.Minute)
	zoomEnd := fx.baseTime.Add(40 * time.Minute)
	steadyPointTime := fx.baseTime.Add(28 * time.Minute)
	earlyRefTime := fx.baseTime.Add(21 * time.Minute)
	lateRefTime := fx.baseTime.Add(36 * time.Minute)

	type badgeSyncOutcome struct {
		Overlay        *Overlay
		MarkerBadges   []string
		DiffBadges     []string
		ManifestBadges []string
		LogBadges      []string
	}

	testCases := []struct {
		name string
		spec Spec
		want badgeSyncOutcome
	}{
		{
			name: "clamps revisionIndex manifest excerpt active across viewStart and anchors time-queried manifest excerpt at exact point",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Filter: workbench.Filter{
					StartTime: &zoomStart,
					EndTime:   &zoomEnd,
				},
				Annotations: &Annotations{
					Palette: []PaletteEntry{
						{ID: "before", Color: "slate"},
						{ID: "incident", Color: "amber"},
					},
					ManifestExcerpts: []ManifestExcerpt{
						{
							TimelineID:    fx.nodeTimelineID,
							RevisionIndex: &revZero,
							Badge:         "BEFORE",
							PaletteID:     "before",
							Paths:         []string{"$.status.ready", "$.status.reason"},
							Note:          "Steady Ready=True prior to incident",
						},
						{
							TimelineID: fx.nodeTimelineID,
							Time:       &steadyPointTime,
							Badge:      "DURING",
							PaletteID:  "incident",
							Paths:      []string{"$.status.ready", "$.status.reason"},
							Note:       "Point-in-time state at 09:28Z",
						},
					},
				},
			},
			want: badgeSyncOutcome{
				Overlay: &Overlay{
					AutoIncludeTimelineIDs: []uint32{fx.nodeTimelineID},
					Palette: []ResolvedPaletteColors{
						{
							ID:      "before",
							Main:    "#546e7a",
							OnMain:  "#fafafa",
							Soft:    "rgba(84, 110, 122, 0.13)",
							Surface: "#eef1f2",
						},
						{
							ID:      "incident",
							Main:    "#b45309",
							OnMain:  "#fafafa",
							Soft:    "rgba(180, 83, 9, 0.13)",
							Surface: "#f8eee6",
						},
					},
					Excerpts: []ResolvedExcerptCard{
						{
							ID:           "excerpt-1",
							Kind:         ExcerptKindManifest,
							Badge:        "BEFORE",
							PaletteID:    "before",
							TimeMs:       zoomStart.UnixMilli(),
							TimeLabel:    "09:20:00Z",
							TimelineID:   fx.nodeTimelineID,
							ResourcePath: "node · gke-node-1",
							Title:        "Manifest · Resource exists",
							Subtitle:     "7 lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 lines omitted"},
								{
									Kind:       ExcerptLineKindContext,
									NewLineNum: 6,
									Text:       `  ready: "True"`,
									Parts: []LinePart{
										{Text: "  "},
										{Text: `ready: "True"`, Matched: true},
									},
								},
								{
									Kind:       ExcerptLineKindContext,
									NewLineNum: 7,
									Text:       "  reason: KubeletReady",
									Parts: []LinePart{
										{Text: "  "},
										{Text: "reason: KubeletReady", Matched: true},
									},
								},
							},
							Note:          "Steady Ready=True prior to incident",
							LogTimestamp:  "2026-10-01T09:10:00Z",
							LogID:         nodeRev0Bounds.Revision.LogID,
							RevisionIndex: &revZero,
						},
						{
							ID:           "excerpt-2",
							Kind:         ExcerptKindManifest,
							Badge:        "DURING",
							PaletteID:    "incident",
							TimeMs:       steadyPointTime.UnixMilli(),
							TimeLabel:    "09:28:00Z",
							TimelineID:   fx.nodeTimelineID,
							ResourcePath: "node · gke-node-1",
							Title:        "Manifest · Resource exists",
							Subtitle:     "7 lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 lines omitted"},
								{
									Kind:       ExcerptLineKindContext,
									NewLineNum: 6,
									Text:       `  ready: "False"`,
									Parts: []LinePart{
										{Text: "  "},
										{Text: `ready: "False"`, Matched: true},
									},
								},
								{
									Kind:       ExcerptLineKindContext,
									NewLineNum: 7,
									Text:       "  reason: KubeletNotReady",
									Parts: []LinePart{
										{Text: "  "},
										{Text: "reason: KubeletNotReady", Matched: true},
									},
								},
							},
							Note:          "Point-in-time state at 09:28Z",
							LogTimestamp:  "2026-10-01T09:24:00Z",
							LogID:         nodeRev1Bounds.Revision.LogID,
							RevisionIndex: &revOne,
						},
					},
				},
				ManifestBadges: []string{"BEFORE", "DURING"},
			},
		},
		{
			name: "synchronizes co-anchored badges in both directions, numbers unanchored markers chronologically, and orders tied diff before manifest before log",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Markers: []MarkerAnnotation{
						// Input index 0 is latest (09:36Z) -> receives chronological badge "2".
						{
							Anchor: Anchor{Time: &lateRefTime},
							Label:  "Post-incident check",
							Style:  MarkerStyleReference,
						},
						// Input index 1 is co-anchored with node rev 1 diff/manifest (09:24Z) -> inherits "E2".
						{
							Anchor: Anchor{TimelineID: fx.nodeTimelineID, RevisionIndex: &revOne},
							Label:  "Node NotReady transition",
						},
						// Input index 2 has explicit badge "OOM" at 09:25Z -> co-anchored LogExcerpt inherits "OOM".
						{
							Anchor: Anchor{LogID: fx.oomEventLogID},
							Label:  "App OOMKilled",
							Badge:  "OOM",
						},
						// Input index 3 is earliest (09:21Z) -> receives chronological badge "1".
						{
							Anchor: Anchor{Time: &earlyRefTime},
							Label:  "Traffic surge started",
							Style:  MarkerStyleReference,
						},
					},
					ManifestExcerpts: []ManifestExcerpt{
						{
							TimelineID:    fx.podTimelineID,
							RevisionIndex: &revZero,
							Paths:         []string{"$.status.phase"},
						},
						{
							TimelineID:    fx.nodeTimelineID,
							RevisionIndex: &revOne,
							Paths:         []string{"$.metadata.name"},
						},
					},
					DiffExcerpts: []DiffExcerpt{
						{
							Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1},
							Paths:    []string{"$.status.reason"},
						},
					},
					LogExcerpts: []LogExcerpt{
						{
							LogID: fx.oomEventLogID,
						},
					},
				},
			},
			want: badgeSyncOutcome{
				Overlay: &Overlay{
					AutoIncludeTimelineIDs: []uint32{fx.nodeTimelineID, fx.podTimelineID, fx.appTimelineID},
					Palette: []ResolvedPaletteColors{
						{
							ID:      "default",
							Main:    "#1976d2",
							OnMain:  "#fafafa",
							Soft:    "rgba(25, 118, 210, 0.13)",
							Surface: "#e8f1fb",
						},
						{
							ID:      "default-reference",
							Main:    "#5e35b1",
							OnMain:  "#fafafa",
							Soft:    "rgba(94, 53, 177, 0.13)",
							Surface: "#efebf7",
						},
					},
					Markers: []ResolvedMarker{
						{
							ID:        "marker-1",
							Badge:     "1",
							Label:     "Traffic surge started",
							Style:     MarkerStyleReference,
							PaletteID: "default-reference",
							TimeMs:    earlyRefTime.UnixMilli(),
							TimeLabel: "09:21:00Z",
						},
						{
							ID:            "marker-2",
							Badge:         "E2",
							Label:         "Node NotReady transition",
							Style:         MarkerStylePin,
							PaletteID:     "default",
							TimeMs:        fx.baseTime.Add(24 * time.Minute).UnixMilli(),
							TimeLabel:     "09:24:00Z",
							TimelineID:    fx.nodeTimelineID,
							LogID:         nodeRev1Bounds.Revision.LogID,
							RevisionIndex: &revOne,
						},
						{
							ID:         "marker-3",
							Badge:      "OOM",
							Label:      "App OOMKilled",
							Style:      MarkerStylePin,
							PaletteID:  "default",
							TimeMs:     fx.baseTime.Add(25 * time.Minute).UnixMilli(),
							TimeLabel:  "09:25:00Z",
							TimelineID: fx.appTimelineID,
							LogID:      fx.oomEventLogID,
						},
						{
							ID:        "marker-4",
							Badge:     "2",
							Label:     "Post-incident check",
							Style:     MarkerStyleReference,
							PaletteID: "default-reference",
							TimeMs:    lateRefTime.UnixMilli(),
							TimeLabel: "09:36:00Z",
						},
					},
					Excerpts: []ResolvedExcerptCard{
						{
							ID:           "excerpt-1",
							Kind:         ExcerptKindManifest,
							Badge:        "E1",
							PaletteID:    "default",
							TimeMs:       fx.baseTime.Add(12 * time.Minute).UnixMilli(),
							TimeLabel:    "09:12:00Z",
							TimelineID:   fx.podTimelineID,
							ResourcePath: "pod · default/web-1",
							Title:        "Manifest · Resource exists",
							Subtitle:     "6 lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 lines omitted"},
								{
									Kind:       ExcerptLineKindContext,
									NewLineNum: 6,
									Text:       "  phase: Running",
									Parts: []LinePart{
										{Text: "  "},
										{Text: "phase: Running", Matched: true},
									},
								},
							},
							LogTimestamp:  "2026-10-01T09:12:00Z",
							InsertID:      "audit-pod-web-1",
							LogID:         podRev0Bounds.Revision.LogID,
							RevisionIndex: &revZero,
						},
						{
							ID:           "excerpt-2",
							Kind:         ExcerptKindDiff,
							Badge:        "E2",
							PaletteID:    "default",
							TimeMs:       fx.baseTime.Add(24 * time.Minute).UnixMilli(),
							TimeLabel:    "09:24:00Z",
							TimelineID:   fx.nodeTimelineID,
							ResourcePath: "node · gke-node-1",
							Title:        "Patch · Resource exists",
							Subtitle:     "+2 / -2 · 7 diff lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 diff lines omitted"},
								{
									Kind:       ExcerptLineKindDeleted,
									OldLineNum: 7,
									Text:       "  reason: KubeletReady",
									Parts: []LinePart{
										{Text: "  "},
										{Text: "reason: KubeletReady", Matched: true},
									},
								},
								{
									Kind:       ExcerptLineKindAdded,
									NewLineNum: 7,
									Text:       "  reason: KubeletNotReady",
									Parts: []LinePart{
										{Text: "  "},
										{Text: "reason: KubeletNotReady", Matched: true},
									},
								},
							},
							LogTimestamp:  "2026-10-01T09:24:00Z",
							LogID:         nodeRev1Bounds.Revision.LogID,
							RevisionIndex: &revOne,
						},
						{
							ID:           "excerpt-3",
							Kind:         ExcerptKindManifest,
							Badge:        "E3",
							PaletteID:    "default",
							TimeMs:       fx.baseTime.Add(24 * time.Minute).UnixMilli(),
							TimeLabel:    "09:24:00Z",
							TimelineID:   fx.nodeTimelineID,
							ResourcePath: "node · gke-node-1",
							Title:        "Manifest · Resource exists",
							Subtitle:     "7 lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindOmitted, Omitted: 3, Text: "⋮ 3 lines omitted"},
								{
									Kind:       ExcerptLineKindContext,
									NewLineNum: 4,
									Text:       "  name: gke-node-1",
									Parts: []LinePart{
										{Text: "  "},
										{Text: "name: gke-node-1", Matched: true},
									},
								},
								{Kind: ExcerptLineKindOmitted, Omitted: 3, Text: "⋮ 3 lines omitted"},
							},
							LogTimestamp:  "2026-10-01T09:24:00Z",
							LogID:         nodeRev1Bounds.Revision.LogID,
							RevisionIndex: &revOne,
						},
						{
							ID:           "excerpt-4",
							Kind:         ExcerptKindLog,
							Badge:        "OOM",
							PaletteID:    "default",
							TimeMs:       fx.baseTime.Add(25 * time.Minute).UnixMilli(),
							TimeLabel:    "09:25:00Z",
							TimelineID:   fx.appTimelineID,
							ResourcePath: "web-1/app · container",
							Title:        "k8s event · ERROR",
							Subtitle:     "4 lines",
							Lines: []ExcerptLine{
								{Kind: ExcerptLineKindSummary, Text: "Container app OOMKilled"},
							},
							LogTimestamp: "2026-10-01T09:25:00Z",
							InsertID:     "evt-oom",
							LogID:        fx.oomEventLogID,
						},
					},
				},
				// Back-filled on spec.Annotations in input slice order.
				MarkerBadges:   []string{"2", "E2", "OOM", "1"},
				DiffBadges:     []string{"E2"},
				ManifestBadges: []string{"E1", "E3"},
				LogBadges:      []string{"OOM"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			specCopy := tc.spec.Clone()
			applied := fx.wb.ResolveFilter(specCopy.Filter)
			gotOverlay, errRes, err := Resolve(ctx, fx.wb, fx.inspectionID, &specCopy, applied)
			if err != nil {
				t.Fatalf("Resolve() unexpected error = %v", err)
			}
			if errRes != nil {
				t.Fatalf("Resolve() unexpected tool error = %+v", errRes)
			}
			got := badgeSyncOutcome{Overlay: gotOverlay}
			if specCopy.Annotations != nil {
				for _, m := range specCopy.Annotations.Markers {
					got.MarkerBadges = append(got.MarkerBadges, m.Badge)
				}
				for _, d := range specCopy.Annotations.DiffExcerpts {
					got.DiffBadges = append(got.DiffBadges, d.Badge)
				}
				for _, me := range specCopy.Annotations.ManifestExcerpts {
					got.ManifestBadges = append(got.ManifestBadges, me.Badge)
				}
				for _, l := range specCopy.Annotations.LogExcerpts {
					got.LogBadges = append(got.LogBadges, l.Badge)
				}
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolve_Errors(t *testing.T) {
	fx := setupResolverFixture(t)
	ctx := context.Background()

	narrowStart := fx.baseTime.Add(30 * time.Minute)
	narrowEnd := fx.baseTime.Add(40 * time.Minute)

	testCases := []struct {
		name     string
		spec     Spec
		wantCode string
	}{
		{
			name: "anchor with neither logId nor revision nor time returns INVALID_ANCHOR",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Markers: []MarkerAnnotation{{Label: "bad"}},
				},
			},
			wantCode: "INVALID_ANCHOR",
		},
		{
			name: "anchor outside filter.startTime..endTime returns ANCHOR_OUT_OF_VIEWPORT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Filter: workbench.Filter{
					StartTime: &narrowStart,
					EndTime:   &narrowEnd,
				},
				Annotations: &Annotations{
					Markers: []MarkerAnnotation{
						{Label: "OOM", Anchor: Anchor{LogID: fx.oomEventLogID}},
					},
				},
			},
			wantCode: "ANCHOR_OUT_OF_VIEWPORT",
		},
		{
			name: "span end before start returns INVALID_ANCHOR",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Spans: []SpanAnnotation{
						{
							Label: "reversed",
							Start: Anchor{Time: &narrowEnd},
							End:   Anchor{Time: &narrowStart},
						},
					},
				},
			},
			wantCode: "INVALID_ANCHOR",
		},
		{
			name: "invalid CEL in rowHighlights.timelineQuery returns INVALID_CEL",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					RowHighlights: []RowHighlight{
						{TimelineQuery: "invalid &&& cel"},
					},
				},
			},
			wantCode: "INVALID_CEL",
		},
		{
			name: "omitted diffExcerpt paths returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					DiffExcerpts: []DiffExcerpt{
						{
							Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1},
						},
					},
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "invalid JSONPath syntax in diffExcerpt returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					DiffExcerpts: []DiffExcerpt{
						{
							Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1},
							Paths:    []string{"$.status["},
						},
					},
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "unmatched diff JSONPath returns SLICE_QUERY_NOT_FOUND with guidance",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					DiffExcerpts: []DiffExcerpt{
						{
							Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1},
							Paths:    []string{"$.status.noSuchCondition"},
						},
					},
				},
			},
			wantCode: "SLICE_QUERY_NOT_FOUND",
		},
		{
			name: "unmatched log slice query returns SLICE_QUERY_NOT_FOUND with summary",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					LogExcerpts: []LogExcerpt{
						{
							LogID:   fx.oomEventLogID,
							Windows: []SliceWindow{{Query: "NoSuchField"}},
						},
					},
				},
			},
			wantCode: "SLICE_QUERY_NOT_FOUND",
		},
		{
			name: "empty slice window path and query returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					LogExcerpts: []LogExcerpt{
						{
							LogID:   fx.oomEventLogID,
							Windows: []SliceWindow{{}},
						},
					},
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "unknown logId in logExcerpt returns LOG_NOT_FOUND",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					LogExcerpts: []LogExcerpt{
						{
							LogID: 999999,
						},
					},
				},
			},
			wantCode: "LOG_NOT_FOUND",
		},
		{
			name: "exceeding total excerpt card cap returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					LogExcerpts: make([]LogExcerpt, 7),
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "exceeding total excerpt card cap across diff, manifest, and log excerpts returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					DiffExcerpts:     make([]DiffExcerpt, 2),
					ManifestExcerpts: make([]ManifestExcerpt, 3),
					LogExcerpts:      make([]LogExcerpt, 2),
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "manifestExcerpt with neither revisionIndex nor time returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					ManifestExcerpts: []ManifestExcerpt{
						{TimelineID: fx.nodeTimelineID, Paths: []string{"$.status"}},
					},
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "manifestExcerpt with omitted paths returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					ManifestExcerpts: []ManifestExcerpt{
						{TimelineID: fx.nodeTimelineID, RevisionIndex: intRef(0)},
					},
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "manifestExcerpt with both revisionIndex and time returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					ManifestExcerpts: []ManifestExcerpt{
						{
							TimelineID:    fx.nodeTimelineID,
							RevisionIndex: intRef(0),
							Time:          &narrowStart,
							Paths:         []string{"$.status"},
						},
					},
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "manifestExcerpt time outside viewport returns ANCHOR_OUT_OF_VIEWPORT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Filter: workbench.Filter{
					StartTime: &narrowStart,
					EndTime:   &narrowEnd,
				},
				Annotations: &Annotations{
					ManifestExcerpts: []ManifestExcerpt{
						{
							TimelineID: fx.nodeTimelineID,
							Time:       timeRef(fx.baseTime.Add(25 * time.Minute)),
							Paths:      []string{"$.status"},
						},
					},
				},
			},
			wantCode: "ANCHOR_OUT_OF_VIEWPORT",
		},
		{
			name: "manifestExcerpt revisionIndex whose active interval ended before viewStart returns ANCHOR_OUT_OF_VIEWPORT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Filter: workbench.Filter{
					StartTime: &narrowStart,
					EndTime:   &narrowEnd,
				},
				Annotations: &Annotations{
					ManifestExcerpts: []ManifestExcerpt{
						{
							TimelineID:    fx.nodeTimelineID,
							RevisionIndex: intRef(0), // active [09:10, 09:24), ended before 09:30
							Paths:         []string{"$.status"},
						},
					},
				},
			},
			wantCode: "ANCHOR_OUT_OF_VIEWPORT",
		},
		{
			name: "manifestExcerpt revisionIndex starting after viewEnd returns ANCHOR_OUT_OF_VIEWPORT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Filter: workbench.Filter{
					StartTime: timeRef(fx.baseTime.Add(15 * time.Minute)),
					EndTime:   timeRef(fx.baseTime.Add(30 * time.Minute)),
				},
				Annotations: &Annotations{
					ManifestExcerpts: []ManifestExcerpt{
						{
							TimelineID:    fx.nodeTimelineID,
							RevisionIndex: intRef(2), // starts at 09:34 > 09:30
							Paths:         []string{"$.status"},
						},
					},
				},
			},
			wantCode: "ANCHOR_OUT_OF_VIEWPORT",
		},
		{
			name: "unmatched manifest JSONPath returns SLICE_QUERY_NOT_FOUND",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					ManifestExcerpts: []ManifestExcerpt{
						{
							TimelineID:    fx.nodeTimelineID,
							RevisionIndex: intRef(0),
							Paths:         []string{"$.status.noSuchManifestField"},
						},
					},
				},
			},
			wantCode: "SLICE_QUERY_NOT_FOUND",
		},
		{
			name: "duplicate manual excerpt badges return INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					DiffExcerpts: []DiffExcerpt{
						{
							Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1},
							Paths:    []string{"$.status"},
							Badge:    "DUP",
						},
					},
					LogExcerpts: []LogExcerpt{
						{
							LogID: fx.oomEventLogID,
							Badge: "DUP",
						},
					},
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "out-of-order manual E<n> excerpt badges reversing chronological order return INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					DiffExcerpts: []DiffExcerpt{
						{
							Revision: RevisionRef{TimelineID: fx.nodeTimelineID, RevisionIndex: 1}, // 09:24Z
							Paths:    []string{"$.status"},
							Badge:    "E4",
						},
					},
					LogExcerpts: []LogExcerpt{
						{
							LogID: fx.oomEventLogID, // 09:25Z
							Badge: "E1",
						},
					},
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "unregistered paletteId returns UNKNOWN_PALETTE_ID",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Palette: []PaletteEntry{{ID: "victim", Color: "red"}},
					Markers: []MarkerAnnotation{
						{Label: "OOM", Anchor: Anchor{LogID: fx.oomEventLogID}, PaletteID: "missing-id"},
					},
				},
			},
			wantCode: "UNKNOWN_PALETTE_ID",
		},
		{
			name: "exceeding max markers returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Markers: make([]MarkerAnnotation, 13),
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			specCopy := tc.spec.Clone()
			applied := fx.wb.ResolveFilter(specCopy.Filter)
			_, errRes, err := Resolve(ctx, fx.wb, fx.inspectionID, &specCopy, applied)
			if err != nil {
				t.Fatalf("Resolve() unexpected error = %v", err)
			}
			if errRes == nil || !errRes.IsError || len(errRes.Content) == 0 {
				t.Fatalf("Resolve() errRes = %+v, want error result with code %s", errRes, tc.wantCode)
			}
			textContent, ok := errRes.Content[0].(*mcpsdk.TextContent)
			if !ok {
				t.Fatalf("errRes.Content[0] = %T, want *mcpsdk.TextContent", errRes.Content[0])
			}
			wantHeader := "Error: " + tc.wantCode
			if !strings.HasPrefix(textContent.Text, wantHeader) {
				t.Errorf("error text = %q, want prefix %q", textContent.Text, wantHeader)
			}
		})
	}
}
