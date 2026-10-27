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
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/serialport"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// logFilterTask removes logs with empty message.
// This message is mostly just contained escape sequences and stripped by ANSIEscapeSequenceStripper.
var logFilterTask = inspectiontaskbase.DefineLogFilterTask(
	serialport.LogFilterTaskID,
	serialport.LogQueryTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogFilterFunc {
		return func(ctx context.Context, l *log.Log) bool {
			msg, err := serialport.ExtractGCESerialPortMessage(l.NodeReader)
			if err != nil {
				return false
			}
			return msg != ""
		}
	},
)

// cgroupPodUIDPatternFinderTask builds a pattern finder for underscore-separated Pod UIDs from the resource UID inventory.
// Systemd cgroup slice names in serial port logs encode Kubernetes Pod UIDs with underscores instead of hyphens.
var cgroupPodUIDPatternFinderTask = inspectiontaskbase.DefineInspectionTask(
	serialport.CgroupPodUIDPatternFinderTaskID,
	func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[patternfinder.PatternFinder[*k8saudit.ResourceIdentity]] {
		uidMapInput := coretask.Use(b, k8saudit.ResourceUIDInventoryTaskID.Ref())
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (patternfinder.PatternFinder[*k8saudit.ResourceIdentity], error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return nil, nil
			}
			uidMap := uidMapInput.Get(ctx)
			finder := patternfinder.NewRadixPatternFinder[*k8saudit.ResourceIdentity]()
			for uid, resource := range uidMap {
				if resource == nil || resource.APIVersion != "core/v1" || resource.Kind != "pod" {
					continue
				}
				underscoreUID := strings.ReplaceAll(uid, "-", "_")
				if err := finder.AddPattern(underscoreUID, resource); err != nil {
					return nil, err
				}
			}
			return finder, nil
		}
	},
)

// isUIDTokenChar reports whether b is an ASCII alphanumeric character or underscore.
// Treating underscore as part of the token prevents re-scanning intermediate segments
// of an underscore-separated UID token.
func isUIDTokenChar(b byte) bool {
	return ('0' <= b && b <= '9') || ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z') || b == '_'
}

// findAllUIDTokens finds all occurrences of registered underscore-separated UID patterns
// at token boundaries within searchText, including UIDs prefixed with "pod" in systemd cgroup slice names.
//
// When a match is found after a "pod" prefix, the returned PatternMatchResult Start and End positions
// span only the UID portion (excluding the leading "pod").
func findAllUIDTokens[T any](searchText string, finder patternfinder.PatternFinder[T]) []patternfinder.PatternMatchResult[T] {
	if finder == nil || len(searchText) == 0 {
		return nil
	}
	var results []patternfinder.PatternMatchResult[T]
	for i := 0; i < len(searchText); {
		if !isUIDTokenChar(searchText[i]) || (i > 0 && isUIDTokenChar(searchText[i-1])) {
			i++
			continue
		}

		matchStart := i
		match, ok := finder.Match(searchText[matchStart:])
		if (!ok || match.End == 0) && i+3 < len(searchText) && searchText[i:i+3] == "pod" {
			matchStart = i + 3
			match, ok = finder.Match(searchText[matchStart:])
		}
		matchEnd := matchStart + match.End
		if !ok || match.End == 0 || (matchEnd < len(searchText) && isUIDTokenChar(searchText[matchEnd])) {
			i++
			continue
		}
		results = append(results, patternfinder.PatternMatchResult[T]{
			Value: match.Value,
			Start: matchStart,
			End:   matchEnd,
		})
		i = matchEnd
	}
	return results
}

// processSerialPortLog sets the log type, timestamp, severity and summary of a serial port log.
func processSerialPortLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(serialport.LogTypeSerialPort)
	cs.SetTimestamp(l.Timestamp)

	if severity, err := gcpcommon.ExtractGCPSeverity(l.NodeReader); err == nil {
		cs.SetSeverity(severity)
	}

	if serialFS, err := serialport.ExtractGCESerialPortLog(l.NodeReader); err == nil {
		cs.SetSummary(serialFS.Message)
	}

	return cs, nil
}

// logIngesterTask ingests the metadata of the filtered serial port logs.
var logIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	serialport.LogIngesterTaskID,
	serialport.LogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		return processSerialPortLog
	},
)

// logGrouperTask is the grouper task for GCE serial port logs.
// It groups logs by the node name and port name.
var logGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	serialport.LogGrouperTaskID,
	serialport.LogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			serialFS, err := serialport.ExtractGCESerialPortLog(l.NodeReader)
			if err != nil {
				return ""
			}
			return fmt.Sprintf("%s#%s", serialFS.NodeName, serialFS.Port)
		}
	},
)

