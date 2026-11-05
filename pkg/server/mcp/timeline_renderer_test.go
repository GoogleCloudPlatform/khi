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

package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image/png"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/server"
	serverapiv1 "github.com/GoogleCloudPlatform/khi/pkg/server/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/server/cdp"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/inspectionserver"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/net/websocket"
)

type mockCDPRequest struct {
	ID     int            `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

type recordedViewport struct {
	Width             float64
	Height            float64
	DeviceScaleFactor float64
}

// fakeBridgePollStep defines one scripted outcome for a bridge readiness poll.
//
// Evaluates disconnect first, then protocolErrCode/protocolErrMessage, and finally ready.
type fakeBridgePollStep struct {
	ready              bool
	protocolErrCode    int
	protocolErrMessage string
	disconnect         bool
}

// fakeApplyStep defines one scripted outcome for an applyAndMeasure evaluation.
//
// Returns a JS exception when exceptionText is non-empty, or measure otherwise.
type fakeApplyStep struct {
	measure       timelineMeasurePayload
	exceptionText string
}

// fakeTimelineCDPConfig configures scripted step queues for fakeTimelineCDPServer.
//
// Once bridgePolls is exhausted, readiness checks default to ready=true; once applySteps is exhausted, extra applyAndMeasure calls fail the test.
type fakeTimelineCDPConfig struct {
	bridgePolls   []fakeBridgePollStep
	applySteps    []fakeApplyStep
	screenshotPNG []byte
}

type fakeCDPSessionSnapshot struct {
	NavigatedURLs        []string
	BridgePollCount      int
	Viewports            []recordedViewport
	RemainingBridgePolls int
	RemainingApplySteps  int
}

// fakeTimelineCDPServer simulates the /render/mcp-timeline CDP protocol from declarative step queues.
type fakeTimelineCDPServer struct {
	t               *testing.T
	mu              sync.Mutex
	bridgePolls     []fakeBridgePollStep
	applySteps      []fakeApplyStep
	screenshotPNG   []byte
	navigatedURLs   []string
	bridgePollCount int
	viewports       []recordedViewport
	renderer        *ChromeTimelineImageRenderer
}

func newFakeTimelineCDPServer(t *testing.T, cfg fakeTimelineCDPConfig) *fakeTimelineCDPServer {
	t.Helper()
	srv := &fakeTimelineCDPServer{
		t:             t,
		bridgePolls:   slices.Clone(cfg.bridgePolls),
		applySteps:    slices.Clone(cfg.applySteps),
		screenshotPNG: slices.Clone(cfg.screenshotPNG),
	}
	wsServer := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		for {
			var raw string
			if err := websocket.Message.Receive(ws, &raw); err != nil {
				return
			}
			var req mockCDPRequest
			if err := json.Unmarshal([]byte(raw), &req); err != nil {
				srv.t.Errorf("failed to unmarshal CDP request %q: %v", raw, err)
				return
			}
			resp, keepOpen := srv.handleRequest(req)
			if !keepOpen {
				_ = ws.Close()
				return
			}
			respBytes, _ := json.Marshal(resp)
			_ = websocket.Message.Send(ws, string(respBytes))
		}
	}))
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	browser := cdp.NewBrowserFromPageWSURL(wsURL)
	srv.renderer = NewChromeTimelineImageRenderer(browser, "http://127.0.0.1:8080")
	t.Cleanup(func() {
		_ = srv.renderer.Close()
		_ = browser.Close()
		wsServer.Close()
	})
	return srv
}

func (s *fakeTimelineCDPServer) snapshot() fakeCDPSessionSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fakeCDPSessionSnapshot{
		NavigatedURLs:        slices.Clone(s.navigatedURLs),
		BridgePollCount:      s.bridgePollCount,
		Viewports:            slices.Clone(s.viewports),
		RemainingBridgePolls: len(s.bridgePolls),
		RemainingApplySteps:  len(s.applySteps),
	}
}

func (s *fakeTimelineCDPServer) handleRequest(req mockCDPRequest) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch req.Method {
	case "Page.enable", "Runtime.enable":
		return cdpResultResponse(req.ID, map[string]any{}), true
	case "Page.navigate":
		if u, ok := req.Params["url"].(string); ok {
			s.navigatedURLs = append(s.navigatedURLs, u)
		}
		return cdpResultResponse(req.ID, map[string]any{"frameId": "frame-1"}), true
	case "Emulation.setDeviceMetricsOverride":
		w, _ := req.Params["width"].(float64)
		h, _ := req.Params["height"].(float64)
		dsf, _ := req.Params["deviceScaleFactor"].(float64)
		s.viewports = append(s.viewports, recordedViewport{
			Width:             w,
			Height:            h,
			DeviceScaleFactor: dsf,
		})
		return cdpResultResponse(req.ID, map[string]any{}), true
	case "Page.captureScreenshot":
		return cdpResultResponse(req.ID, map[string]any{
			"data": base64.StdEncoding.EncodeToString(s.screenshotPNG),
		}), true
	case "Runtime.evaluate":
		return s.handleEvaluateLocked(req)
	default:
		s.t.Errorf("unexpected CDP method %q", req.Method)
		return cdpErrorResponse(req.ID, -32601, "unexpected CDP method: "+req.Method), true
	}
}

func (s *fakeTimelineCDPServer) handleEvaluateLocked(req mockCDPRequest) (map[string]any, bool) {
	expr, _ := req.Params["expression"].(string)
	switch {
	case expr == "typeof window.__khiCDPBridge?.timelineRenderer !== 'undefined'":
		s.bridgePollCount++
		if len(s.bridgePolls) == 0 {
			return cdpEvalValueResponse(req.ID, "boolean", true), true
		}
		step := s.bridgePolls[0]
		s.bridgePolls = s.bridgePolls[1:]
		if step.disconnect {
			return nil, false
		}
		if step.protocolErrCode != 0 || step.protocolErrMessage != "" {
			return cdpErrorResponse(req.ID, step.protocolErrCode, step.protocolErrMessage), true
		}
		return cdpEvalValueResponse(req.ID, "boolean", step.ready), true
	case strings.HasPrefix(expr, "window.__khiCDPBridge.timelineRenderer.applyAndMeasure("):
		if len(s.applySteps) == 0 {
			s.t.Errorf("unexpected applyAndMeasure call with no remaining fakeApplyStep")
			return cdpErrorResponse(req.ID, -32000, "unexpected applyAndMeasure call with no remaining fakeApplyStep"), true
		}
		step := s.applySteps[0]
		s.applySteps = s.applySteps[1:]
		if step.exceptionText != "" {
			return cdpResultResponse(req.ID, map[string]any{
				"result":           map[string]any{"type": "object"},
				"exceptionDetails": map[string]any{"text": step.exceptionText},
			}), true
		}
		return cdpEvalValueResponse(req.ID, "object", step.measure), true
	case expr == "window.__khiCDPBridge.timelineRenderer.waitForStableFrame()":
		return cdpResultResponse(req.ID, map[string]any{
			"result": map[string]any{"type": "undefined"},
		}), true
	default:
		s.t.Errorf("unexpected Runtime.evaluate expression %q", expr)
		return cdpErrorResponse(req.ID, -32601, "unexpected Runtime.evaluate expression: "+expr), true
	}
}

func cdpResultResponse(id int, result any) map[string]any {
	return map[string]any{"id": id, "result": result}
}

func cdpEvalValueResponse(id int, jsType string, value any) map[string]any {
	return cdpResultResponse(id, map[string]any{
		"result": map[string]any{"type": jsType, "value": value},
	})
}

func cdpErrorResponse(id int, code int, message string) map[string]any {
	return map[string]any{
		"id":    id,
		"error": map[string]any{"code": code, "message": message},
	}
}

func TestTimelineDeviceScaleFactor(t *testing.T) {
	testCases := []struct {
		name   string
		width  int
		height int
		want   int
	}{
		{
			name:   "default 1600x460 uses 2x Retina scale",
			width:  1600,
			height: 460,
			want:   2,
		},
		{
			name:   "exact 4096 boundary uses 2x Retina scale",
			width:  1600,
			height: 4096,
			want:   2,
		},
		{
			name:   "height 4097 exceeding 8192 physical pixels falls back to 1x",
			width:  1600,
			height: 4097,
			want:   1,
		},
		{
			name:   "width 4097 exceeding 8192 physical pixels falls back to 1x",
			width:  4097,
			height: 900,
			want:   1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := timelineDeviceScaleFactor(tc.width, tc.height)
			if got != tc.want {
				t.Errorf("timelineDeviceScaleFactor(%d, %d) = %d, want %d", tc.width, tc.height, got, tc.want)
			}
		})
	}
}

func TestChromeTimelineImageRenderer_RenderTimelineWithMockCDP(t *testing.T) {
	wantPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x01, 0x02, 0x03, 0x04}

	type renderCallOutcome struct {
		Result         *TimelineRenderResult
		HasErr         bool
		IsTransportErr bool
	}

	standardMeasure := timelineMeasurePayload{
		Width:           1600,
		Height:          360,
		RenderedCount:   6,
		TotalMatched:    10,
		HasMore:         true,
		ViewStartTimeMs: 1700000000000,
		ViewEndTimeMs:   1700003600000,
	}
	oversizedMeasure := standardMeasure
	oversizedMeasure.Height = 4500

	standardResult := &TimelineRenderResult{
		PNGBytes:        wantPNG,
		RenderedCount:   6,
		TotalMatched:    10,
		HasMore:         true,
		ViewStartTimeMs: 1700000000000,
		ViewEndTimeMs:   1700003600000,
	}
	oversizedResult := &TimelineRenderResult{
		PNGBytes:        wantPNG,
		RenderedCount:   6,
		TotalMatched:    10,
		HasMore:         true,
		ViewStartTimeMs: 1700000000000,
		ViewEndTimeMs:   1700003600000,
	}

	testCases := []struct {
		name         string
		cdpConfig    fakeTimelineCDPConfig
		calls        []TimelineRenderRequest
		maxElapsed   time.Duration
		wantOutcomes []renderCallOutcome
		wantSession  fakeCDPSessionSnapshot
	}{
		{
			name: "reuses warm page session and falls back to 1x scale on oversized height",
			cdpConfig: fakeTimelineCDPConfig{
				screenshotPNG: wantPNG,
				applySteps: []fakeApplyStep{
					{measure: standardMeasure},
					{measure: standardMeasure},
					{measure: oversizedMeasure},
				},
			},
			calls: []TimelineRenderRequest{
				{InspectionID: "insp-1", MaxNodes: 50},
				{InspectionID: "insp-1", MaxNodes: 25},
				{InspectionID: "insp-1", MaxNodes: 150},
			},
			wantOutcomes: []renderCallOutcome{
				{Result: standardResult},
				{Result: standardResult},
				{Result: oversizedResult},
			},
			wantSession: fakeCDPSessionSnapshot{
				NavigatedURLs:   []string{"http://127.0.0.1:8080/render/mcp-timeline"},
				BridgePollCount: 1,
				Viewports: []recordedViewport{
					{Width: 1600, Height: 360, DeviceScaleFactor: 2},
					{Width: 1600, Height: 360, DeviceScaleFactor: 2},
					{Width: 1600, Height: 4500, DeviceScaleFactor: 1},
				},
			},
		},
		{
			name: "resets CDP session on JS exception and recovers on subsequent call",
			cdpConfig: fakeTimelineCDPConfig{
				screenshotPNG: wantPNG,
				applySteps: []fakeApplyStep{
					{exceptionText: "Uncaught Error: Failed to load inspection data"},
					{
						measure: timelineMeasurePayload{
							Width:           1200,
							Height:          240,
							RenderedCount:   3,
							TotalMatched:    3,
							HasMore:         false,
							ViewStartTimeMs: 1000,
							ViewEndTimeMs:   2000,
						},
					},
				},
			},
			calls: []TimelineRenderRequest{
				{InspectionID: "insp-fail", MaxNodes: 50},
				{InspectionID: "insp-ok", MaxNodes: 50},
			},
			wantOutcomes: []renderCallOutcome{
				{HasErr: true},
				{
					Result: &TimelineRenderResult{
						PNGBytes:        wantPNG,
						RenderedCount:   3,
						TotalMatched:    3,
						HasMore:         false,
						ViewStartTimeMs: 1000,
						ViewEndTimeMs:   2000,
					},
				},
			},
			wantSession: fakeCDPSessionSnapshot{
				NavigatedURLs: []string{
					"http://127.0.0.1:8080/render/mcp-timeline",
					"http://127.0.0.1:8080/render/mcp-timeline",
				},
				BridgePollCount: 2,
				Viewports: []recordedViewport{
					{Width: 1200, Height: 240, DeviceScaleFactor: 2},
				},
			},
		},
		{
			name: "fails fast on WebSocket disconnect after transient bridge poll error",
			cdpConfig: fakeTimelineCDPConfig{
				bridgePolls: []fakeBridgePollStep{
					{
						protocolErrCode:    -32000,
						protocolErrMessage: "Cannot find context with specified id",
					},
					{disconnect: true},
				},
			},
			calls: []TimelineRenderRequest{
				{InspectionID: "insp-crash", MaxNodes: 50},
			},
			maxElapsed: 2 * time.Second,
			wantOutcomes: []renderCallOutcome{
				{HasErr: true, IsTransportErr: true},
			},
			wantSession: fakeCDPSessionSnapshot{
				NavigatedURLs:   []string{"http://127.0.0.1:8080/render/mcp-timeline"},
				BridgePollCount: 2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeTimelineCDPServer(t, tc.cdpConfig)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			start := time.Now()
			gotOutcomes := make([]renderCallOutcome, 0, len(tc.calls))
			for i, call := range tc.calls {
				res, err := fake.renderer.RenderTimeline(ctx, call)
				if err != nil {
					t.Logf("call[%d] returned error: %v", i, err)
				}
				var transportErr *cdp.TransportError
				gotOutcomes = append(gotOutcomes, renderCallOutcome{
					Result:         res,
					HasErr:         err != nil,
					IsTransportErr: errors.As(err, &transportErr),
				})
			}
			elapsed := time.Since(start)

			if tc.maxElapsed > 0 && elapsed >= tc.maxElapsed {
				t.Errorf("RenderTimeline() took %v, want fast failure (<%v)", elapsed, tc.maxElapsed)
			}
			if diff := cmp.Diff(tc.wantOutcomes, gotOutcomes); diff != "" {
				t.Errorf("RenderTimeline() outcomes mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantSession, fake.snapshot()); diff != "" {
				t.Errorf("CDP session snapshot mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func setupRealisticTimelineTestServer(t *testing.T) (*coreinspection.InspectionTaskServer, string) {
	t.Helper()

	baseTime := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	populateTaskID := taskid.NewDefaultImplementationID[struct{}]("populate-render-timelines")
	populateTask := inspectiontaskbase.DefineInspectionTask(
		populateTaskID,
		func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[struct{}] {
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

				nodePath := k8sPath("core/v1", "node", "cluster-scope", "gke-prod-default-pool-a1b2c3")
				nodeReadyCond := tlAcc.GetPath(nodePath, khifilev6.PathSegment{Name: "Ready", Type: k8saudit.TimelineTypeResourceCondition})

				deployPath := k8sPath("apps/v1", "deployment", "default", "frontend-web")

				pod1Path := k8sPath("core/v1", "pod", "default", "frontend-web-6d8f9b-x7k2p")
				pod1Phase := tlAcc.GetPath(pod1Path, khifilev6.PathSegment{Name: "phase", Type: k8saudit.TimelineTypePodPhase})
				pod1Container := tlAcc.GetPath(pod1Path, khifilev6.PathSegment{Name: "app", Type: k8saudit.TimelineTypeContainer})
				pod1Endpoint := tlAcc.GetPath(pod1Path, khifilev6.PathSegment{Name: "default/frontend-web", Type: k8saudit.TimelineTypeEndpointSlice})

				pod2Path := k8sPath("core/v1", "pod", "default", "frontend-web-7c9a4d-m9q4w")
				pod2Phase := tlAcc.GetPath(pod2Path, khifilev6.PathSegment{Name: "phase", Type: k8saudit.TimelineTypePodPhase})
				pod2Container := tlAcc.GetPath(pod2Path, khifilev6.PathSegment{Name: "app", Type: k8saudit.TimelineTypeContainer})
				pod2Endpoint := tlAcc.GetPath(pod2Path, khifilev6.PathSegment{Name: "default/frontend-web", Type: k8saudit.TimelineTypeEndpointSlice})

				dnsPodPath := k8sPath("core/v1", "pod", "kube-system", "kube-dns-5b7c8d-z2v8n")
				dnsContainer := tlAcc.GetPath(dnsPodPath, khifilev6.PathSegment{Name: "kubedns", Type: k8saudit.TimelineTypeContainer})

				recordRevision := func(offset time.Duration, path *khifilev6.TimelinePath, state *pb.RevisionState, verb *pb.Verb, principal, summary string, sev *pb.Severity) error {
					ts := baseTime.Add(offset)
					node, err := structured.FromYAML("summary: ok")
					if err != nil {
						return err
					}
					l := log.NewLogWithTimestamp(idGen, structured.NewNodeReader(node), ts)
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
						ResourceBody: node,
						Principal:    principal,
						VerbType:     verb,
						StateType:    state,
					})
					return tlCS.Flush(tlAcc, builder.LogAccumulator)
				}

				recordEvent := func(offset time.Duration, path *khifilev6.TimelinePath, sev *pb.Severity, summary string) error {
					ts := baseTime.Add(offset)
					node, err := structured.FromYAML("message: ok")
					if err != nil {
						return err
					}
					l := log.NewLogWithTimestamp(idGen, structured.NewNodeReader(node), ts)
					logCS, err := khifilev6.NewLogChangeSet(l)
					if err != nil {
						return err
					}
					logCS.SetTimestamp(ts)
					logCS.SetSeverity(sev)
					logCS.SetLogType(k8saudit.LogTypeEvent)
					logCS.SetSummary(summary)
					if err := logCS.Flush(builder.LogAccumulator); err != nil {
						return err
					}

					tlCS := khifilev6.NewTimelineChangeSet(l)
					defer tlCS.Release()
					tlCS.AddEvent(path)
					return tlCS.Flush(tlAcc, builder.LogAccumulator)
				}

				steps := []func() error{
					// Node lifecycle & temporary pressure
					func() error {
						return recordRevision(0*time.Minute, nodePath, k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbCreate, "system:node:gke-prod", "Node registered", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(0*time.Minute, nodeReadyCond, k8saudit.RevisionStateConditionTrue, k8saudit.VerbStatusTrue, "kubelet", "Node Ready=True", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(24*time.Minute, nodeReadyCond, k8saudit.RevisionStateConditionFalse, k8saudit.VerbStatusFalse, "kubelet", "MemoryPressure detected", inspectioncore.SeverityWarning)
					},
					func() error {
						return recordEvent(24*time.Minute+30*time.Second, nodePath, inspectioncore.SeverityWarning, "SystemOOM: host memory pressure high")
					},
					func() error {
						return recordRevision(34*time.Minute, nodeReadyCond, k8saudit.RevisionStateConditionTrue, k8saudit.VerbStatusTrue, "kubelet", "Node Ready=True recovered", inspectioncore.SeverityInfo)
					},

					// Deployment rollout
					func() error {
						return recordRevision(2*time.Minute, deployPath, k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbCreate, "ci-deployer@example.com", "Create deployment frontend-web", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(15*time.Minute, deployPath, k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbUpdate, "ci-deployer@example.com", "Scale deployment replicas", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordEvent(26*time.Minute, deployPath, inspectioncore.SeverityWarning, "ReplicaSet updated after pod OOM termination")
					},
					func() error {
						return recordRevision(30*time.Minute, deployPath, k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbPatch, "deployment-controller", "Deployment status available", inspectioncore.SeverityInfo)
					},

					// Pod 1: Created -> Running -> OOMKilled -> Deleted
					func() error {
						return recordRevision(2*time.Minute+15*time.Second, pod1Path, k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbCreate, "replicaset-controller", "Pod created", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(2*time.Minute+15*time.Second, pod1Phase, k8saudit.RevisionStatePodPhasePending, k8saudit.VerbCreate, "kube-scheduler", "Pod pending", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(2*time.Minute+40*time.Second, pod1Phase, k8saudit.RevisionStatePodPhaseScheduled, k8saudit.VerbUpdate, "kube-scheduler", "Successfully assigned pod to node", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(2*time.Minute+45*time.Second, pod1Container, k8saudit.RevisionStateContainerWaiting, k8saudit.VerbCreate, "kubelet", "Pulling container image", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(3*time.Minute+30*time.Second, pod1Phase, k8saudit.RevisionStatePodPhaseRunning, k8saudit.VerbReady, "kubelet", "Pod running", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(3*time.Minute+30*time.Second, pod1Container, k8saudit.RevisionStateContainerRunningNonReady, k8saudit.VerbNonReady, "kubelet", "Container started", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(4*time.Minute+30*time.Second, pod1Container, k8saudit.RevisionStateContainerRunningReady, k8saudit.VerbReady, "kubelet", "Readiness probe succeeded", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(4*time.Minute+30*time.Second, pod1Endpoint, k8saudit.RevisionStateEndpointReady, k8saudit.VerbReady, "endpointslice-controller", "Endpoint ready", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordEvent(12*time.Minute, pod1Container, inspectioncore.SeverityInfo, "GET /api/v1/checkout 200 OK")
					},
					func() error {
						return recordEvent(20*time.Minute, pod1Container, inspectioncore.SeverityWarning, "High heap allocation rate detected")
					},
					func() error {
						return recordEvent(25*time.Minute, pod1Container, inspectioncore.SeverityError, "OOMKilled: container app exceeded memory limit 512Mi")
					},
					func() error {
						return recordRevision(25*time.Minute+10*time.Second, pod1Container, k8saudit.RevisionStateContainerTerminatedWithError, k8saudit.VerbDelete, "kubelet", "Container exited with code 137", inspectioncore.SeverityError)
					},
					func() error {
						return recordRevision(25*time.Minute+15*time.Second, pod1Endpoint, k8saudit.RevisionStateEndpointUnready, k8saudit.VerbNonReady, "endpointslice-controller", "Endpoint unready", inspectioncore.SeverityWarning)
					},
					func() error {
						return recordRevision(25*time.Minute+30*time.Second, pod1Phase, k8saudit.RevisionStatePodPhaseFailed, k8saudit.VerbDelete, "kubelet", "Pod failed", inspectioncore.SeverityError)
					},
					func() error {
						return recordRevision(26*time.Minute, pod1Path, k8saudit.RevisionStateK8sResourceDeleting, k8saudit.VerbDelete, "replicaset-controller", "Pod terminating", inspectioncore.SeverityWarning)
					},
					func() error {
						return recordRevision(26*time.Minute, pod1Endpoint, k8saudit.RevisionStateEndpointTerminating, k8saudit.VerbTerminating, "endpointslice-controller", "Endpoint terminating", inspectioncore.SeverityWarning)
					},
					func() error {
						return recordRevision(28*time.Minute, pod1Path, k8saudit.RevisionStateK8sResourceDeleted, k8saudit.VerbDelete, "kubelet", "Pod deleted", inspectioncore.SeverityInfo)
					},

					// Pod 2: Replacement pod Created -> Scheduled -> Running -> Ready
					func() error {
						return recordRevision(26*time.Minute+15*time.Second, pod2Path, k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbCreate, "replicaset-controller", "Replacement pod created", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(26*time.Minute+15*time.Second, pod2Phase, k8saudit.RevisionStatePodPhasePending, k8saudit.VerbCreate, "kube-scheduler", "Pod pending", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(26*time.Minute+45*time.Second, pod2Phase, k8saudit.RevisionStatePodPhaseScheduled, k8saudit.VerbUpdate, "kube-scheduler", "Scheduled replacement pod", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(27*time.Minute, pod2Container, k8saudit.RevisionStateContainerWaiting, k8saudit.VerbCreate, "kubelet", "Container waiting", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(28*time.Minute, pod2Phase, k8saudit.RevisionStatePodPhaseRunning, k8saudit.VerbReady, "kubelet", "Pod running", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(28*time.Minute, pod2Container, k8saudit.RevisionStateContainerRunningNonReady, k8saudit.VerbNonReady, "kubelet", "Container starting", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(29*time.Minute+30*time.Second, pod2Container, k8saudit.RevisionStateContainerRunningReady, k8saudit.VerbReady, "kubelet", "Readiness probe succeeded", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(29*time.Minute+30*time.Second, pod2Endpoint, k8saudit.RevisionStateEndpointReady, k8saudit.VerbReady, "endpointslice-controller", "Endpoint ready", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordEvent(36*time.Minute, pod2Container, inspectioncore.SeverityInfo, "GET /api/v1/checkout 200 OK")
					},
					func() error {
						return recordEvent(48*time.Minute, pod2Container, inspectioncore.SeverityInfo, "GET /healthz 200 OK")
					},

					// kube-system DNS pod
					func() error {
						return recordRevision(0*time.Minute, dnsPodPath, k8saudit.RevisionStateK8sResourceExisting, k8saudit.VerbCreate, "kubelet", "kube-dns pod active", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordRevision(0*time.Minute, dnsContainer, k8saudit.RevisionStateContainerRunningReady, k8saudit.VerbReady, "kubelet", "kubedns container ready", inspectioncore.SeverityInfo)
					},
					func() error {
						return recordEvent(25*time.Minute+20*time.Second, dnsContainer, inspectioncore.SeverityWarning, "Transient DNS lookup retry during node memory pressure")
					},
					func() error {
						return recordEvent(40*time.Minute, dnsContainer, inspectioncore.SeverityInfo, "DNS cache stats healthy")
					},
				}

				for _, step := range steps {
					if err := step(); err != nil {
						return struct{}{}, err
					}
				}
				return struct{}{}, nil
			}
		},
		coretask.ProvidesTag(inspectiontaskbase.TagTimelineMapper),
		coretask.WithLabelValue(inspectioncore.LabelKeyInspectionDefaultFeatureFlag, true),
		coretask.WithLabelValue(inspectioncore.LabelKeyInspectionFeatureFlag, true),
	)
	return inspectionserver.NewServerWithInspection(t, populateTask)
}

func TestChromeTimelineImageRenderer_EndToEndWithLocalServer(t *testing.T) {
	if _, err := cdp.FindChromeBinary(); err != nil {
		t.Skipf("skipping browser integration test: %v", err)
	}
	gin.SetMode(gin.TestMode)

	inspectionServer, inspectionID := setupRealisticTimelineTestServer(t)
	indexMgr := workbench.NewInspectionIndexManager(inspectionServer, t.TempDir())
	workbenchMgr := workbench.NewWorkbenchManager(inspectionServer, indexMgr, 5)
	t.Cleanup(indexMgr.Wait)

	ginEngine := gin.New()
	server.SetupFrontendMiddleware(ginEngine, "", "")
	server.SetupFrontendRoutes(ginEngine, "")

	inspPath, inspHandler := apiv1connect.NewInspectionServiceHandler(
		serverapiv1.NewInspectionServiceServerWithIntervals(inspectionServer, 100*time.Millisecond, 50*time.Millisecond),
	)
	ginEngine.Any(inspPath+"*action", gin.WrapH(inspHandler))

	wbPath, wbHandler := apiv1connect.NewWorkbenchServiceHandler(
		serverapiv1.NewWorkbenchServiceServer(workbenchMgr),
	)
	ginEngine.Any(wbPath+"*action", gin.WrapH(wbHandler))

	httpServer := httptest.NewServer(ginEngine)
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	browser, err := cdp.StartBrowser(ctx)
	if err != nil {
		t.Fatalf("cdp.StartBrowser() failed: %v", err)
	}
	renderer := NewChromeTimelineImageRenderer(browser, httpServer.URL)
	defer func() {
		_ = renderer.Close()
		_ = browser.Close()
		httpServer.CloseClientConnections()
	}()

	baseTime := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	const minStagePNGWidth = 1588 * defaultTimelineDeviceScaleFactor
	const minHeaderPNGHeight = 60 * defaultTimelineDeviceScaleFactor

	testCases := []struct {
		name            string
		req             TimelineRenderRequest
		wantPNGWidth    int
		wantMinPNGWidth int
		want            *TimelineRenderResult
	}{
		{
			name: "short hierarchy labels keep minimum stage width",
			req: TimelineRenderRequest{
				InspectionID: inspectionID,
				MaxDepth:     1,
				MaxNodes:     50,
			},
			wantPNGWidth: minStagePNGWidth,
			want: &TimelineRenderResult{
				RenderedCount:   5,
				TotalMatched:    5,
				HasMore:         false,
				ViewStartTimeMs: baseTime.UnixMilli(),
				ViewEndTimeMs:   baseTime.Add(1 * time.Hour).UnixMilli(),
			},
		},
		{
			name: "long resource names expand stage width horizontally",
			req: TimelineRenderRequest{
				InspectionID: inspectionID,
				MaxNodes:     50,
			},
			wantMinPNGWidth: minStagePNGWidth + 1,
			want: &TimelineRenderResult{
				RenderedCount:   22,
				TotalMatched:    22,
				HasMore:         false,
				ViewStartTimeMs: baseTime.UnixMilli(),
				ViewEndTimeMs:   baseTime.Add(1 * time.Hour).UnixMilli(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := renderer.RenderTimeline(ctx, tc.req)
			if err != nil {
				t.Fatalf("RenderTimeline() unexpected error = %v", err)
			}
			pngCfg, err := png.DecodeConfig(bytes.NewReader(res.PNGBytes))
			if err != nil {
				t.Fatalf("png.DecodeConfig() failed: %v", err)
			}
			if pngCfg.Height <= minHeaderPNGHeight {
				t.Errorf("captured PNG height = %d, want > %d", pngCfg.Height, minHeaderPNGHeight)
			}
			if tc.wantPNGWidth > 0 && pngCfg.Width != tc.wantPNGWidth {
				t.Errorf("captured PNG width = %d, want %d", pngCfg.Width, tc.wantPNGWidth)
			}
			if tc.wantMinPNGWidth > 0 && pngCfg.Width < tc.wantMinPNGWidth {
				t.Errorf("captured PNG width = %d, want >= %d", pngCfg.Width, tc.wantMinPNGWidth)
			}
			if diff := cmp.Diff(tc.want, res, cmpopts.IgnoreFields(TimelineRenderResult{}, "PNGBytes")); diff != "" {
				t.Errorf("RenderTimeline() result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
