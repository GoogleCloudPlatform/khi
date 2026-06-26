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

package googlecloudlogcsm_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestCSMAccessLogLogIngester_ProcessLog(t *testing.T) {
	testCases := []struct {
		desc                string
		inputGCPAccessLog   *googlecloudcommon_contract.GCPAccessLogFieldSet
		inputIstioAccessLog *googlecloudlogcsm_contract.IstioAccessLogFieldSet
		wantSummary         string
	}{
		{
			desc: "server access log with normal response",
			inputGCPAccessLog: &googlecloudcommon_contract.GCPAccessLogFieldSet{
				Status:     200,
				Method:     "GET",
				RequestURL: "/productpage",
			},
			inputIstioAccessLog: &googlecloudlogcsm_contract.IstioAccessLogFieldSet{
				Type:         googlecloudlogcsm_contract.AccessLogTypeServer,
				ResponseFlag: googlecloudlogcsm_contract.ResponseFlagNoError,
			},
			wantSummary: "200 GET /productpage",
		},
		{
			desc: "server access log with error response",
			inputGCPAccessLog: &googlecloudcommon_contract.GCPAccessLogFieldSet{
				Status:     503,
				Method:     "GET",
				RequestURL: "/productpage",
			},
			inputIstioAccessLog: &googlecloudlogcsm_contract.IstioAccessLogFieldSet{
				Type:         googlecloudlogcsm_contract.AccessLogTypeServer,
				ResponseFlag: googlecloudlogcsm_contract.ResponseFlagNoHealthyUpstream,
			},
			wantSummary: "【No healthy upstream(UH)】503 GET /productpage",
		},
		{
			desc: "server access log with multiple error response flags",
			inputGCPAccessLog: &googlecloudcommon_contract.GCPAccessLogFieldSet{
				Status:     503,
				Method:     "GET",
				RequestURL: "/productpage",
			},
			inputIstioAccessLog: &googlecloudlogcsm_contract.IstioAccessLogFieldSet{
				Type:         googlecloudlogcsm_contract.AccessLogTypeServer,
				ResponseFlag: "UH,URX",
			},
			wantSummary: "【No healthy upstream, Upstream retry limit exceeded(UH,URX)】503 GET /productpage",
		},
	}

	ingester := &CSMAccessLogLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)},
				tc.inputGCPAccessLog,
				tc.inputIstioAccessLog,
			)
			cs, err := ingester.ProcessLog(t.Context(), l)
			if err != nil {
				t.Fatalf("ProcessLog() failed: %v", err)
			}
			testchangeset.AssertLog(t, cs).
				HasSummary(tc.wantSummary).
				HasLogType(googlecloudlogcsm_contract.LogTypeCSMAccessLog)
		})
	}
}

func TestCSMAccessLogLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	testCases := []struct {
		desc                string
		inputGCPAccessLog   *googlecloudcommon_contract.GCPAccessLogFieldSet
		inputIstioAccessLog *googlecloudlogcsm_contract.IstioAccessLogFieldSet
		assert              func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "server access log with client and service",
			inputGCPAccessLog: &googlecloudcommon_contract.GCPAccessLogFieldSet{
				Status:     200,
				Method:     "GET",
				RequestURL: "/productpage",
			},
			inputIstioAccessLog: &googlecloudlogcsm_contract.IstioAccessLogFieldSet{
				Type:                        googlecloudlogcsm_contract.AccessLogTypeServer,
				ResponseFlag:                googlecloudlogcsm_contract.ResponseFlagNoError,
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
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore_contract.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "istio-ingressgateway", Type: inspectioncore_contract.TimelineTypeResource},
					khifilev6.PathSegment{Name: "client", Type: googlecloudlogcsm_contract.TimelineTypeCSMAccessLog},
				)
				wantProductpagePath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore_contract.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "productpage-v1", Type: inspectioncore_contract.TimelineTypeResource},
					khifilev6.PathSegment{Name: "server:istio-proxy", Type: googlecloudlogcsm_contract.TimelineTypeCSMAccessLog},
				)
				wantServicePath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore_contract.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "service", Type: inspectioncore_contract.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "productpage", Type: inspectioncore_contract.TimelineTypeResource},
					khifilev6.PathSegment{Name: "server", Type: googlecloudlogcsm_contract.TimelineTypeCSMAccessLog},
				)

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantGatewayPath).
					HasEvent(wantProductpagePath).
					HasEvent(wantServicePath)
			},
		},
		{
			desc: "client access log with destination and service",
			inputGCPAccessLog: &googlecloudcommon_contract.GCPAccessLogFieldSet{
				Status:     200,
				Method:     "GET",
				RequestURL: "/details",
			},
			inputIstioAccessLog: &googlecloudlogcsm_contract.IstioAccessLogFieldSet{
				Type:                        googlecloudlogcsm_contract.AccessLogTypeClient,
				ResponseFlag:                googlecloudlogcsm_contract.ResponseFlagNoError,
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
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore_contract.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "details-v1", Type: inspectioncore_contract.TimelineTypeResource},
					khifilev6.PathSegment{Name: "server", Type: googlecloudlogcsm_contract.TimelineTypeCSMAccessLog},
				)
				wantProductpagePath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore_contract.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "productpage-v1", Type: inspectioncore_contract.TimelineTypeResource},
					khifilev6.PathSegment{Name: "client", Type: googlecloudlogcsm_contract.TimelineTypeCSMAccessLog},
				)
				wantDetailsServicePath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore_contract.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "service", Type: inspectioncore_contract.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "details", Type: inspectioncore_contract.TimelineTypeResource},
					khifilev6.PathSegment{Name: "client", Type: googlecloudlogcsm_contract.TimelineTypeCSMAccessLog},
				)

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantDetailsPath).
					HasEvent(wantProductpagePath).
					HasEvent(wantDetailsServicePath)
			},
		},
	}

	mapper := &CSMAccessLogLogToTimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			builder := khifilev6.NewBuilder()
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, googlecloudlogcsm_contract.ClusterIdentityTaskID.Ref(), googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			})

			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)},
				tc.inputGCPAccessLog,
				tc.inputIstioAccessLog,
			)
			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() failed: %v", err)
			}
			tc.assert(t, builder, cs)
		})
	}
}
