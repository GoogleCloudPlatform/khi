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
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

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
	unlinkedLogID  uint32
}

type allLayersResolveResult struct {
	Overlay    *Overlay
	DiffBadges []string
	LogBadges  []string
}

type fixtureRevisionSpec struct {
	offset       time.Duration
	path         *khifilev6.TimelinePath
	logYAML      string
	resourceYAML string
	summary      string
	state        *pb.RevisionState
	verb         *pb.Verb
	severity     *pb.Severity
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
				return struct{}{}, populateResolverFixtureData(ctx, baseTime)
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

	return buildResolverFixture(t, wb, inspectionID, baseTime)
}

func populateResolverFixtureData(ctx context.Context, baseTime time.Time) error {
	metadataSet := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
	if header, found := typedmap.Get(metadataSet, inspectionmetadata.HeaderMetadataKey); found && header != nil {
		header.StartTimeUnixSeconds = baseTime.Unix()
		header.EndTimeUnixSeconds = baseTime.Add(1 * time.Hour).Unix()
	}

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

	revs := []fixtureRevisionSpec{
		{10 * time.Minute, nodePath, nodeRev0, nodeRev0, "Node Ready=True", k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbCreate, inspectioncore.SeverityInfo},
		{24 * time.Minute, nodePath, nodeRev1, nodeRev1, "Node Ready=False", k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbPatch, inspectioncore.SeverityWarning},
		{34 * time.Minute, nodePath, nodeRev2, nodeRev2, "Node recovered", k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbPatch, inspectioncore.SeverityInfo},
		{12 * time.Minute, podPath, podAudit0, podRev0, "Pod web-1 running", k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbCreate, inspectioncore.SeverityInfo},
	}
	for _, r := range revs {
		if err := recordFixtureRevision(ctx, baseTime.Add(r.offset), r); err != nil {
			return err
		}
	}
	if err := recordFixtureEventLog(ctx, baseTime.Add(25*time.Minute), podPath, appPath); err != nil {
		return err
	}
	return recordFixtureUnlinkedLog(ctx, baseTime.Add(26*time.Minute))
}

func recordFixtureRevision(ctx context.Context, ts time.Time, r fixtureRevisionSpec) error {
	idGen := khictx.MustGetValue(ctx, inspectioncore.IDGenerator)
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	logBodyNode, err := structured.FromYAML(r.logYAML)
	if err != nil {
		return err
	}
	resourceBodyNode, err := structured.FromYAML(r.resourceYAML)
	if err != nil {
		return err
	}
	l := log.NewLogWithTimestamp(idGen, structured.NewNodeReader(logBodyNode), ts)
	logCS, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return err
	}
	logCS.SetTimestamp(ts)
	logCS.SetSeverity(r.severity)
	logCS.SetLogType(k8saudit.LogTypeAudit)
	logCS.SetSummary(r.summary)
	if err := logCS.Flush(builder.LogAccumulator); err != nil {
		return err
	}

	tlCS := khifilev6.NewTimelineChangeSet(l)
	defer tlCS.Release()
	tlCS.AddRevision(r.path, &khifilev6.StagingRevision{
		ChangedTime:  ts,
		ResourceBody: resourceBodyNode,
		Principal:    "kubelet",
		VerbType:     r.verb,
		StateType:    r.state,
	})
	return tlCS.Flush(builder.TimelineAccumulator, builder.LogAccumulator)
}

func recordFixtureEventLog(ctx context.Context, oomTime time.Time, podPath, appPath *khifilev6.TimelinePath) error {
	idGen := khictx.MustGetValue(ctx, inspectioncore.IDGenerator)
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	logNode, err := structured.FromYAML(`insertId: evt-oom
message: "OOMKilled container app\nexitCode: 137"
`)
	if err != nil {
		return err
	}
	l := log.NewLogWithTimestamp(idGen, structured.NewNodeReader(logNode), oomTime)
	logCS, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return err
	}
	logCS.SetTimestamp(oomTime)
	logCS.SetSeverity(inspectioncore.SeverityError)
	logCS.SetLogType(k8saudit.LogTypeEvent)
	logCS.SetSummary("Container app OOMKilled")
	if err := logCS.Flush(builder.LogAccumulator); err != nil {
		return err
	}
	tlCS := khifilev6.NewTimelineChangeSet(l)
	defer tlCS.Release()
	tlCS.AddEvent(podPath)
	tlCS.AddEvent(appPath)
	return tlCS.Flush(builder.TimelineAccumulator, builder.LogAccumulator)
}

