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

package commonlogk8saudit_contract

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
)

func TestIterateMultiGroupLog(t *testing.T) {
	t1 := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Minute)
	t3 := t1.Add(2 * time.Minute)
	t4 := t1.Add(3 * time.Minute)

	logA1 := log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{Timestamp: t1})
	logB1 := log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{Timestamp: t2})
	logA2 := log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{Timestamp: t3})
	logC1 := log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{Timestamp: t4})

	groupSet := RelatedGroupSet{
		Roles: map[string]*ResourceManifestLogGroup{
			"roleA": {
				Logs: []*ResourceManifestLog{
					{Log: logA1},
					{Log: logA2},
				},
			},
			"roleB": {
				Logs: []*ResourceManifestLog{
					{Log: logB1},
				},
			},
			"roleC": {
				Logs: []*ResourceManifestLog{
					{Log: logC1},
				},
			},
		},
	}

	var gotEvents []struct {
		Role string
		Time time.Time
	}

	for event := range iterateMultiGroupLog(groupSet) {
		commonSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
		gotEvents = append(gotEvents, struct {
			Role string
			Time time.Time
		}{
			Role: event.GroupRole,
			Time: commonSet.Timestamp,
		})
	}

	wantEvents := []struct {
		Role string
		Time time.Time
	}{
		{Role: "roleA", Time: t1},
		{Role: "roleB", Time: t2},
		{Role: "roleA", Time: t3},
		{Role: "roleC", Time: t4},
	}

	if diff := cmp.Diff(wantEvents, gotEvents); diff != "" {
		t.Errorf("iterateMultiGroupLog() mismatch (-want +got):\n%s", diff)
	}
}

func TestGetLastBody(t *testing.T) {
	t1 := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Minute)
	t3 := t1.Add(2 * time.Minute)

	nodeA1, _ := structured.FromGoValue(map[string]any{"value": "A1"}, &structured.AlphabeticalGoMapKeyOrderProvider{})
	nodeB1, _ := structured.FromGoValue(map[string]any{"value": "B1"}, &structured.AlphabeticalGoMapKeyOrderProvider{})
	nodeA2, _ := structured.FromGoValue(map[string]any{"value": "A2"}, &structured.AlphabeticalGoMapKeyOrderProvider{})

	logA1 := log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{Timestamp: t1})
	logB1 := log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{Timestamp: t2})
	logA2 := log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{Timestamp: t3})

	groupSet := RelatedGroupSet{
		Roles: map[string]*ResourceManifestLogGroup{
			"roleA": {
				Logs: []*ResourceManifestLog{
					{Log: logA1, ResourceBodyYAML: "value: A1", ResourceBodyReader: structured.NewNodeReader(nodeA1)},
					{Log: logA2, ResourceBodyYAML: "value: A2", ResourceBodyReader: structured.NewNodeReader(nodeA2)},
				},
			},
			"roleB": {
				Logs: []*ResourceManifestLog{
					{Log: logB1, ResourceBodyYAML: "value: B1", ResourceBodyReader: structured.NewNodeReader(nodeB1)},
				},
			},
		},
	}

	events := make([]MultiGroupLogEvent, 0)
	for event := range iterateMultiGroupLog(groupSet) {
		events = append(events, event)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	testCases := []struct {
		name         string
		eventIndex   int
		expectedRole string
		roleToCheck  string
		wantFound    bool
		wantYAML     string
		wantValue    string
	}{
		{
			name:         "event 0: check roleA body",
			eventIndex:   0,
			expectedRole: "roleA",
			roleToCheck:  "roleA",
			wantFound:    true,
			wantYAML:     "value: A1",
			wantValue:    "A1",
		},
		{
			name:         "event 0: check roleB body (should not exist)",
			eventIndex:   0,
			expectedRole: "roleA",
			roleToCheck:  "roleB",
			wantFound:    false,
		},
		{
			name:         "event 1: check roleA body",
			eventIndex:   1,
			expectedRole: "roleB",
			roleToCheck:  "roleA",
			wantFound:    true,
			wantYAML:     "value: A1",
			wantValue:    "A1",
		},
		{
			name:         "event 1: check roleB body",
			eventIndex:   1,
			expectedRole: "roleB",
			roleToCheck:  "roleB",
			wantFound:    true,
			wantYAML:     "value: B1",
			wantValue:    "B1",
		},
		{
			name:         "event 2: check roleA body",
			eventIndex:   2,
			expectedRole: "roleA",
			roleToCheck:  "roleA",
			wantFound:    true,
			wantYAML:     "value: A2",
			wantValue:    "A2",
		},
		{
			name:         "event 2: check roleB body",
			eventIndex:   2,
			expectedRole: "roleA",
			roleToCheck:  "roleB",
			wantFound:    true,
			wantYAML:     "value: B1",
			wantValue:    "B1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.eventIndex >= len(events) {
				t.Fatalf("eventIndex %d out of range", tc.eventIndex)
			}
			e := events[tc.eventIndex]

			if e.GroupRole != tc.expectedRole {
				t.Errorf("expected group role %q, got %q", tc.expectedRole, e.GroupRole)
			}

			yaml, ok := e.GetLastBodyYAML(tc.roleToCheck)
			if ok != tc.wantFound {
				t.Errorf("GetLastBodyYAML(%q) ok = %t, want %t", tc.roleToCheck, ok, tc.wantFound)
			}
			if ok && yaml != tc.wantYAML {
				t.Errorf("GetLastBodyYAML(%q) = %q, want %q", tc.roleToCheck, yaml, tc.wantYAML)
			}

			if tc.wantFound {
				reader, ok := e.GetLastBodyReader(tc.roleToCheck)
				if !ok || reader == nil {
					t.Errorf("GetLastBodyReader(%q) failed, want success", tc.roleToCheck)
				} else if tc.wantValue != "" {
					val, err := reader.ReadString("value")
					if err != nil {
						t.Errorf("failed to read string \"value\" from NodeReader: %v", err)
					} else if val != tc.wantValue {
						t.Errorf("NodeReader value = %q, want %q", val, tc.wantValue)
					}
				}
			} else {
				_, ok := e.GetLastBodyReader(tc.roleToCheck)
				if ok {
					t.Errorf("GetLastBodyReader(%q) returned true, want false", tc.roleToCheck)
				}
			}
		})
	}
}
