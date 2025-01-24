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

package compute_api

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	parser_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/parser"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestComputeApiParser_Parse_OperationFirstLog(t *testing.T) {
	nodeName := "gke-gke-basic-1-default-5e5b794d-2m33"
	serviceAccountName := "serviceaccount@project-id.iam.gserviceaccount.com"
	operationId := "operation-1726191739294-621f6556f5492-0777bde4-78d02b5a"
	wantLogSummary := "v1.compute.instances.insert Started"
	cs, err := parser_test.ParseFromYamlLogFile("test/logs/compute_api/operation_first.yaml", &computeAPIParser{}, nil, nil)
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	revisions := cs.GetRevisions(resourcepath.Operation(resourcepath.Node(nodeName), "insert", operationId))
	if len(revisions) != 1 {
		t.Errorf("got %d revisions, want 1", len(revisions))
	}
	gotRevision := revisions[0]
	wantRevision := &history.StagingResourceRevision{
		Verb:       enum.RevisionVerbOperationStart,
		State:      enum.RevisionStateOperationStarted,
		Requestor:  serviceAccountName,
		ChangeTime: testutil.MustParseTimeRFC3339("2024-01-01T01:00:00Z"),
		Partial:    false,
	}
	if diff := cmp.Diff(gotRevision, wantRevision, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "request", gotRevision.Body)

	nodeEvent := cs.GetEvents(resourcepath.Node(nodeName))
	if len(nodeEvent) != 1 {
		t.Errorf("got %d events, want 1", len(nodeEvent))
	}

	gotLogSummary := cs.GetLogSummary()
	if gotLogSummary != wantLogSummary {
		t.Errorf("got %q log summary, want %q", gotLogSummary, wantLogSummary)
	}
}

func TestComputeApiParser_Parse_OperationLastLog(t *testing.T) {
	nodeName := "gke-gke-basic-1-default-5e5b794d-2m33"
	serviceAccountName := "serviceaccount@project-id.iam.gserviceaccount.com"
	operationId := "operation-1726191739294-621f6556f5492-0777bde4-78d02b5a"
	wantLogSummary := "v1.compute.instances.insert Finished"
	cs, err := parser_test.ParseFromYamlLogFile("test/logs/compute_api/operation_last.yaml", &computeAPIParser{}, nil, nil)
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	revisions := cs.GetRevisions(resourcepath.Operation(resourcepath.Node(nodeName), "insert", operationId))
	if len(revisions) != 1 {
		t.Errorf("got %d revisions, want 1", len(revisions))
	}
	gotRevision := revisions[0]
	wantRevision := &history.StagingResourceRevision{
		Verb:       enum.RevisionVerbOperationFinish,
		State:      enum.RevisionStateOperationFinished,
		Requestor:  serviceAccountName,
		ChangeTime: testutil.MustParseTimeRFC3339("2024-01-01T01:05:00Z"),
		Partial:    false,
	}
	if diff := cmp.Diff(gotRevision, wantRevision, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
	testutil.VerifyWithGolden(t, "request", gotRevision.Body)

	nodeEvent := cs.GetEvents(resourcepath.Node(nodeName))
	if len(nodeEvent) != 1 {
		t.Errorf("got %d events, want 1", len(nodeEvent))
	}

	gotLogSummary := cs.GetLogSummary()
	if gotLogSummary != wantLogSummary {
		t.Errorf("got %q log summary, want %q", gotLogSummary, wantLogSummary)
	}
}