func recordFixtureUnlinkedLog(ctx context.Context, unlinkedTime time.Time) error {
	idGen := khictx.MustGetValue(ctx, inspectioncore.IDGenerator)
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	unlinkedNode, err := structured.FromYAML(`message: unlinked diagnostic`)
	if err != nil {
		return err
	}
	ul := log.NewLogWithTimestamp(idGen, structured.NewNodeReader(unlinkedNode), unlinkedTime)
	ulCS, err := khifilev6.NewLogChangeSet(ul)
	if err != nil {
		return err
	}
	ulCS.SetTimestamp(unlinkedTime)
	ulCS.SetSeverity(inspectioncore.SeverityInfo)
	ulCS.SetLogType(k8saudit.LogTypeEvent)
	ulCS.SetSummary("Unlinked log entry")
	return ulCS.Flush(builder.LogAccumulator)
}

func buildResolverFixture(t *testing.T, wb *workbench.Workbench, inspectionID string, baseTime time.Time) resolverFixture {
	t.Helper()

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
	var oomLogID, unlinkedLogID uint32
	for i := range idx.Logs {
		switch idx.InternPool.ResolveStringFromID(idx.Logs[i].SummaryStringID) {
		case "Container app OOMKilled":
			oomLogID = idx.Logs[i].ID
		case "Unlinked log entry":
			unlinkedLogID = idx.Logs[i].ID
		}
	}
	if nodeID == 0 || podID == 0 || appID == 0 || oomLogID == 0 || unlinkedLogID == 0 {
		t.Fatalf("missing fixture IDs: node=%d pod=%d app=%d oomLog=%d unlinked=%d", nodeID, podID, appID, oomLogID, unlinkedLogID)
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
		unlinkedLogID:  unlinkedLogID,
	}
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

	fullOverlay := &Overlay{
		AutoIncludeTimelineIDs: []uint32{fx.nodeTimelineID, fx.podTimelineID, fx.appTimelineID},
		Palette: []ResolvedPaletteColors{
			{ID: "trigger", Main: "#b45309", OnMain: "#fafafa", Soft: "rgba(180, 83, 9, 0.13)", Surface: "#f8eee6"},
			{ID: "victim", Main: "#d32f2f", OnMain: "#fafafa", Soft: "rgba(211, 47, 47, 0.13)", Surface: "#fbeaea"},
			{ID: "default-reference", Main: "#5e35b1", OnMain: "#fafafa", Soft: "rgba(94, 53, 177, 0.13)", Surface: "#efebf7"},
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
				ID:             "span-1",
				Label:          "Node NotReady window",
				StartTimeMs:    fx.baseTime.Add(24 * time.Minute).UnixMilli(),
				EndTimeMs:      fx.baseTime.Add(34 * time.Minute).UnixMilli(),
				DurationLabel:  "10m",
				Pattern:        SpanPatternHatched,
				LabelPlacement: SpanLabelPlacementAuto,
				PaletteID:      "trigger",
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

	testCases := []struct {
		name string
		spec Spec
		want allLayersResolveResult
	}{
		{
			name: "empty annotations returns nil overlay",
			spec: Spec{InspectionID: fx.inspectionID},
			want: allLayersResolveResult{},
		},
		{
			name: "resolves all layers and auto-includes timelines",
			spec: spec,
			want: allLayersResolveResult{
				Overlay:    fullOverlay,
				DiffBadges: []string{"E1"},
				LogBadges:  []string{"E2"},
			},
		},
		{
			name: "accepts exact maxAnnotationRunes marker label",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Markers: []MarkerAnnotation{
						{
							Anchor: Anchor{LogID: fx.oomEventLogID},
							Label:  strings.Repeat("x", maxAnnotationRunes),
							Badge:  "MAX",
						},
					},
				},
			},
			want: allLayersResolveResult{
				Overlay: &Overlay{
					AutoIncludeTimelineIDs: []uint32{fx.appTimelineID},
					Palette: []ResolvedPaletteColors{
						{
							ID:      "default",
							Main:    "#1976d2",
							OnMain:  "#fafafa",
							Soft:    "rgba(25, 118, 210, 0.13)",
							Surface: "#e8f1fb",
						},
					},
					Markers: []ResolvedMarker{
						{
							ID:         "marker-1",
							Badge:      "MAX",
							Label:      strings.Repeat("x", maxAnnotationRunes),
							Style:      MarkerStylePin,
							PaletteID:  "default",
							TimeMs:     fx.baseTime.Add(25 * time.Minute).UnixMilli(),
							TimeLabel:  "09:25:00Z",
							TimelineID: fx.appTimelineID,
							LogID:      fx.oomEventLogID,
						},
					},
				},
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
			got := collectAllLayersResult(gotOverlay, specCopy.Annotations)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func collectAllLayersResult(gotOverlay *Overlay, ann *Annotations) allLayersResolveResult {
	out := allLayersResolveResult{Overlay: gotOverlay}
	if ann == nil {
		return out
	}
	for _, d := range ann.DiffExcerpts {
		out.DiffBadges = append(out.DiffBadges, d.Badge)
	}
	for _, l := range ann.LogExcerpts {
		out.LogBadges = append(out.LogBadges, l.Badge)
	}
	return out
}

func TestResolve_Errors(t *testing.T) {
	fx := setupResolverFixture(t)
	ctx := context.Background()

	narrowStart := fx.baseTime.Add(30 * time.Minute)
	narrowEnd := fx.baseTime.Add(40 * time.Minute)

	testCases := []struct {
		name       string
		spec       Spec
		wantCode   string
		wantSubstr string
	}{
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
			name: "duplicate paletteId returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{Palette: []PaletteEntry{{ID: "dup", Color: "red"}, {ID: "dup", Color: "blue"}}},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "markers exceeding cap returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations:  &Annotations{Markers: make([]MarkerAnnotation, maxSceneMarkers+1)},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name: "unregistered paletteId returns UNKNOWN_PALETTE_ID",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Palette:    []PaletteEntry{{ID: "victim", Color: "red"}},
					Connectors: []ConnectorAnnotation{{From: Anchor{LogID: fx.oomEventLogID}, To: Anchor{LogID: fx.oomEventLogID}, PaletteID: "missing-id"}},
				},
			},
			wantCode: "UNKNOWN_PALETTE_ID",
		},
		{
			name: "unsupported callout placement returns INVALID_ARGUMENT",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Callouts: []CalloutAnnotation{{Label: "bad", Placement: "bottom", Anchor: Anchor{LogID: fx.oomEventLogID}}},
				},
			},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name:     "exceeding max palette entries returns INVALID_ARGUMENT",
			spec:     Spec{InspectionID: fx.inspectionID, Annotations: &Annotations{Palette: make([]PaletteEntry, 17)}},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name:     "unsupported connector style returns INVALID_ARGUMENT",
			spec:     Spec{InspectionID: fx.inspectionID, Annotations: &Annotations{Connectors: []ConnectorAnnotation{{Style: "invalid"}}}},
			wantCode: "INVALID_ARGUMENT",
		},
		{
			name:     "empty callout label returns INVALID_ARGUMENT",
			spec:     Spec{InspectionID: fx.inspectionID, Annotations: &Annotations{Callouts: []CalloutAnnotation{{}}}},
			wantCode: "INVALID_ARGUMENT",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertResolveError(t, ctx, fx, tc.spec, tc.wantCode, tc.wantSubstr)
		})
	}
}

