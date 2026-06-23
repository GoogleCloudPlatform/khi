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

package googlecloudcommon_contract

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestGCPOperationTracker_ProcessOperationLog(t *testing.T) {
	testTime := time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC)
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	parentPath := MustGCPProjectTimeline(ctx, "test-project")
	targetPath := MustGCPOperationTimeline(ctx, parentPath, "insert", "op-1")

	t.Run("immediate operation adds event", func(t *testing.T) {
		tracker := NewGCPOperationTracker()
		dummyLog := &log.Log{}
		cs := khifilev6.NewTimelineChangeSet(dummyLog)
		audit := &GCPAuditLogFieldSet{
			MethodName:     "compute.instances.insert",
			OperationFirst: true,
			OperationLast:  true,
		}

		tracker.ProcessOperationLog(ctx, cs, targetPath, audit, testTime)

		testchangeset.AssertTimeline(t, cs).
			HasEvent(targetPath)
	})

	t.Run("long running operation normal flow", func(t *testing.T) {
		tracker := NewGCPOperationTracker()
		dummyLog := &log.Log{}
		csStart := khifilev6.NewTimelineChangeSet(dummyLog)
		auditStart := &GCPAuditLogFieldSet{
			MethodName:     "compute.instances.insert",
			OperationID:    "op-1",
			OperationFirst: true,
			OperationLast:  false,
		}

		tracker.ProcessOperationLog(ctx, csStart, targetPath, auditStart, testTime)

		testchangeset.AssertTimeline(t, csStart).
			HasRevision(targetPath, &khifilev6.StagingRevision{
				VerbType:    VerbOperationStart,
				StateType:   RevisionStateOperationStarted,
				ChangedTime: testTime,
			})

		csFinish := khifilev6.NewTimelineChangeSet(dummyLog)
		auditFinish := &GCPAuditLogFieldSet{
			MethodName:     "compute.instances.insert",
			OperationID:    "op-1",
			OperationFirst: false,
			OperationLast:  true,
			Status:         0,
		}

		tracker.ProcessOperationLog(ctx, csFinish, targetPath, auditFinish, testTime.Add(time.Minute))

		testchangeset.AssertTimeline(t, csFinish).
			HasRevision(targetPath, &khifilev6.StagingRevision{
				VerbType:    VerbOperationFinish,
				StateType:   RevisionStateOperationSucceed,
				ChangedTime: testTime.Add(time.Minute),
			})
	})

	t.Run("long running operation start log missing flow", func(t *testing.T) {
		tracker := NewGCPOperationTracker()
		dummyLog := &log.Log{}
		csFinish := khifilev6.NewTimelineChangeSet(dummyLog)
		auditFinish := &GCPAuditLogFieldSet{
			MethodName:     "compute.instances.insert",
			OperationID:    "op-2",
			OperationFirst: false,
			OperationLast:  true,
			Status:         0,
		}

		tracker.ProcessOperationLog(ctx, csFinish, targetPath, auditFinish, testTime)

		testchangeset.AssertTimeline(t, csFinish).
			HasRevision(targetPath, &khifilev6.StagingRevision{
				VerbType:    VerbOperationStart,
				StateType:   RevisionStateOperationStartedLogNotFound,
				ChangedTime: time.Unix(0, 0),
			}).
			HasRevision(targetPath, &khifilev6.StagingRevision{
				VerbType:    VerbOperationFinish,
				StateType:   RevisionStateOperationSucceed,
				ChangedTime: testTime,
			})
	})

	t.Run("long running operation failed flow", func(t *testing.T) {
		tracker := NewGCPOperationTracker()
		dummyLog := &log.Log{}
		csFinish := khifilev6.NewTimelineChangeSet(dummyLog)
		auditFinish := &GCPAuditLogFieldSet{
			MethodName:     "compute.instances.insert",
			OperationID:    "op-3",
			OperationFirst: false,
			OperationLast:  true,
			Status:         13,
		}

		tracker.ProcessOperationLog(ctx, csFinish, targetPath, auditFinish, testTime)

		testchangeset.AssertTimeline(t, csFinish).
			HasRevision(targetPath, &khifilev6.StagingRevision{
				VerbType:    VerbOperationFinish,
				StateType:   RevisionStateOperationFailed,
				ChangedTime: testTime,
			})
	})

	t.Run("long running operation request body inclusion", func(t *testing.T) {
		tracker := NewGCPOperationTracker()
		dummyLog := &log.Log{}
		cs := khifilev6.NewTimelineChangeSet(dummyLog)
		dummyNode := structured.NewStandardScalarNode("test-body")
		audit := &GCPAuditLogFieldSet{
			MethodName:     "compute.instances.insert",
			OperationID:    "op-4",
			OperationFirst: true,
			Request:        structured.NewNodeReader(dummyNode),
		}

		tracker.ProcessOperationLog(ctx, cs, targetPath, audit, testTime)

		testchangeset.AssertTimeline(t, cs).
			HasRevision(targetPath, &khifilev6.StagingRevision{
				ResourceBody: dummyNode,
				VerbType:     VerbOperationStart,
				StateType:    RevisionStateOperationStarted,
				ChangedTime:  testTime,
			}, cmp.AllowUnexported(structured.StandardScalarNode[string]{}))
	})
}
