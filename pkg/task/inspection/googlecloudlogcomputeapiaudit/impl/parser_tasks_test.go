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

package googlecloudlogcomputeapiaudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestLogToTimelineMapperTask(t *testing.T) {
	testCommonFieldSet := &log.CommonFieldSet{
		Timestamp: time.Date(2025, time.January, 1, 1, 1, 1, 1, time.UTC),
	}
	testCases := []struct {
		desc      string
		input     googlecloudcommon_contract.GCPAuditLogFieldSet
		asserters []testchangeset.ChangeSetAsserter
	}{
		{
			desc: "operation started",
			input: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "compute.instances.insert",
				ResourceName:   "projects/123/resources/abc",
				PrincipalEmail: "foobar@qux.test",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#abc#insert-op-1",
					WantRevision: history.StagingResourceRevision{
						ChangeTime: testCommonFieldSet.Timestamp,
						State:      enum.RevisionStateOperationStarted,
						Verb:       enum.RevisionVerbOperationStart,
						Requestor:  "foobar@qux.test",
					},
				},
			},
		},
		{
			desc: "operation finished",
			input: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "compute.instances.insert",
				ResourceName:   "projects/123/resources/abc",
				PrincipalEmail: "foobar@qux.test",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#abc#insert-op-1",
					WantRevision: history.StagingResourceRevision{
						ChangeTime: testCommonFieldSet.Timestamp,
						State:      enum.RevisionStateOperationFinished,
						Verb:       enum.RevisionVerbOperationFinish,
						Requestor:  "foobar@qux.test",
					},
				},
			},
		},
		{
			desc: "immediate operation",
			input: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  true,
				MethodName:     "compute.instances.delete",
				ResourceName:   "projects/123/resources/def",
				PrincipalEmail: "foobar@qux.test",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{"core/v1#node#cluster-scope#def"},
				},
			},
		},
		{
			desc: "deletion operation started",
			input: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-3",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "compute.instances.delete",
				ResourceName:   "projects/123/resources/ghi",
				PrincipalEmail: "foobar@qux.test",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#ghi#delete-op-3",
					WantRevision: history.StagingResourceRevision{
						ChangeTime: testCommonFieldSet.Timestamp,
						State:      enum.RevisionStateOperationStarted,
						Verb:       enum.RevisionVerbOperationStart,
						Requestor:  "foobar@qux.test",
					},
				},
			},
		},
		{
			desc: "deletion operation finished",
			input: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-3",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "compute.instances.delete",
				ResourceName:   "projects/123/resources/ghi",
				PrincipalEmail: "foobar@qux.test",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#ghi#delete-op-3",
					WantRevision: history.StagingResourceRevision{
						ChangeTime: testCommonFieldSet.Timestamp,
						State:      enum.RevisionStateOperationFinished,
						Verb:       enum.RevisionVerbOperationFinish,
						Requestor:  "foobar@qux.test",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(testCommonFieldSet, &tc.input)
			mapperSetting := &gcpComputeAuditLogLogToTimelineMapperSetting{}
			cs := history.NewChangeSet(l)

			_, err := mapperSetting.ProcessLogByGroup(t.Context(), l, cs, nil, struct{}{})
			if err != nil {
				t.Errorf("got error %v, want nil", err)
			}

			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}

func TestGetInstanceNameFromResourceName(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  string
	}{
		{
			desc:  "standard resource name",
			input: "projects/123/zones/us-central1-a/instances/my-instance",
			want:  "my-instance",
		},
		{
			desc:  "empty resource name",
			input: "",
			want:  "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := getInstanceNameFromResourceName(tc.input)
			if got != tc.want {
				t.Errorf("getInstanceNameFromResourceName(%q) got %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
