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

				const nodeRev0 = "apiVersion: v1\nkind: Node\nmetadata:\n  name: gke-node-1\nstatus:\n  ready: \"True\"\n  reason: KubeletReady\n"
				const nodeRev1 = "apiVersion: v1\nkind: Node\nmetadata:\n  name: gke-node-1\nstatus:\n  ready: \"False\"\n  reason: KubeletNotReady\n"
				const nodeRev2 = "apiVersion: v1\nkind: Node\nmetadata:\n  name: gke-node-1\nstatus:\n  ready: \"True\"\n  reason: KubeletReady\n"
				const podAudit0 = "insertId: audit-pod-web-1\nprotoPayload:\n  methodName: io.k8s.core.v1.pods.create\n"
				const podRev0 = "apiVersion: v1\nkind: Pod\nmetadata:\n  name: web-1\nstatus:\n  phase: Running\n"

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
				logNode, err := structured.FromYAML("insertId: evt-oom\nmessage: \"OOMKilled container app\\nexitCode: 137\"\n")
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
		},
		DimUnlistedRows: true,
		Markers: []ResolvedMarker{
			{
				ID:        "marker-1",
				Badge:     "ALERT",
				Label:     "PagerDuty alert fired",
				Style:     MarkerStyleReference,
				PaletteID: "",
				TimeMs:    alertTime.UnixMilli(),
				TimeLabel: "09:24:30Z",
			},
			{
				ID:         "marker-2",
				Badge:      "1",
				Label:      "Container OOMKilled",
				Style:      MarkerStylePin,
				PaletteID:  "",
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
				PaletteID:  "",
				TimeMs:     fx.baseTime.Add(25 * time.Minute).UnixMilli(),
				TimelineID: fx.appTimelineID,
				LogID:      fx.oomEventLogID,
			},
		},
	}

	testCases := []struct {
		name        string
		spec        Spec
		wantOverlay *Overlay
	}{
		{
			name:        "empty annotations returns nil overlay",
			spec:        Spec{InspectionID: fx.inspectionID},
			wantOverlay: nil,
		},
		{
			name:        "resolves all layers and auto-includes timelines",
			spec:        spec,
			wantOverlay: fullOverlay,
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
			if diff := cmp.Diff(tc.wantOverlay, gotOverlay); diff != "" {
				t.Errorf("Resolve() overlay mismatch (-want +got):\n%s", diff)
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
			name: "anchor with both logId and time returns INVALID_ANCHOR",
			spec: Spec{
				InspectionID: fx.inspectionID,
				Annotations: &Annotations{
					Markers: []MarkerAnnotation{
						{Label: "both", Anchor: Anchor{LogID: fx.oomEventLogID, Time: &fx.baseTime}},
					},
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