// serialPortMapperState tracks Pod UIDs whose PodPhase and binding timelines have already been
// supplemented within the current serial port log group.
type serialPortMapperState struct {
	supplementedPodUIDs map[string]struct{}
}

// serialPortLogToTimelineMapper maps logs to hierarchical node serial port timelines and to
// the timelines of Pods referenced by UID in the serial port message.
type serialPortLogToTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*serialPortMapperState]
	clusterIdentity           coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	cgroupPodUIDPatternFinder coretask.Input[patternfinder.PatternFinder[*k8saudit.ResourceIdentity]]
	timelineCreationTimes     coretask.Input[k8saudit.TimelineCreationTimes]
}

// ProcessLogByGroup processes each log inside the group and stages the event on the timeline.
func (s *serialPortLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, state *serialPortMapperState) (*khifilev6.TimelineChangeSet, *serialPortMapperState, error) {
	return mapSerialPortLog(
		ctx,
		l,
		state,
		s.clusterIdentity.Get(ctx).ClusterName,
		s.cgroupPodUIDPatternFinder.Get(ctx),
		s.timelineCreationTimes.Get(ctx),
	)
}

var _ inspectiontaskbase.TimelineMapper[*serialPortMapperState] = (*serialPortLogToTimelineMapper)(nil)

// mapSerialPortLog adds an event for a serial port log to the serial port timeline of its node in the given cluster,
// adds events to the timelines of any Pods referenced by UID in the message, and supplements missing PodPhase
// and binding revisions when a Pod UID is observed on the node.
func mapSerialPortLog(
	ctx context.Context,
	l *log.Log,
	state *serialPortMapperState,
	clusterName string,
	cgroupPodUIDPatternFinder patternfinder.PatternFinder[*k8saudit.ResourceIdentity],
	timelineCreationTimes k8saudit.TimelineCreationTimes,
) (*khifilev6.TimelineChangeSet, *serialPortMapperState, error) {
	serialportFieldSet, err := serialport.ExtractGCESerialPortLog(l.NodeReader)
	if err != nil {
		return nil, state, err
	}

	if state == nil {
		state = &serialPortMapperState{
			supplementedPodUIDs: map[string]struct{}{},
		}
	}

	targetPath := serialport.MustSerialPortTimeline(
		ctx,
		clusterName,
		serialportFieldSet.NodeName,
		serialportFieldSet.Port,
	)

	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(targetPath)

	for _, result := range findAllUIDTokens(serialportFieldSet.Message, cgroupPodUIDPatternFinder) {
		res := result.Value
		cs.AddEvent(k8saudit.MustResourceTimeline(ctx, clusterName, res))

		if serialportFieldSet.NodeName != "" && serialportFieldSet.NodeName != "unknown" {
			matchedUID := serialportFieldSet.Message[result.Start:result.End]
			canonicalUID := strings.ReplaceAll(matchedUID, "_", "-")
			if _, alreadySupplemented := state.supplementedPodUIDs[canonicalUID]; alreadySupplemented {
				continue
			}
			state.supplementedPodUIDs[canonicalUID] = struct{}{}

			if err := supplementPodNodeBinding(ctx, cs, clusterName, serialportFieldSet.NodeName, res, canonicalUID, timelineCreationTimes); err != nil {
				return nil, state, err
			}
		}
	}

	return cs, state, nil
}

