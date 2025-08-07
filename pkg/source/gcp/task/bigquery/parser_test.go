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

package bigquery

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	parser_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/parser"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParseSuccessBigQueryJob(t *testing.T) {
	cs, err := parser_test.ParseFromYamlLogFile(
		"test/logs/bigquery/single.yaml",
		&bigqueryJobParser{},
		nil)
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	requester := "requester"
	gotRevisions := cs.GetRevisions((&BigQueryJob{Name: BigqueryJobName{JobId: "bquxjob_44832c55_195ea1bc54b", ProjectId: "project-id", Location: "US"}, Statistics: BigQueryJobStatistics{Reservation: "unreserved"}}).ToResourcePath())
	wantRevisions := []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbBigQuryJobCreate,
			State:      enum.RevisionStateBigQueryJobPending,
			Requestor:  requester,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-03-31T02:50:44.192Z"),
		},
		{
			Verb:       enum.RevisionVerbBigQuryJobStart,
			State:      enum.RevisionStateBigQueryJobRunning,
			Requestor:  requester,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-03-31T02:50:44.290Z"),
		},
		{
			Verb:       enum.RevisionVerbBigQuryJobDone,
			State:      enum.RevisionStateBigQueryJobSuccess,
			Requestor:  requester,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-03-31T02:50:44.612Z"),
		},
	}
	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
}

func TestParseFailedBVigQueryJob(t *testing.T) {
	cs, err := parser_test.ParseFromYamlLogFile(
		"test/logs/bigquery/internal_error.yaml",
		&bigqueryJobParser{},
		nil)
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	requester := "requester"
	jobPath := (&BigQueryJob{
		Name: BigqueryJobName{
			JobId:     "scheduled_query_67fe258f-0000-29b2-818e-2405888306ac",
			ProjectId: "project-id",
			Location:  "US",
		},
		Statistics: BigQueryJobStatistics{
			Reservation: "unreserved",
		},
	}).ToResourcePath()

	gotRevisions := cs.GetRevisions(jobPath)
	wantRevisions := []*history.StagingResourceRevision{
		{
			Verb:       enum.RevisionVerbBigQuryJobCreate,
			State:      enum.RevisionStateBigQueryJobPending,
			Requestor:  requester,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-03-18T13:01:01.665Z"),
		},
		{
			Verb:       enum.RevisionVerbBigQuryJobStart,
			State:      enum.RevisionStateBigQueryJobRunning,
			Requestor:  requester,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-03-18T13:01:02.015Z"),
		},
		{
			Verb:       enum.RevisionVerbBigQuryJobDone,
			State:      enum.RevisionStateBigQueryJobFailed, // This should be Failed state
			Requestor:  requester,
			ChangeTime: testutil.MustParseTimeRFC3339("2025-03-18T13:01:03.571Z"),
		},
	}

	if diff := cmp.Diff(gotRevisions, wantRevisions, cmpopts.IgnoreFields(history.StagingResourceRevision{}, "Body")); diff != "" {
		t.Errorf("got revision mismatch (-want +got):\n%s", diff)
	}
}
