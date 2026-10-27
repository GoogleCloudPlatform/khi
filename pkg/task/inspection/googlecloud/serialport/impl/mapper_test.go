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

package serialport_impl

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/serialport"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

func TestCgroupPodUIDPatternFinderTask(t *testing.T) {
	podIdentity := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Namespace:  "default",
		Name:       "test-pod",
	}
	nodeIdentity := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "node",
		Name:       "test-node",
	}

	testCases := []struct {
		name      string
		uidMap    k8saudit.UIDToResourceIdentity
		query     string
		wantMatch patternfinder.PatternMatchResult[*k8saudit.ResourceIdentity]
		wantFound bool
	}{
		{
			name: "matches underscore-separated Pod UID converted from canonical UID",
			uidMap: k8saudit.UIDToResourceIdentity{
				"8b04c24f-b9f7-49dd-ab67-f55568b07817": podIdentity,
			},
			query: "8b04c24f_b9f7_49dd_ab67_f55568b07817.slice",
			wantMatch: patternfinder.PatternMatchResult[*k8saudit.ResourceIdentity]{
				Value: podIdentity,
				Start: 0,
				End:   36,
			},
			wantFound: true,
		},
		{
			name: "does not match canonical hyphen-separated UID",
			uidMap: k8saudit.UIDToResourceIdentity{
				"8b04c24f-b9f7-49dd-ab67-f55568b07817": podIdentity,
			},
			query:     "8b04c24f-b9f7-49dd-ab67-f55568b07817/suffix",
			wantMatch: patternfinder.PatternMatchResult[*k8saudit.ResourceIdentity]{},
			wantFound: false,
		},
		{
			name: "ignores non-Pod resources",
			uidMap: k8saudit.UIDToResourceIdentity{
				"d3a7ae3c-7108-4ab2-ae02-b6d7ed7a21d8": nodeIdentity,
			},
			query:     "d3a7ae3c_7108_4ab2_ae02_b6d7ed7a21d8",
			wantMatch: patternfinder.PatternMatchResult[*k8saudit.ResourceIdentity]{},
			wantFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			finder, _, err := inspectiontest.Run(t, ctx, cgroupPodUIDPatternFinderTask, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.Given(k8saudit.ResourceUIDInventoryTaskID.Ref(), tc.uidMap),
			)
			if err != nil {
				t.Fatalf("inspectiontest.Run() failed: %v", err)
			}

			gotMatch, gotFound := finder.Match(tc.query)
			if gotFound != tc.wantFound {
				t.Fatalf("finder.Match(%q) found = %v, want %v", tc.query, gotFound, tc.wantFound)
			}
			if diff := cmp.Diff(tc.wantMatch, gotMatch); diff != "" {
				t.Errorf("finder.Match(%q) mismatch (-want +got):\n%s", tc.query, diff)
			}
		})
	}
}

