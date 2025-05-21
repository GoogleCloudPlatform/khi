// Copyright 2024 Google LLC
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

package gke_audit

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	parser_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/parser"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestGkeAuditLogParser_ClusterCreationStartLog(t *testing.T) {
	userAccountName := "user@example.com"
	operationId := "operation-1726191072114-d3db4945-ad7b-4fff-aff7-55a867e4bc54"

	cs, err := parser_test.ParseFromYamlLogFile(
		"test/logs/gke_audit/cluster_creation_started.yaml",
		&gkeAuditLogParser{},
		nil, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	gotRevisions := cs.GetRevisions(resourcepath.Cluster("gke-basic-1"))
	wantRevisions := []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbCreate,
			State:      enum.RevisionStateProvisioning,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2024-01-01T01:05:00Z"),
		},
	}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "cluster-body", gotRevisions[0].Body)

	gotRevisions = cs.GetRevisions(resourcepath.Operation(resourcepath.Cluster("gke-basic-1"), "CreateCluster", operationId))
	wantRevisions = []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbOperationStart,
			State:      enum.RevisionStateOperationStarted,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2024-01-01T01:05:00Z"),
		},
	}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "operation-body", gotRevisions[0].Body)
}

func TestGkeAuditLogParser_ClusterCreationFinishedLog(t *testing.T) {
	userAccountName := "user@example.com"
	operationId := "operation-1726191072114-d3db4945-ad7b-4fff-aff7-55a867e4bc54"

	cs, err := parser_test.ParseFromYamlLogFile(
		"test/logs/gke_audit/cluster_creation_started.yaml",
		&gkeAuditLogParser{}, nil, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	gotRevisions := cs.GetRevisions(resourcepath.Cluster("gke-basic-1"))
	wantRevision := []*history.StagingResourceRevision{{
		Verb:       enum.RevisionVerbCreate,
		State:      enum.RevisionStateProvisioning,
		Requestor:  userAccountName,
		ChangeTime: testutil.MustParseTimeRFC3339("2024-01-01T01:05:00Z"),
	}}
	if diff := cmp.Diff(gotRevisions, wantRevision, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "cluster-body", gotRevisions[0].Body)

	gotRevisions = cs.GetRevisions(resourcepath.Operation(resourcepath.Cluster("gke-basic-1"), "CreateCluster", operationId))
	wantRevision = []*history.StagingResourceRevision{{
		Verb:       enum.RevisionVerbOperationStart,
		State:      enum.RevisionStateOperationStarted,
		Requestor:  userAccountName,
		ChangeTime: testutil.MustParseTimeRFC3339("2024-01-01T01:05:00Z"),
	}}
	if diff := cmp.Diff(gotRevisions, wantRevision, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "operation-body", gotRevisions[0].Body)
}

func TestGkeAuditLogParser_ClusterDeletionStartLog(t *testing.T) {
	userAccountName := "user@example.com"
	operationId := "operation-1726199159930-7409b104-8654-4667-b477-4ce504d09bea"

	cs, err := parser_test.ParseFromYamlLogFile(
		"test/logs/gke_audit/cluster_deletion_started.yaml",
		&gkeAuditLogParser{},
		nil, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	gotRevisions := cs.GetRevisions(resourcepath.Cluster("gke-basic-1"))
	wantRevisions := []*history.StagingResourceRevision{
		{
			State:      enum.RevisionStateDeleting,
			Verb:       enum.RevisionVerbDelete,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "cluster-body", gotRevisions[0].Body)

	gotRevisions = cs.GetRevisions(resourcepath.Operation(resourcepath.Cluster("gke-basic-1"), "DeleteCluster", operationId))
	wantRevisions = []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbOperationStart,
			State:      enum.RevisionStateOperationStarted,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "operation-body", gotRevisions[0].Body)
}

func TestGkeAuditLogParser_ClusterDeletionFinishedLog(t *testing.T) {
	operationId := "operation-1726199159930-7409b104-8654-4667-b477-4ce504d09bea"
	userAccountName := "unknown"
	cs, err := parser_test.ParseFromYamlLogFile(
		"test/logs/gke_audit/cluster_deletion_finished.yaml",
		&gkeAuditLogParser{},
		nil, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	gotRevisions := cs.GetRevisions(resourcepath.Cluster("gke-basic-1"))
	wantRevisions := []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbDelete,
			State:      enum.RevisionStateDeleted,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "cluster-body", gotRevisions[0].Body)

	gotRevisions = cs.GetRevisions(resourcepath.Operation(resourcepath.Cluster("gke-basic-1"), "DeleteCluster", operationId))
	wantRevisions = []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbOperationFinish,
			State:      enum.RevisionStateOperationFinished,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "operation-body", gotRevisions[0].Body)
}

func TestGkeAuditLogParser_NodepoolCreationStartLog(t *testing.T) {
	userAccountName := "user@example.com"
	operationId := "operation-1726191716103-f4072772-f902-453d-8776-b69047cebae6"

	cs, err := parser_test.ParseFromYamlLogFile(
		"test/logs/gke_audit/nodepool_creation_started.yaml",
		&gkeAuditLogParser{},
		nil, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	gotRevisions := cs.GetRevisions(resourcepath.Nodepool("gke-basic-1", "default"))
	wantRevisions := []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbCreate,
			State:      enum.RevisionStateProvisioning,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "nodepool-body", gotRevisions[0].Body)

	gotRevisions = cs.GetRevisions(resourcepath.Operation(resourcepath.Nodepool("gke-basic-1", "default"), "CreateNodePool", operationId))
	wantRevisions = []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbOperationStart,
			State:      enum.RevisionStateOperationStarted,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "operation-body", gotRevisions[0].Body)
}

