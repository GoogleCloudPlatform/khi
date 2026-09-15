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

package csm_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csm"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestCSMTrafficLogLogIngester_ProcessLog(t *testing.T) {
	testCases := []struct {
		desc                string
		inputGCPAccessLog   *gcpcommon.GCPAccessLogFieldSet
		inputIstioAccessLog *csm.IstioAccessLogFieldSet
		wantSummary         string
	}{
		{
			desc: "server access log with normal response",
			inputGCPAccessLog: &gcpcommon.GCPAccessLogFieldSet{
				Status:     200,
				Method:     "GET",
				RequestURL: "/productpage",
			},
			inputIstioAccessLog: &csm.IstioAccessLogFieldSet{
				Type:          csm.AccessLogTypeServer,
				ResponseFlags: logutil.EnvoyResponseFlags{logutil.EnvoyResponseFlagNoError},
			},
			wantSummary: "200 GET /productpage",
		},
		{
			desc: "server access log with missing response flags",
			inputGCPAccessLog: &gcpcommon.GCPAccessLogFieldSet{
				Status:     200,
				Method:     "GET",
				RequestURL: "/productpage",
			},
			inputIstioAccessLog: &csm.IstioAccessLogFieldSet{
				Type:          csm.AccessLogTypeServer,
				ResponseFlags: logutil.EnvoyResponseFlags{},
			},
			wantSummary: "200 GET /productpage",
		},
		{
			desc: "server access log with error response",
			inputGCPAccessLog: &gcpcommon.GCPAccessLogFieldSet{
				Status:     503,
				Method:     "GET",
				RequestURL: "/productpage",
			},
			inputIstioAccessLog: &csm.IstioAccessLogFieldSet{
				Type:          csm.AccessLogTypeServer,
				ResponseFlags: logutil.EnvoyResponseFlags{logutil.EnvoyResponseFlagNoHealthyUpstream},
			},
			wantSummary: "【No healthy upstream(UH)】503 GET /productpage",
		},
		{
			desc: "server access log with multiple error response flags",
			inputGCPAccessLog: &gcpcommon.GCPAccessLogFieldSet{
				Status:     503,
				Method:     "GET",
				RequestURL: "/productpage",
			},
			inputIstioAccessLog: &csm.IstioAccessLogFieldSet{
				Type:          csm.AccessLogTypeServer,
				ResponseFlags: logutil.EnvoyResponseFlags{logutil.EnvoyResponseFlagNoHealthyUpstream, logutil.EnvoyResponseFlagUpstreamRetryLimitExceeded},
			},
			wantSummary: "【No healthy upstream, Upstream retry limit exceeded(UH,URX)】503 GET /productpage",
		},
	}

	ingester := &CSMTrafficLogLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := testlog.NewMockLog(
				time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
				*tc.inputGCPAccessLog,
				*tc.inputIstioAccessLog,
			)
			cs, err := ingester.ProcessLog(t.Context(), l)
			if err != nil {
				t.Fatalf("ProcessLog() failed: %v", err)
			}
			testchangeset.AssertLog(t, cs).
				HasSummary(tc.wantSummary).
				HasLogType(csm.LogTypeCSMTrafficLog)
		})
	}
}

func TestCSMTrafficLogLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	testCases := []struct {
		desc                string
		inputGCPAccessLog   *gcpcommon.GCPAccessLogFieldSet
		inputIstioAccessLog *csm.IstioAccessLogFieldSet
		assert              func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "server access log with client and service",
			inputGCPAccessLog: &gcpcommon.GCPAccessLogFieldSet{
				Status:     200,
				Method:     "GET",
				RequestURL: "/productpage",
			},
			inputIstioAccessLog: &csm.IstioAccessLogFieldSet{
				Type:                        csm.AccessLogTypeServer,
				ResponseFlags:               logutil.EnvoyResponseFlags{logutil.EnvoyResponseFlagNoError},
				ReporterPodNamespace:        "default",
				ReporterPodName:             "productpage-v1",
				ReporterContainerName:       "istio-proxy",
				SourceNamespace:             "default",
				SourceName:                  "istio-ingressgateway",
				DestinationNamespace:        "default",
				DestinationServiceName:      "productpage",
				DestinationServiceNamespace: "default",
			},
			assert: func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				wantGatewayPath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "pod", Type: inspectioncore.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "istio-ingressgateway", Type: inspectioncore.TimelineTypeResource},
					khifilev6.PathSegment{Name: "client", Type: csm.TimelineTypeCSMTrafficLog},
				)
				wantProductpagePath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "pod", Type: inspectioncore.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "productpage-v1", Type: inspectioncore.TimelineTypeResource},
					khifilev6.PathSegment{Name: "server:istio-proxy", Type: csm.TimelineTypeCSMTrafficLog},
				)
				wantServicePath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "service", Type: inspectioncore.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "productpage", Type: inspectioncore.TimelineTypeResource},
					khifilev6.PathSegment{Name: "server", Type: csm.TimelineTypeCSMTrafficLog},
				)

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantGatewayPath).
					HasEvent(wantProductpagePath).
					HasEvent(wantServicePath)
			},
		},
		{
			desc: "client access log with destination and service",
			inputGCPAccessLog: &gcpcommon.GCPAccessLogFieldSet{
				Status:     200,
				Method:     "GET",
				RequestURL: "/details",
			},
			inputIstioAccessLog: &csm.IstioAccessLogFieldSet{
				Type:                        csm.AccessLogTypeClient,
				ResponseFlags:               logutil.EnvoyResponseFlags{logutil.EnvoyResponseFlagNoError},
				ReporterPodNamespace:        "default",
				ReporterPodName:             "productpage-v1",
				SourceNamespace:             "default",
				SourceName:                  "productpage-v1",
				DestinationNamespace:        "default",
				DestinationName:             "details-v1",
				DestinationServiceName:      "details",
				DestinationServiceNamespace: "default",
			},
			assert: func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				wantDetailsPath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "pod", Type: inspectioncore.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "details-v1", Type: inspectioncore.TimelineTypeResource},
					khifilev6.PathSegment{Name: "server", Type: csm.TimelineTypeCSMTrafficLog},
				)
				wantProductpagePath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "pod", Type: inspectioncore.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "productpage-v1", Type: inspectioncore.TimelineTypeResource},
					khifilev6.PathSegment{Name: "client", Type: csm.TimelineTypeCSMTrafficLog},
				)
				wantDetailsServicePath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "service", Type: inspectioncore.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "details", Type: inspectioncore.TimelineTypeResource},
					khifilev6.PathSegment{Name: "client", Type: csm.TimelineTypeCSMTrafficLog},
				)

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantDetailsPath).
					HasEvent(wantProductpagePath).
					HasEvent(wantDetailsServicePath)
			},
		},
	}

	mapper := &CSMTrafficLogLogToTimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			builder := khifilev6.NewTestBuilder(id.NewGenerator())
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, csm.ClusterIdentityTaskID.Ref(), k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			})

			l := testlog.NewMockLog(
				time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
				*tc.inputGCPAccessLog,
				*tc.inputIstioAccessLog,
			)
			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() failed: %v", err)
			}
			tc.assert(t, builder, cs)
		})
	}
}
