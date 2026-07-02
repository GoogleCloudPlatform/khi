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

package commonlogk8saudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestNonSuccessLogLogToTimelineMapperTaskSetting_ProcessLogByGroup(t *testing.T) {
	// 1. Set up the mock Builder and construct comparison paths hierarchically.
	builder := khifilev6.NewBuilder()
	cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})
	api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
	kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
	ns := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})

	parentPath := builder.TimelineAccumulator.GetPath(ns, khifilev6.PathSegment{Name: "test-pod", Type: inspectioncore_contract.TimelineTypeResource})
	otherSubresourcePath := builder.TimelineAccumulator.GetPath(parentPath, khifilev6.PathSegment{Name: "proxy", Type: inspectioncore_contract.TimelineTypeSubresource})

	testCases := []struct {
		name            string
		subresourceName string
		wantPath        *khifilev6.TimelinePath
	}{
		{
			name:            "standard pod mapping",
			subresourceName: "",
			wantPath:        parentPath,
		},
		{
			name:            "status subresource mapped to parent",
			subresourceName: "status",
			wantPath:        parentPath,
		},
		{
			name:            "non-status subresource proxy NOT mapped to parent",
			subresourceName: "proxy",
			wantPath:        otherSubresourcePath,
		},
	}

	mapperSetting := &nonSuccessLogLogToTimelineMapperTaskSetting{
		subresourceMapToWriteToParent: map[string]struct{}{
			"status":   {},
			"finalize": {},
			"approve":  {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
				APIVersion:      "core/v1",
				PluralKind:      "pods",
				Namespace:       "default",
				ResourceName:    "test-pod",
				SubresourceName: tc.subresourceName,
				ClusterName:     "k8s",
			}
			logObj := log.NewLogWithFieldSetsForTest(fs, &log.CommonFieldSet{Timestamp: time.Now()})
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

			cs, _, err := mapperSetting.ProcessLogByGroup(ctx, logObj, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() failed: %v", err)
			}

			testchangeset.AssertTimeline(t, cs).
				HasEvent(tc.wantPath)
		})
	}
}
