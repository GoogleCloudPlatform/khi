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

package googlecloudlogserialport_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogserialport_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogserialport/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestSerialPortLogIngester_ProcessLog(t *testing.T) {
	testTime := time.Date(2025, 9, 29, 6, 39, 24, 0, time.UTC)
	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name: "successful log ingestion",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityError,
				},
				&googlecloudlogserialport_contract.GCESerialPortLogFieldSet{
					Message:  "foo payload",
					NodeName: "node-name-bar",
					Port:     "serial_port_output_qux",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("foo payload").
					HasTimestamp(testTime).
					HasSeverity(inspectioncore_contract.SeverityError).
					HasLogType(googlecloudlogserialport_contract.LogTypeSerialPort)
			},
		},
	}

	ingester := &serialPortLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			cs, err := ingester.ProcessLog(ctx, tc.input)
			if err != nil {
				t.Fatalf("ProcessLog() returned unexpected error: %v", err)
			}
			tc.assert(t, cs)
		})
	}
}

func TestSerialPortLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewBuilder()

	clusterTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: inspectioncore_contract.TimelineTypeK8sCluster,
	})
	apiVersionTimeline := builder.TimelineAccumulator.GetPath(clusterTimeline, khifilev6.PathSegment{
		Name: "core/v1",
		Type: inspectioncore_contract.TimelineTypeAPIVersion,
	})
	kindTimeline := builder.TimelineAccumulator.GetPath(apiVersionTimeline, khifilev6.PathSegment{
		Name: "node",
		Type: inspectioncore_contract.TimelineTypeKind,
	})
	namespaceTimeline := builder.TimelineAccumulator.GetPath(kindTimeline, khifilev6.PathSegment{
		Name: "cluster-scope",
		Type: inspectioncore_contract.TimelineTypeNamespace,
	})
	nodeTimeline := builder.TimelineAccumulator.GetPath(namespaceTimeline, khifilev6.PathSegment{
		Name: "node-name-bar",
		Type: inspectioncore_contract.TimelineTypeResource,
	})
	wantSerialPortPath := builder.TimelineAccumulator.GetPath(nodeTimeline, khifilev6.PathSegment{
		Name: "serial_port_output_qux",
		Type: googlecloudlogserialport_contract.TimelineTypeSerialPort,
	})

	testCases := []struct {
		name     string
		inputLog *log.Log
		assert   func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "create timeline event",
			inputLog: log.NewLogWithFieldSetsForTest(
				&googlecloudlogserialport_contract.GCESerialPortLogFieldSet{
					Message:  "foo payload",
					NodeName: "node-name-bar",
					Port:     "serial_port_output_qux",
				},
			),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantSerialPortPath)
			},
		},
	}

	mapper := &serialportLogToTimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(), googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			})

			cs, _, err := mapper.ProcessLogByGroup(ctx, tc.inputLog, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}

			tc.assert(t, ctx, cs)
		})
	}
}