func TestFindAllUIDTokens(t *testing.T) {
	const (
		uidUnderscore = "8b04c24f_b9f7_49dd_ab67_f55568b07817"
		uidStartingD  = "d3a7ae3c_7108_4ab2_ae02_b6d7ed7a21d8"
	)
	finder := patternfinder.NewRadixPatternFinder[string]()
	_ = finder.AddPattern(uidUnderscore, "pod-a")
	_ = finder.AddPattern(uidStartingD, "pod-d")

	cgroupLine := "[100.000000] Memory cgroup stats for /kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod" + uidUnderscore + ".slice/cri-containerd-fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256.scope:"
	cgroupIdx := strings.Index(cgroupLine, uidUnderscore)

	oomLine := "[100.100000] oom-kill:constraint=CONSTRAINT_MEMCG,oom_memcg=/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod" + uidUnderscore + ".slice,task_memcg=/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod" + uidUnderscore + ".slice"
	oomFirst := strings.Index(oomLine, uidUnderscore)
	oomSecond := strings.LastIndex(oomLine, uidUnderscore)

	guaranteedLine := "/kubepods.slice/kubepods-pod" + uidStartingD + ".slice"
	guaranteedIdx := strings.Index(guaranteedLine, uidStartingD)

	testCases := []struct {
		name string
		text string
		want []patternfinder.PatternMatchResult[string]
	}{
		{
			name: "underscore UID with pod prefix in burstable cgroup slice",
			text: cgroupLine,
			want: []patternfinder.PatternMatchResult[string]{
				{Value: "pod-a", Start: cgroupIdx, End: cgroupIdx + len(uidUnderscore)},
			},
		},
		{
			name: "multiple occurrences of underscore UID in oom-kill log line",
			text: oomLine,
			want: []patternfinder.PatternMatchResult[string]{
				{Value: "pod-a", Start: oomFirst, End: oomFirst + len(uidUnderscore)},
				{Value: "pod-a", Start: oomSecond, End: oomSecond + len(uidUnderscore)},
			},
		},
		{
			name: "underscore UID starting with letter d after pod prefix",
			text: guaranteedLine,
			want: []patternfinder.PatternMatchResult[string]{
				{Value: "pod-d", Start: guaranteedIdx, End: guaranteedIdx + len(uidStartingD)},
			},
		},
		{
			name: "UID at beginning of text",
			text: uidUnderscore,
			want: []patternfinder.PatternMatchResult[string]{
				{Value: "pod-a", Start: 0, End: len(uidUnderscore)},
			},
		},
		{
			name: "UID followed by alphanumeric characters is not matched",
			text: "kubepods-pod" + uidUnderscore + "deadbeef.slice",
			want: nil,
		},
		{
			name: "UID followed by underscore suffix is not matched",
			text: "kubepods-pod" + uidUnderscore + "_foo.slice",
			want: nil,
		},
		{
			name: "UID preceded by non-pod alphanumeric prefix is not matched",
			text: "notpod" + uidUnderscore + ".slice",
			want: nil,
		},
		{
			name: "unregistered UID does not match",
			text: "/kubepods.slice/kubepods-burstable-pod8244a6f9_2063_489a_bbb0_e95393416ab5.slice",
			want: nil,
		},
		{
			name: "empty text",
			text: "",
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findAllUIDTokens(tc.text, finder)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("findAllUIDTokens() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProcessSerialPortLog(t *testing.T) {
	testTime := time.Date(2025, 9, 29, 6, 39, 24, 0, time.UTC)

	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name: "successful log ingestion",
			input: testlog.NewMockLog(
				testTime,
				inspectioncore.DefaultSeverityFieldSet{
					Severity: inspectioncore.SeverityError,
				},
				serialport.GCESerialPortLogFieldSet{
					Message:  "foo payload",
					NodeName: "node-name-bar",
					Port:     "serial_port_output_qux",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("foo payload").
					HasTimestamp(testTime).
					HasSeverity(inspectioncore.SeverityError).
					HasLogType(serialport.LogTypeSerialPort)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			cs, err := processSerialPortLog(ctx, tc.input)
			if err != nil {
				t.Fatalf("processSerialPortLog() returned unexpected error: %v", err)
			}
			tc.assert(t, cs)
		})
	}
}

func TestMapSerialPortLog(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())
	ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

	const (
		clusterName   = "test-cluster"
		nodeName      = "node-name-bar"
		portName      = "serial_port_output_qux"
		uidHyphen     = "8b04c24f-b9f7-49dd-ab67-f55568b07817"
		uidUnderscore = "8b04c24f_b9f7_49dd_ab67_f55568b07817"
	)

	podIdentity := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Namespace:  "default",
		Name:       "test-pod",
	}
	finder := patternfinder.NewRadixPatternFinder[*k8saudit.ResourceIdentity]()
	_ = finder.AddPattern(uidUnderscore, podIdentity)

	wantSerialPortPath := serialport.MustSerialPortTimeline(ctx, clusterName, nodeName, portName)
	wantPodPath := k8saudit.MustResourceTimeline(ctx, clusterName, podIdentity)
	wantBindingPath := k8saudit.MustK8sSubresourceTimeline(ctx, wantPodPath, "binding")
	wantPodPhasePath := mustPodPhaseTimelinePath(ctx, clusterName, nodeName, podIdentity.Namespace, podIdentity.Name, uidHyphen)

	podCreationTime := time.Date(2025, 9, 29, 6, 0, 0, 0, time.UTC)
	logTime1 := time.Date(2025, 9, 29, 6, 39, 24, 0, time.UTC)
	logTime2 := time.Date(2025, 9, 29, 6, 39, 25, 0, time.UTC)

	wantPodNode, err := makePodManifestNode(podIdentity.Namespace, podIdentity.Name, nodeName, uidHyphen)
	if err != nil {
		t.Fatalf("makePodManifestNode() failed: %v", err)
	}
	wantBindingNode, err := makeBindingManifestNode(podIdentity.Namespace, podIdentity.Name, nodeName)
	if err != nil {
		t.Fatalf("makeBindingManifestNode() failed: %v", err)
	}

	testCases := []struct {
		name                  string
		logs                  []*log.Log
		timelineCreationTimes k8saudit.TimelineCreationTimes
		assert                func(t *testing.T, ctx context.Context, changeSets []*khifilev6.TimelineChangeSet)
	}{
		{
			name: "create timeline event without resource UID",
			logs: []*log.Log{
				testlog.NewMockLog(
					logTime1,
					serialport.GCESerialPortLogFieldSet{
						Message:  "foo payload",
						NodeName: nodeName,
						Port:     portName,
					},
				),
			},
			timelineCreationTimes: k8saudit.TimelineCreationTimes{},
			assert: func(t *testing.T, ctx context.Context, changeSets []*khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, changeSets[0]).
					HasEvent(wantSerialPortPath).
					HasNoEvent(wantPodPath).
					HasNoRevision(wantPodPhasePath).
					HasNoRevision(wantBindingPath)
			},
		},
		{
			name: "maps underscore Pod UID to Pod timeline and supplements missing Pod, PodPhase, and binding once per group",
			logs: []*log.Log{
				testlog.NewMockLog(
					logTime1,
					serialport.GCESerialPortLogFieldSet{
						Message:  "[100.000000] Memory cgroup stats for /kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod" + uidUnderscore + ".slice:",
						NodeName: nodeName,
						Port:     portName,
					},
				),
				testlog.NewMockLog(
					logTime2,
					serialport.GCESerialPortLogFieldSet{
						Message:  "[100.100000] oom-kill:oom_memcg=/kubepods.slice/kubepods-burstable-pod" + uidUnderscore + ".slice,task_memcg=/kubepods.slice/kubepods-burstable-pod" + uidUnderscore + ".slice",
						NodeName: nodeName,
						Port:     portName,
					},
				),
			},
			timelineCreationTimes: k8saudit.TimelineCreationTimes{},
			assert: func(t *testing.T, ctx context.Context, changeSets []*khifilev6.TimelineChangeSet) {
				epochTime := time.Unix(0, 0).UTC()
				testchangeset.AssertTimeline(t, changeSets[0]).
					HasEvent(wantSerialPortPath).
					HasEvent(wantPodPath).
					HasEventCount(2).
					HasRevision(wantPodPath, &khifilev6.StagingRevision{
						ChangedTime:  epochTime,
						ResourceBody: wantPodNode,
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, testutil.CompareStructuredNode()).
					HasRevision(wantPodPhasePath, &khifilev6.StagingRevision{
						ChangedTime:  epochTime,
						ResourceBody: wantPodNode,
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStatePodPhaseUnknown,
					}, testutil.CompareStructuredNode()).
					HasRevision(wantBindingPath, &khifilev6.StagingRevision{
						ChangedTime:  epochTime,
						ResourceBody: wantBindingNode,
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, testutil.CompareStructuredNode())

				testchangeset.AssertTimeline(t, changeSets[1]).
					HasEvent(wantSerialPortPath).
					HasEvent(wantPodPath).
					HasEventCount(2).
					HasNoRevision(wantPodPath).
					HasNoRevision(wantPodPhasePath).
					HasNoRevision(wantBindingPath)
			},
		},
		{
			name: "skips Pod, PodPhase, and binding supplementation when audit logs already created the Pod timeline",
			logs: []*log.Log{
				testlog.NewMockLog(
					logTime1,
					serialport.GCESerialPortLogFieldSet{
						Message:  "/kubepods.slice/kubepods-burstable-pod" + uidUnderscore + ".slice",
						NodeName: nodeName,
						Port:     portName,
					},
				),
			},
			timelineCreationTimes: k8saudit.TimelineCreationTimes{
				wantPodPath: {podCreationTime},
			},
			assert: func(t *testing.T, ctx context.Context, changeSets []*khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, changeSets[0]).
					HasEvent(wantSerialPortPath).
					HasEvent(wantPodPath).
					HasEventCount(2).
					HasNoRevision(wantPodPath).
					HasNoRevision(wantPodPhasePath).
					HasNoRevision(wantBindingPath)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
			var state *serialPortMapperState
			var changeSets []*khifilev6.TimelineChangeSet
			for _, l := range tc.logs {
				var cs *khifilev6.TimelineChangeSet
				var err error
				cs, state, err = mapSerialPortLog(ctx, l, state, clusterName, finder, tc.timelineCreationTimes)
				if err != nil {
					t.Fatalf("mapSerialPortLog() returned unexpected error: %v", err)
				}
				changeSets = append(changeSets, cs)
			}

			tc.assert(t, ctx, changeSets)
		})
	}
}
