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

package googlecloudlogcsm_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestLogToTimelineMapper(t *testing.T) {
	testCases := []struct {
		desc                string
		inputGCPAccessLog   *googlecloudcommon_contract.GCPAccessLogFieldSet
		inputIstioAccessLog *googlecloudlogcsm_contract.IstioAccessLogFieldSet
		asserters           []testchangeset.ChangeSetAsserter
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
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasLogSummary{WantLogSummary: "200 GET /productpage"},
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"core/v1#pod#default#istio-ingressgateway#client",
						"core/v1#pod#default#productpage-v1#server:istio-proxy",
						"core/v1#service#default#productpage#server",
					},
				},
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
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasLogSummary{WantLogSummary: "200 GET /details"},
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"core/v1#pod#default#details-v1#server",
						"core/v1#pod#default#productpage-v1#client",
						"core/v1#service#default#details#client",
					},
				},
			},
		},
		{
			desc: "server access log with error",
			inputGCPAccessLog: &googlecloudcommon_contract.GCPAccessLogFieldSet{
				Status:     503,
				Method:     "GET",
				RequestURL: "/productpage",
			},
			inputIstioAccessLog: &googlecloudlogcsm_contract.IstioAccessLogFieldSet{
				Type:                        googlecloudlogcsm_contract.AccessLogTypeServer,
				ResponseFlag:                googlecloudlogcsm_contract.ResponseFlagNoHealthyUpstream,
				ReporterPodNamespace:        "default",
				ReporterPodName:             "productpage-v1",
				ReporterContainerName:       "istio-proxy",
				SourceNamespace:             "default",
				SourceName:                  "istio-ingressgateway",
				DestinationNamespace:        "default",
				DestinationServiceName:      "productpage",
				DestinationServiceNamespace: "product",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasLogSummary{WantLogSummary: "【No healthy upstream(UH)】503 GET /productpage"},
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"core/v1#pod#default#istio-ingressgateway#client",
						"core/v1#pod#default#productpage-v1#server:istio-proxy",
						"core/v1#service#product#productpage#server",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(tc.inputGCPAccessLog, tc.inputIstioAccessLog)
			cs := history.NewChangeSet(l)

			_, err := (&csmAccessLogLogToTimelineMapperSetting{}).ProcessLogByGroup(t.Context(), l, cs, nil, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() failed: %v", err)
			}
			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})

	}
}
