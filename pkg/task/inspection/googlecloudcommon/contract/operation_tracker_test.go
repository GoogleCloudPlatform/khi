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
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
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

func TestProcessGCPClusterNodepoolOperationLog(t *testing.T) {
	testTime := time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC)
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	parentPath := MustGCPProjectTimeline(ctx, "test-project")
	targetTimeline := MustGKEClusterTimeline(ctx, parentPath, "test-cluster")
	operationTimeline := MustGCPOperationTimeline(ctx, targetTimeline, "CreateCluster", "op-1")

	t.Run("creation start log adds provisioning revision", func(t *testing.T) {
		tracker := NewGCPOperationTracker()
		dummyLog := &log.Log{}
		cs := khifilev6.NewTimelineChangeSet(dummyLog)
		audit := &GCPAuditLogFieldSet{
			OperationID:    "op-1",
			OperationFirst: true,
		}
		common := &log.CommonFieldSet{
			Timestamp: testTime,
		}

		ProcessGCPClusterNodepoolOperationLog(ctx, cs, tracker, targetTimeline, operationTimeline, audit, common, "CreateCluster", true)

		testchangeset.AssertTimeline(t, cs).
			HasRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbCreate,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sClusterProvisioning,
				ChangedTime: testTime,
			})
	})

	t.Run("creation finish log without start log prepends provisioning log not found at unix 0", func(t *testing.T) {
		tracker := NewGCPOperationTracker()
		dummyLog := &log.Log{}
		cs := khifilev6.NewTimelineChangeSet(dummyLog)
		audit := &GCPAuditLogFieldSet{
			OperationID:   "op-1",
			OperationLast: true,
		}
		common := &log.CommonFieldSet{
			Timestamp: testTime,
		}

		ProcessGCPClusterNodepoolOperationLog(ctx, cs, tracker, targetTimeline, operationTimeline, audit, common, "CreateCluster", true)

		testchangeset.AssertTimeline(t, cs).
			HasRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbCreate,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sClusterProvisioningLogNotFound,
				ChangedTime: time.Unix(0, 0),
			}).
			HasRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbCreate,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sClusterExisting,
				ChangedTime: testTime,
			})
	})

	t.Run("deletion start log without prior resource revision prepends existing log not found at unix 0", func(t *testing.T) {
		tracker := NewGCPOperationTracker()
		dummyLog := &log.Log{}
		cs := khifilev6.NewTimelineChangeSet(dummyLog)
		audit := &GCPAuditLogFieldSet{
			OperationID:    "op-2",
			OperationFirst: true,
		}
		common := &log.CommonFieldSet{
			Timestamp: testTime,
		}

		ProcessGCPClusterNodepoolOperationLog(ctx, cs, tracker, targetTimeline, operationTimeline, audit, common, "DeleteCluster", true)

		testchangeset.AssertTimeline(t, cs).
			HasRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbCreate,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sClusterExistingLogNotFound,
				ChangedTime: time.Unix(0, 0),
			}).
			HasRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbDelete,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sClusterDeleting,
				ChangedTime: testTime,
			})
	})

	t.Run("deletion finish log without start log prepends deleting log not found at unix 0", func(t *testing.T) {
		tracker := NewGCPOperationTracker()
		dummyLog := &log.Log{}
		cs := khifilev6.NewTimelineChangeSet(dummyLog)
		audit := &GCPAuditLogFieldSet{
			OperationID:   "op-2",
			OperationLast: true,
		}
		common := &log.CommonFieldSet{
			Timestamp: testTime,
		}

		ProcessGCPClusterNodepoolOperationLog(ctx, cs, tracker, targetTimeline, operationTimeline, audit, common, "DeleteCluster", true)

		testchangeset.AssertTimeline(t, cs).
			HasRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbDelete,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sClusterDeletingLogNotFound,
				ChangedTime: time.Unix(0, 0),
			}).
			HasRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbDelete,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sClusterDeleted,
				ChangedTime: testTime,
			})
	})

	t.Run("nodepool creation finish log without start log prepends nodepool provisioning log not found at unix 0", func(t *testing.T) {
		tracker := NewGCPOperationTracker()
		dummyLog := &log.Log{}
		cs := khifilev6.NewTimelineChangeSet(dummyLog)
		audit := &GCPAuditLogFieldSet{
			OperationID:   "op-1",
			OperationLast: true,
		}
		common := &log.CommonFieldSet{
			Timestamp: testTime,
		}

		ProcessGCPClusterNodepoolOperationLog(ctx, cs, tracker, targetTimeline, operationTimeline, audit, common, "CreateNodePool", false)

		testchangeset.AssertTimeline(t, cs).
			HasRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbCreate,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sNodepoolProvisioningLogNotFound,
				ChangedTime: time.Unix(0, 0),
			}).
			HasRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbCreate,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sNodepoolExisting,
				ChangedTime: testTime,
			})
	})

	t.Run("nodepool deletion finish log without start log prepends nodepool deleting log not found at unix 0", func(t *testing.T) {
		tracker := NewGCPOperationTracker()
		dummyLog := &log.Log{}
		cs := khifilev6.NewTimelineChangeSet(dummyLog)
		audit := &GCPAuditLogFieldSet{
			OperationID:   "op-2",
			OperationLast: true,
		}
		common := &log.CommonFieldSet{
			Timestamp: testTime,
		}

		ProcessGCPClusterNodepoolOperationLog(ctx, cs, tracker, targetTimeline, operationTimeline, audit, common, "DeleteNodePool", false)

		testchangeset.AssertTimeline(t, cs).
			HasRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbDelete,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sNodepoolDeletingLogNotFound,
				ChangedTime: time.Unix(0, 0),
			}).
			HasRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbDelete,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sNodepoolDeleted,
				ChangedTime: testTime,
			})
	})
}
