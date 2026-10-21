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
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
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
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/websocket"
)

type mockCDPRequest struct {
	ID     int            `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

func TestChromeTimelineImageRenderer_RenderTimelineWithMockCDP(t *testing.T) {
	wantPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x01, 0x02, 0x03, 0x04}
	encodedPNG := base64.StdEncoding.EncodeToString(wantPNG)

	var navigatedURLs []string
	var viewportWidths []float64
	var viewportHeights []float64

	wsServer := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		for {
			var raw string
			if err := websocket.Message.Receive(ws, &raw); err != nil {
				return
			}
			var req mockCDPRequest
			if err := json.Unmarshal([]byte(raw), &req); err != nil {
				return
			}

			var result any
			switch req.Method {
			case "Page.enable", "Runtime.enable":
				result = map[string]any{}
			case "Page.navigate":
				if u, ok := req.Params["url"].(string); ok {
					navigatedURLs = append(navigatedURLs, u)
				}
				result = map[string]any{"frameId": "frame-1"}
			case "Runtime.evaluate":
				expr, _ := req.Params["expression"].(string)
				switch {
				case strings.Contains(expr, "typeof window.__khiCDPBridge?.timelineRenderer"):
					result = map[string]any{
						"result": map[string]any{
							"type":  "boolean",
							"value": true,
						},
					}
				case strings.Contains(expr, "applyAndMeasure"):
					result = map[string]any{
						"result": map[string]any{
							"type": "object",
							"value": timelineMeasurePayload{
								Width:           1600,
								Height:          360,
								RenderedCount:   6,
								TotalMatched:    10,
								HasMore:         true,
								ViewStartTimeMs: 1700000000000,
								ViewEndTimeMs:   1700003600000,
							},
						},
					}
				case strings.Contains(expr, "waitForStableFrame"):
					result = map[string]any{
						"result": map[string]any{
							"type": "undefined",
						},
					}
				}
			case "Emulation.setDeviceMetricsOverride":
				if w, ok := req.Params["width"].(float64); ok {
					viewportWidths = append(viewportWidths, w)
				}
				if h, ok := req.Params["height"].(float64); ok {
					viewportHeights = append(viewportHeights, h)
				}
				result = map[string]any{}
			case "Page.captureScreenshot":
				result = map[string]any{
					"data": encodedPNG,
				}
			}

			respBytes, _ := json.Marshal(map[string]any{
				"id":     req.ID,
				"result": result,
			})
			_ = websocket.Message.Send(ws, string(respBytes))
		}
	}))
	defer wsServer.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	browser := cdp.NewBrowserFromPageWSURL(wsURL)
	defer browser.Close()
	renderer := NewChromeTimelineImageRenderer(browser, "http://127.0.0.1:8080")
	defer renderer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gotFirst, err := renderer.RenderTimeline(ctx, TimelineRenderRequest{
		InspectionID: "insp-1",
		MaxNodes:     50,
		Width:        1600,
	})
	if err != nil {
		t.Fatalf("first RenderTimeline() unexpected error = %v", err)
	}

	wantRes := &TimelineRenderResult{
		PNGBytes:        wantPNG,
		Width:           1600,
		Height:          360,
		RenderedCount:   6,
		TotalMatched:    10,
		HasMore:         true,
		ViewStartTimeMs: 1700000000000,
		ViewEndTimeMs:   1700003600000,
	}
	if diff := cmp.Diff(wantRes, gotFirst); diff != "" {
		t.Errorf("first RenderTimeline() result mismatch (-want +got):\n%s", diff)
	}

	// Second call reuses the warm page session without re-navigating.
	gotSecond, err := renderer.RenderTimeline(ctx, TimelineRenderRequest{
		InspectionID: "insp-1",
		MaxNodes:     25,
		Width:        1600,
	})
	if err != nil {
		t.Fatalf("second RenderTimeline() unexpected error = %v", err)
	}
	if diff := cmp.Diff(wantRes, gotSecond); diff != "" {
		t.Errorf("second RenderTimeline() result mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"http://127.0.0.1:8080/render/timeline"}, navigatedURLs); diff != "" {
		t.Errorf("navigatedURLs mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]float64{1600, 1600}, viewportWidths); diff != "" {
		t.Errorf("viewportWidths mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]float64{360, 360}, viewportHeights); diff != "" {
		t.Errorf("viewportHeights mismatch (-want +got):\n%s", diff)
	}
}

func TestChromeTimelineImageRenderer_ErrorAndRecoveryWithMockCDP(t *testing.T) {
	wantPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x01, 0x02, 0x03, 0x04}
	encodedPNG := base64.StdEncoding.EncodeToString(wantPNG)

	var applyAttempts int
	var navigateCount int

	wsServer := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		for {
			var raw string
			if err := websocket.Message.Receive(ws, &raw); err != nil {
				return
			}
			var req mockCDPRequest
			if err := json.Unmarshal([]byte(raw), &req); err != nil {
				return
			}

			var result any
			switch req.Method {
			case "Page.enable", "Runtime.enable", "Emulation.setDeviceMetricsOverride":
				result = map[string]any{}
			case "Page.navigate":
				navigateCount++
				result = map[string]any{"frameId": "frame-1"}
			case "Runtime.evaluate":
				expr, _ := req.Params["expression"].(string)
				switch {
				case strings.Contains(expr, "typeof window.__khiCDPBridge?.timelineRenderer"):
					result = map[string]any{
						"result": map[string]any{"type": "boolean", "value": true},
					}
				case strings.Contains(expr, "applyAndMeasure"):
					applyAttempts++
					if applyAttempts == 1 {
						result = map[string]any{
							"result": map[string]any{"type": "object"},
							"exceptionDetails": map[string]any{
								"text": "Uncaught Error: Failed to load inspection data",
							},
						}
					} else {
						result = map[string]any{
							"result": map[string]any{
								"type": "object",
								"value": timelineMeasurePayload{
									Width:           1200,
									Height:          240,
									RenderedCount:   3,
									TotalMatched:    3,
									HasMore:         false,
									ViewStartTimeMs: 1000,
									ViewEndTimeMs:   2000,
								},
							},
						}
					}
				case strings.Contains(expr, "waitForStableFrame"):
					result = map[string]any{
						"result": map[string]any{"type": "undefined"},
					}
				}
			case "Page.captureScreenshot":
				result = map[string]any{"data": encodedPNG}
			}

			respBytes, _ := json.Marshal(map[string]any{
				"id":     req.ID,
				"result": result,
			})
			_ = websocket.Message.Send(ws, string(respBytes))
		}
	}))
	defer wsServer.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	browser := cdp.NewBrowserFromPageWSURL(wsURL)
	defer browser.Close()
	renderer := NewChromeTimelineImageRenderer(browser, "http://127.0.0.1:8080")
	defer renderer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First call fails due to JS exception in applyAndMeasure and resets the CDP session.
	_, err := renderer.RenderTimeline(ctx, TimelineRenderRequest{
		InspectionID: "insp-fail",
		MaxNodes:     50,
		Width:        1200,
	})
	if err == nil {
		t.Fatal("first RenderTimeline() expected error, got nil")
	}

	// Second call reconnects, re-navigates, and succeeds.
	got, err := renderer.RenderTimeline(ctx, TimelineRenderRequest{
		InspectionID: "insp-ok",
		MaxNodes:     50,
		Width:        1200,
	})
	if err != nil {
		t.Fatalf("second RenderTimeline() unexpected error = %v", err)
	}
	if got.RenderedCount != 3 || navigateCount != 2 {
		t.Errorf("second RenderTimeline() renderedCount=%d, navigateCount=%d; want 3, 2", got.RenderedCount, navigateCount)
	}
}

func TestChromeTimelineImageRenderer_NavigateTransportErrorFailsFast(t *testing.T) {
	var evaluatePollCount int

	wsServer := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		for {
			var raw string
			if err := websocket.Message.Receive(ws, &raw); err != nil {
				return
			}
			var req mockCDPRequest
			if err := json.Unmarshal([]byte(raw), &req); err != nil {
				return
			}

			switch req.Method {
			case "Page.enable", "Runtime.enable":
				respBytes, _ := json.Marshal(map[string]any{"id": req.ID, "result": map[string]any{}})
				_ = websocket.Message.Send(ws, string(respBytes))
			case "Page.navigate":
				respBytes, _ := json.Marshal(map[string]any{"id": req.ID, "result": map[string]any{"frameId": "frame-1"}})
				_ = websocket.Message.Send(ws, string(respBytes))
			case "Runtime.evaluate":
				evaluatePollCount++
				if evaluatePollCount == 1 {
					// Simulate transient CDP protocol error during navigation.
					respBytes, _ := json.Marshal(map[string]any{
						"id": req.ID,
						"error": map[string]any{
							"code":    -32000,
							"message": "Cannot find context with specified id",
						},
					})
					_ = websocket.Message.Send(ws, string(respBytes))
					continue
				}
				// On second poll, close the WebSocket connection to simulate browser crash / disconnect.
				_ = ws.Close()
				return
			}
		}
	}))
	defer wsServer.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	browser := cdp.NewBrowserFromPageWSURL(wsURL)
	defer browser.Close()
	renderer := NewChromeTimelineImageRenderer(browser, "http://127.0.0.1:8080")
	defer renderer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	_, err := renderer.RenderTimeline(ctx, TimelineRenderRequest{
		InspectionID: "insp-crash",
		MaxNodes:     50,
		Width:        1200,
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("RenderTimeline() expected transport error, got nil")
	}
	if elapsed >= 2*time.Second {
		t.Errorf("RenderTimeline() took %v on transport disconnect, want fast failure (<2s)", elapsed)
	}
	if evaluatePollCount != 2 {
		t.Errorf("evaluatePollCount = %d, want 2 (1 transient protocol error + 1 transport disconnect)", evaluatePollCount)
	}
}

func setupRealisticTimelineTestServer(t *testing.T) (*coreinspection.InspectionTaskServer, string) {
	t.Helper()
	ioConfig, err := inspectioncore.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("failed to create test IOConfig: %v", err)
	}
	tempDir := t.TempDir()
	ioConfig.DataDestination = tempDir
	ioConfig.TemporaryFolder = tempDir
	inspectionServer, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("failed to create inspection server: %v", err)
	}

	if err := inspectionServer.AddInspectionType(coreinspection.InspectionType{
		Id:   "k8s-render-test",
		Name: "Kubernetes Render Test",
	}); err != nil {
		t.Fatalf("failed to add inspection type: %v", err)
	}

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
	if err := inspectionServer.AddTask(populateTask); err != nil {
		t.Fatalf("failed to add populate task: %v", err)
	}

	inspectionID, err := inspectionServer.CreateInspection("k8s-render-test")
	if err != nil {
		t.Fatalf("failed to create inspection: %v", err)
	}
	runner := inspectionServer.GetInspection(inspectionID)
	if err := runner.Run(context.Background(), &inspectioncore.InspectionRequest{Values: map[string]any{}}); err != nil {
		t.Fatalf("failed to run inspection: %v", err)
	}
	<-runner.Wait()
	return inspectionServer, inspectionID
}

func TestChromeTimelineImageRenderer_EndToEndWithLocalServer(t *testing.T) {
	if _, err := cdp.FindChromeBinary(); err != nil {
		t.Skipf("skipping browser integration test: %v", err)
	}
	gin.SetMode(gin.TestMode)

	inspectionServer, inspectionID := setupRealisticTimelineTestServer(t)
	indexMgr := workbench.NewInspectionIndexManager(inspectionServer, t.TempDir())
	workbenchMgr := workbench.NewWorkbenchManager(inspectionServer, indexMgr, 5)
	t.Cleanup(func() {
		indexMgr.Wait()
	})

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

	res, err := renderer.RenderTimeline(ctx, TimelineRenderRequest{
		InspectionID: inspectionID,
		MaxNodes:     50,
		Width:        1600,
	})
	if err != nil {
		t.Fatalf("RenderTimeline() unexpected error = %v", err)
	}
	if len(res.PNGBytes) < 8 || string(res.PNGBytes[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("RenderTimeline() did not return valid PNG header, len=%d", len(res.PNGBytes))
	}
	if res.Width != 1600 || res.Height <= 60 || res.RenderedCount < 10 {
		t.Errorf("RenderTimeline() dimensions/count = (%d, %d, %d), want width=1600, height>60, renderedCount>=10", res.Width, res.Height, res.RenderedCount)
	}
}
