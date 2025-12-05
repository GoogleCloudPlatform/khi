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

package googlecloudlogk8snode_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestReadGoStructFromString(t *testing.T) {
	testCases := []struct {
		Name       string
		Input      string
		StructName string
		Expected   map[string]string
	}{
		{
			Name:       "An example RunPodSandbox log",
			Input:      "RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"",
			StructName: "PodSandboxMetadata",
			Expected: map[string]string{
				"Name":      "podname",
				"Namespace": "kube-system",
				"Attempt":   "0",
				"Uid":       "b86b49f2431d244c613996c6472eb864",
			},
		},
		{
			Name:       "An example CreateContainer log",
			Input:      "CreateContainer within sandbox \"573208ed2827243aa3db0db52e8f5a8d6fe65fcf67d93ecc76f5a4d92378af83\" for &ContainerMetadata{Name:fluentbit-gke-init,Attempt:0,} returns container id \"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256\"",
			StructName: "ContainerMetadata",
			Expected: map[string]string{
				"Attempt": "0",
				"Name":    "fluentbit-gke-init",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := readGoStructFromString(tc.Input, tc.StructName)
			if diff := cmp.Diff(tc.Expected, result); diff != "" {
				t.Errorf("result is not matching with the expected result\n%s", diff)
			}
		})
	}
}

func TestReadNextQuotedString(t *testing.T) {
	testCases := []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name:     "standard input obtained from RunPodSandbox",
			Input:    "returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"",
			Expected: "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
		},
		{
			Name:     "not containing quote",
			Input:    "foo bar",
			Expected: "",
		},
		{
			Name:     "contains single double quote",
			Input:    "\"foo bar",
			Expected: "",
		},
		{
			Name:     "contains more than 3 double quote",
			Input:    "\"foo bar\" \"qux baz\"",
			Expected: "foo bar",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			nextQuoted := readNextQuotedString(tc.Input)
			if nextQuoted != tc.Expected {
				t.Errorf("expected:%s\nactual:%s", tc.Expected, nextQuoted)
			}
		})
	}
}

func TestSlashSplittedPodNameToNamespaceAndName(t *testing.T) {
	testCases := []struct {
		desc     string
		input    string
		wantNs   string
		wantName string
		wantErr  bool
	}{
		{
			desc:     "valid format",
			input:    "kube-system/kube-dns-abcd",
			wantNs:   "kube-system",
			wantName: "kube-dns-abcd",
			wantErr:  false,
		},
		{
			desc:    "invalid format - no slash",
			input:   "kube-dns-abcd",
			wantErr: true,
		},
		{
			desc:    "invalid format - too many slashes",
			input:   "a/b/c",
			wantErr: true,
		},
		{
			desc:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			gotNs, gotName, err := slashSplittedPodNameToNamespaceAndName(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("slashSplittedPodNameToNamespaceAndName() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if gotNs != tc.wantNs || gotName != tc.wantName {
					t.Errorf("slashSplittedPodNameToNamespaceAndName() = (%v, %v), want (%v, %v)", gotNs, gotName, tc.wantNs, tc.wantName)
				}
			}
		})
	}
}

func TestToReadablePodSandboxName(t *testing.T) {
	got := toReadablePodSandboxName("ns1", "pod1")
	want := "【pod1 (Namespace: ns1)】"
	if got != want {
		t.Errorf("toReadablePodSandboxName() = %v, want %v", got, want)
	}
}

func TestToReadableContainerName(t *testing.T) {
	got := toReadableContainerName("ns1", "pod1", "container1")
	want := "【container1 (Pod: pod1, Namespace: ns1)】"
	if got != want {
		t.Errorf("toReadableContainerName() = %v, want %v", got, want)
	}
}

func TestToReadableResourceName(t *testing.T) {
	got := toReadableResourceName("core/v1", "node", "cluster-scope", "node-foo")
	want := "【node-foo (Namespace: cluster-scope, APIVersion: core/v1, Kind: node)】"
	if got != want {
		t.Errorf("toReadableResourceName() = %v, want %v", got, want)
	}
}

func TestCheckStartingAndTerminationLog(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		desc           string
		logMessage     string
		startingLog    string
		terminationLog string
		asserters      []testchangeset.ChangeSetAsserter
	}{
		{
			desc:        "starting log match",
			logMessage:  "component is starting",
			startingLog: "component is starting",
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#test-node#test-component",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateExisting,
						Requestor:  "test-component",
						ChangeTime: testTime,
					},
				},
			},
		},
		{
			desc:           "termination log match",
			logMessage:     "component is stopping",
			terminationLog: "component is stopping",
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#test-node#test-component",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleted,
						Requestor:  "test-component",
						ChangeTime: testTime,
					},
				},
			},
		},
		{
			desc:           "no match",
			logMessage:     "some other message",
			startingLog:    "starting",
			terminationLog: "stopping",
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				&googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
					Message: &logutil.ParseStructuredLogResult{
						Fields: map[string]any{
							logutil.MainMessageStructuredFieldKey: tc.logMessage,
						},
					},
					Component: "test-component",
					NodeName:  "test-node",
				},
			)
			cs := history.NewChangeSet(l)
			checkStartingAndTerminationLog(cs, l, tc.startingLog, tc.terminationLog)

			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}
