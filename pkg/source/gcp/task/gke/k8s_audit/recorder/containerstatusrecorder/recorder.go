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

package containerstatusrecorder

import (
	"context"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/manifestutil"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

func Register(manager *recorder.RecorderTaskManager) error {
	manager.AddRecorder("containers", []string{}, func(ctx context.Context, resourcePath string, currentLog *types.ResourceSpecificParserInput, prevStateInGroup any, cs *history.ChangeSet, builder *history.Builder, vs *task.VariableSet) (any, error) {
		var prevPod *corev1.Pod
		if prevStateInGroup != nil {
			prevPod = prevStateInGroup.(*corev1.Pod)
		}
		return recordChangeSetForLog(ctx, resourcePath, currentLog, prevPod, cs, builder)
	}, recorder.ResourceKindLogGroupFilter("pod"), recorder.AndLogFilter(recorder.OnlySucceedLogs(), recorder.OnlyWithResourceBody()))
	return nil
}

func recordChangeSetForLog(ctx context.Context, resourcePath string, log *types.ResourceSpecificParserInput, prevPodStatus *corev1.Pod, cs *history.ChangeSet, builder *history.Builder) (*corev1.Pod, error) {
	const errorTimestampInUnix = -62135596800 // Unix time for 0001-01-01T00:00:00Z
	var pod corev1.Pod
	err := log.ResourceBodyReader.ReadReflectK8sManifest("", &pod)
	if err != nil {
		return nil, err
	}
	// Audit logs is not assured to be generated when a container becomes ready. And the ready field has no timestamp of the change
	// containers ready last transition time is used for the time of becoming ready in the containers.
	containersReadyTime := time.Unix(1<<63-1, 0)
	for _, podConditions := range pod.Status.Conditions {
		if podConditions.Type == "ContainersReady" {
			containersReadyTime = podConditions.LastTransitionTime.Time
			break
		}
	}

	deletionStatus := manifestutil.ParseDeletionStatus(ctx, log.ResourceBodyReader, log.Operation)
	isDeletionRequest := deletionStatus == manifestutil.DeletionStatusDeleted
	statuses := []corev1.ContainerStatus{}
	statuses = append(statuses, pod.Status.ContainerStatuses...)
	statuses = append(statuses, pod.Status.InitContainerStatuses...)
	for i, status := range statuses {
		statusYaml, err := yaml.Marshal(status)
		if err != nil {
			return &pod, err
		}
		isInitContainer := i >= len(pod.Status.ContainerStatuses)
		cpath := resourcepath.Container(log.Operation.Namespace, log.Operation.Name, status.Name)
		changed := builder.ClusterResource.ContainerStatuses.IsNewChange(log.Operation.Namespace, log.Operation.Name, status.Name, status)
		tb := builder.GetTimelineBuilder(cpath.Path)
		last := tb.GetLatestRevision()
		if changed {
			switch {
			case status.State.Running != nil:
				// Current container is running
				running := status.State.Running
				time := running.StartedAt.Time
				if last != nil && time.Sub(last.ChangeTime) > 0 && log.Log.Timestamp().Sub(time) > 0 && status.Ready {
					cs.RecordRevision(cpath, &history.StagingResourceRevision{
						Verb:       enum.RevisionVerbContainerNonReady,
						Body:       string(statusYaml),
						Requestor:  "",
						Partial:    false,
						ChangeTime: running.StartedAt.Time,
						State:      enum.RevisionStateContainerRunningNonReady,
					})
				}
				if status.Ready {
					readinessChangeTime := log.Log.Timestamp()
					if !isInitContainer && last != nil && containersReadyTime.Sub(last.ChangeTime) > 0 {
						readinessChangeTime = containersReadyTime
					}
					cs.RecordRevision(cpath, &history.StagingResourceRevision{
						Verb:       enum.RevisionVerbContainerReady,
						Body:       string(statusYaml),
						Requestor:  "",
						Partial:    false,
						ChangeTime: readinessChangeTime,
						State:      enum.RevisionStateContainerRunningReady,
					})
				} else {
					cs.RecordRevision(cpath, &history.StagingResourceRevision{
						Verb:       enum.RevisionVerbContainerNonReady,
						Body:       string(statusYaml),
						Requestor:  "",
						Partial:    false,
						ChangeTime: log.Log.Timestamp(),
						State:      enum.RevisionStateContainerRunningNonReady,
					})

				}
			case status.State.Terminated != nil:
				// Current container is terminated
				terminated := status.State.Terminated
				if terminated.FinishedAt.Time.Unix() == errorTimestampInUnix {
					// Pod termination status can have errornous timestamp when it can't be determined.
					// We still don't know the exact time that happening but it should be in between last change time and current log time.
					// Use timestamp log in the case.
					terminated.FinishedAt.Time = log.Log.Timestamp()
				}
				if last == nil || terminated.FinishedAt.Time.Sub(last.ChangeTime) > 0 { // If this is the first log for the container or termination time is later than the last revision change timing.
					verb := enum.RevisionVerbContainerSuccess
					state := enum.RevisionStateContainerTerminatedWithSuccess
					if terminated.ExitCode != 0 {
						verb = enum.RevisionVerbContainerError
						state = enum.RevisionStateContainerTerminatedWithError
					}
					cs.RecordRevision(cpath, &history.StagingResourceRevision{
						Verb:       verb,
						Body:       string(statusYaml),
						Requestor:  "",
						Partial:    false,
						ChangeTime: terminated.FinishedAt.Time,
						State:      state,
					})
				}

			case status.State.Waiting != nil:
				// Current container is waiting
				cs.RecordRevision(cpath, &history.StagingResourceRevision{
					Verb:       enum.RevisionVerbContainerWaiting,
					Body:       string(statusYaml),
					Requestor:  "",
					Partial:    false,
					ChangeTime: log.Log.Timestamp(),
					State:      enum.RevisionStateContainerWaiting,
				})
			}
		}

		if isDeletionRequest {
			cs.RecordRevision(cpath, &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbDelete,
				Body:       "",
				Requestor:  "",
				Partial:    false,
				ChangeTime: log.Log.Timestamp(),
				State:      enum.RevisionStateDeleted,
			})
		}
	}
	return &pod, nil
}