func assertResolveError(t *testing.T, ctx context.Context, fx resolverFixture, spec Spec, wantCode, wantSubstr string) {
	t.Helper()
	specCopy := spec.Clone()
	applied := fx.wb.ResolveFilter(specCopy.Filter)
	_, errRes, err := Resolve(ctx, fx.wb, fx.inspectionID, &specCopy, applied)
	if err != nil {
		t.Fatalf("Resolve() unexpected error = %v", err)
	}
	if errRes == nil || !errRes.IsError || len(errRes.Content) == 0 {
		t.Fatalf("Resolve() errRes = %+v, want error result with code %s", errRes, wantCode)
	}
	textContent, ok := errRes.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("errRes.Content[0] = %T, want *mcpsdk.TextContent", errRes.Content[0])
	}
	wantHeader := "Error: " + wantCode
	if !strings.HasPrefix(textContent.Text, wantHeader) {
		t.Errorf("error text = %q, want prefix %q", textContent.Text, wantHeader)
	}
	if wantSubstr != "" && !strings.Contains(textContent.Text, wantSubstr) {
		t.Errorf("error text = %q, want substring %q", textContent.Text, wantSubstr)
	}
}

func TestValidateAnnotationCounts(t *testing.T) {
	testCases := []struct {
		name    string
		ann     *Annotations
		wantErr string
	}{
		{
			name: "accepts exact maximum counts across all annotation layers",
			ann: &Annotations{
				Markers:          make([]MarkerAnnotation, maxSceneMarkers),
				Spans:            make([]SpanAnnotation, maxSceneSpans),
				DiffExcerpts:     make([]DiffExcerpt, 2),
				ManifestExcerpts: make([]ManifestExcerpt, 2),
				LogExcerpts:      make([]LogExcerpt, 2),
				Connectors:       make([]ConnectorAnnotation, maxSceneConnectors),
				Callouts:         make([]CalloutAnnotation, maxSceneCallouts),
			},
		},
		{
			name:    "rejects markers over cap",
			ann:     &Annotations{Markers: make([]MarkerAnnotation, maxSceneMarkers+1)},
			wantErr: "`annotations.markers` exceeds maximum of 12 items (got 13).",
		},
		{
			name:    "rejects spans over cap",
			ann:     &Annotations{Spans: make([]SpanAnnotation, maxSceneSpans+1)},
			wantErr: "`annotations.spans` exceeds maximum of 6 items (got 7).",
		},
		{
			name: "rejects combined excerpts over cap",
			ann: &Annotations{
				DiffExcerpts:     make([]DiffExcerpt, 2),
				ManifestExcerpts: make([]ManifestExcerpt, 3),
				LogExcerpts:      make([]LogExcerpt, 2),
			},
			wantErr: "Total `diffExcerpts` + `manifestExcerpts` + `logExcerpts` exceeds maximum of 6 cards (got 7).",
		},
		{
			name:    "rejects diffExcerpts alone over cap",
			ann:     &Annotations{DiffExcerpts: make([]DiffExcerpt, maxSceneExcerpts+1)},
			wantErr: "Total `diffExcerpts` + `manifestExcerpts` + `logExcerpts` exceeds maximum of 6 cards (got 7).",
		},
		{
			name:    "rejects manifestExcerpts alone over cap",
			ann:     &Annotations{ManifestExcerpts: make([]ManifestExcerpt, maxSceneExcerpts+1)},
			wantErr: "Total `diffExcerpts` + `manifestExcerpts` + `logExcerpts` exceeds maximum of 6 cards (got 7).",
		},
		{
			name:    "rejects logExcerpts alone over cap",
			ann:     &Annotations{LogExcerpts: make([]LogExcerpt, maxSceneExcerpts+1)},
			wantErr: "Total `diffExcerpts` + `manifestExcerpts` + `logExcerpts` exceeds maximum of 6 cards (got 7).",
		},
		{
			name:    "rejects connectors over cap",
			ann:     &Annotations{Connectors: make([]ConnectorAnnotation, maxSceneConnectors+1)},
			wantErr: "`annotations.connectors` exceeds maximum of 8 items (got 9).",
		},
		{
			name:    "rejects callouts over cap",
			ann:     &Annotations{Callouts: make([]CalloutAnnotation, maxSceneCallouts+1)},
			wantErr: "`annotations.callouts` exceeds maximum of 8 items (got 9).",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errRes := validateAnnotationCounts(tc.ann)
			if tc.wantErr == "" {
				if errRes != nil {
					t.Fatalf("validateAnnotationCounts() unexpected error = %+v", errRes)
				}
				return
			}
			if errRes == nil || len(errRes.Content) == 0 {
				t.Fatalf("validateAnnotationCounts() = %+v, want error", errRes)
			}
			gotText := errRes.Content[0].(*mcpsdk.TextContent).Text
			if !strings.Contains(gotText, tc.wantErr) {
				t.Errorf("error text = %q, want substring %q", gotText, tc.wantErr)
			}
		})
	}
}