// supplementPodNodeBinding adds fallback Pod, PodPhase, and binding revisions for a Pod observed on nodeName
// when audit logs did not already create those timelines.
func supplementPodNodeBinding(
	ctx context.Context,
	cs *khifilev6.TimelineChangeSet,
	clusterName string,
	nodeName string,
	pod *k8saudit.ResourceIdentity,
	canonicalUID string,
	timelineCreationTimes k8saudit.TimelineCreationTimes,
) error {
	podPath := k8saudit.MustResourceTimeline(ctx, clusterName, pod)
	bindingPath := k8saudit.MustK8sSubresourceTimeline(ctx, podPath, "binding")
	podPhasePath := mustPodPhaseTimelinePath(ctx, clusterName, nodeName, pod.Namespace, pod.Name, canonicalUID)

	// Check if audit log has already recorded creation times for the Pod, its binding, or its phase timeline.
	hasPodCreationTime := len(timelineCreationTimes[podPath]) > 0
	hasBindingCreationTime := len(timelineCreationTimes[bindingPath]) > 0
	hasPodPhaseCreationTime := len(timelineCreationTimes[podPhasePath]) > 0
	if hasPodCreationTime || hasBindingCreationTime || hasPodPhaseCreationTime {
		return nil
	}

	podNode, err := makePodManifestNode(pod.Namespace, pod.Name, nodeName, canonicalUID)
	if err != nil {
		return fmt.Errorf("failed to generate pod manifest: %w", err)
	}

	// A Pod UID is immutably bound to a single node and serial port logs carry no mutable Pod metadata,
	// so this fallback revision is only added once at the beginning of the timeline.
	changedTime := time.Unix(0, 0).UTC()

	cs.AddRevision(podPhasePath, &khifilev6.StagingRevision{
		ChangedTime:  changedTime,
		ResourceBody: podNode,
		Principal:    "N/A",
		VerbType:     k8saudit.VerbUnknown,
		StateType:    k8saudit.RevisionStatePodPhaseUnknown,
	})
	cs.AddRevision(podPath, &khifilev6.StagingRevision{
		ChangedTime:  changedTime,
		ResourceBody: podNode,
		Principal:    "N/A",
		VerbType:     k8saudit.VerbUnknown,
		StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
	})

	bindingNode, err := makeBindingManifestNode(pod.Namespace, pod.Name, nodeName)
	if err != nil {
		return fmt.Errorf("failed to generate binding manifest: %w", err)
	}
	cs.AddRevision(bindingPath, &khifilev6.StagingRevision{
		ChangedTime:  changedTime,
		ResourceBody: bindingNode,
		Principal:    "N/A",
		VerbType:     k8saudit.VerbUnknown,
		StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
	})

	return nil
}

// makePodManifestNode constructs a structured YAML node for a synthesized Pod manifest.
func makePodManifestNode(namespace, podName, nodeName, uid string) (structured.Node, error) {
	podManifest := map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":      podName,
			"namespace": namespace,
			"uid":       uid,
		},
		"spec": map[string]any{
			"nodeName": nodeName,
		},
	}
	return structured.FromGoValue(podManifest, &structured.AlphabeticalGoMapKeyOrderProvider{})
}

// makeBindingManifestNode constructs a structured YAML node for a synthesized Binding manifest.
func makeBindingManifestNode(namespace, podName, nodeName string) (structured.Node, error) {
	bindingManifest := map[string]any{
		"apiVersion": "v1",
		"kind":       "Binding",
		"metadata": map[string]any{
			"name":      podName,
			"namespace": namespace,
		},
		"target": map[string]any{
			"kind": "Node",
			"name": nodeName,
		},
	}
	return structured.FromGoValue(bindingManifest, &structured.AlphabeticalGoMapKeyOrderProvider{})
}

// mustPodPhaseTimelinePath resolves the TimelinePath for a PodPhase timeline under the Node timeline.
func mustPodPhaseTimelinePath(ctx context.Context, clusterName, nodeName, namespace, podName, uid string) *khifilev6.TimelinePath {
	cluster := k8saudit.MustK8sClusterTimeline(ctx, clusterName)
	api := k8saudit.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := k8saudit.MustK8sKindTimeline(ctx, api, "node")
	nodePath := k8saudit.MustK8sClusterScopeResourceTimeline(ctx, kind, nodeName)

	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)
	return builder.TimelineAccumulator.GetPath(nodePath, khifilev6.PathSegment{
		Name: fmt.Sprintf("%s/%s[%s]", namespace, podName, uid),
		Type: k8saudit.TimelineTypePodPhase,
	})
}

// logToTimelineMapperTask is the timeline mapper task.
var logToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	serialport.LogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: serialport.LogIngesterTaskID.Ref(),
		GroupedLogs: serialport.LogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[*serialPortMapperState] {
		return &serialPortLogToTimelineMapper{
			clusterIdentity:           coretask.Use(b, k8scommon.ClusterIdentityTaskID.Ref()),
			cgroupPodUIDPatternFinder: coretask.Use(b, serialport.CgroupPodUIDPatternFinderTaskID.Ref()),
			timelineCreationTimes:     coretask.Use(b, k8saudit.TimelineCreationTimeInventoryTaskID.Ref()),
		}
	},
	inspectioncore.FeatureTaskLabel(
		"GCE Node Serial Port Logs",
		`Gather serial port logs from GCE instances to troubleshoot VM bootstrapping and OS initialization issues.`,
		10000,
		false,
	),
)