func TestGkeAuditLogParser_NodepoolCreationFinishedLog(t *testing.T) {
	userAccountName := "unknown"
	operationId := "operation-1726191716103-f4072772-f902-453d-8776-b69047cebae6"

	cs, err := parser_test.ParseFromYamlLogFile(
		"test/logs/gke_audit/nodepool_creation_finished.yaml",
		&gkeAuditLogParser{},
		nil, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	gotRevisions := cs.GetRevisions(resourcepath.Nodepool("gke-basic-1", "default"))
	wantRevisions := []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbCreate,
			State:      enum.RevisionStateExisting,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "nodepool-body", gotRevisions[0].Body)

	gotRevisions = cs.GetRevisions(resourcepath.Operation(resourcepath.Nodepool("gke-basic-1", "default"), "CreateNodePool", operationId))
	wantRevisions = []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbOperationFinish,
			State:      enum.RevisionStateOperationFinished,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "operation-body", gotRevisions[0].Body)
}

func TestGkeAuditLogParser_NodepoolDeletionStartLog(t *testing.T) {
	userAccountName := "user@example.com"
	operationId := "operation-1726191433631-f35aa16e-345f-4a0f-8091-ec613f0635c2"

	cs, err := parser_test.ParseFromYamlLogFile(
		"test/logs/gke_audit/nodepool_deletion_started.yaml",
		&gkeAuditLogParser{},
		nil, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	gotRevisions := cs.GetRevisions(resourcepath.Nodepool("gke-basic-1", "default-pool"))
	wantRevisions := []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbDelete,
			State:      enum.RevisionStateDeleting,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "nodepool-body", gotRevisions[0].Body)

	gotRevisions = cs.GetRevisions(resourcepath.Operation(resourcepath.Nodepool("gke-basic-1", "default-pool"), "DeleteNodePool", operationId))
	wantRevisions = []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbOperationStart,
			State:      enum.RevisionStateOperationStarted,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "operation-body", gotRevisions[0].Body)
}

func TestGkeAuditLogParser_NodepoolDeletionFinishedLog(t *testing.T) {
	userAccountName := "unknown"
	operationId := "operation-1726191433631-f35aa16e-345f-4a0f-8091-ec613f0635c2"

	cs, err := parser_test.ParseFromYamlLogFile(
		"test/logs/gke_audit/nodepool_deletion_finished.yaml",
		&gkeAuditLogParser{},
		nil, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	gotRevisions := cs.GetRevisions(resourcepath.Nodepool("gke-basic-1", "default-pool"))
	wantRevisions := []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbDelete,
			State:      enum.RevisionStateDeleted,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "nodepool-body", gotRevisions[0].Body)

	gotRevisions = cs.GetRevisions(resourcepath.Operation(resourcepath.Nodepool("gke-basic-1", "default-pool"), "DeleteNodePool", operationId))
	wantRevisions = []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbOperationFinish,
			State:      enum.RevisionStateOperationFinished,
			Requestor:  userAccountName,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-01-01T00:00:00Z"),
		}}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "operation-body", gotRevisions[0].Body)
}

func TestGkeAuditLogParser_ClusterCreationWithErrorLog(t *testing.T) {
	clusterName := "p0-gke-basic-1"
	cs, err := parser_test.ParseFromYamlLogFile(
		"test/logs/gke_audit/cluster_creation_started_with_error.yaml",
		&gkeAuditLogParser{},
		nil, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	gotRevisions := cs.GetRevisions(resourcepath.Cluster(clusterName))
	if gotRevisions != nil {
		t.Errorf("got revision %v, want nil", gotRevisions)
	}
	gotEvents := cs.GetEvents(resourcepath.Cluster(clusterName))
	if len(gotEvents) != 1 {
		t.Errorf("got event count %d, want 1", len(gotEvents))
	}
}
