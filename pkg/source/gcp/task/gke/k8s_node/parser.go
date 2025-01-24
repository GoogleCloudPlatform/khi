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

package k8s_node

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/logger"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	"github.com/GoogleCloudPlatform/khi/pkg/parser/k8s"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var GKENodeLogParseJob = parser.NewParserTaskFromParser(gcp_task.GCPPrefix+"feature/nodelog-parser", &k8sNodeParser{}, false)

const ContainerdStartingMsg = "starting containerd"
const DockerdStartingMsg = "Starting up"
const DockerdTerminatingMsg = "Daemon shutdown complete"
const ConfigureShStartingMsg = "Start to install kubernetes files"
const ConfigureShTerminatingMsg = "Done for installing kubernetes files"
const ConfigureHelperShStartingMsg = "Start to configure instance for kubernetes"
const ConfigureHelperShTerminatingMsg = "Done for the configuration for kubernetes"

type k8sNodeParser struct {
}

// Description implements parser.Parser.
func (*k8sNodeParser) Description() string {
	return `GKE worker node components logs mainly from kubelet,containerd and dockerd.

(WARNING)Log volume could be very large for long query duration or big cluster and can lead OOM. Please limit time range shorter.`
}

// GetParserName implements parser.Parser.
func (*k8sNodeParser) GetParserName() string {
	return `Kubernetes Node Logs`
}

func (*k8sNodeParser) Dependencies() []string {
	return []string{}
}

func (*k8sNodeParser) LogTask() string {
	return GKENodeLogQueryTaskID
}

func (*k8sNodeParser) Grouper() grouper.LogGrouper {
	return grouper.NewSingleStringFieldKeyLogGrouper("resource.labels.node_name")
}

func (*k8sNodeParser) GetSyslogIdentifier(l *log.LogEntity) string {
	syslogIdentiefier := l.GetStringOrDefault("jsonPayload.SYSLOG_IDENTIFIER", "Unknown")
	if strings.HasPrefix(syslogIdentiefier, "(") && strings.HasSuffix(syslogIdentiefier, ")") { // dockerd can be "(dockerd)" in SYSLOG_IDENTIFIER field.
		syslogIdentiefier = strings.TrimPrefix(strings.TrimSuffix(syslogIdentiefier, ")"), "(")
	}
	return syslogIdentiefier
}

// Parse implements parser.Parser.
func (p *k8sNodeParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder, v *task.VariableSet) error {
	if !l.HasKLogField("") {
		mainMessage, err := l.MainMessage()
		if err != nil {
			return err
		}
		cs.RecordLogSummary(mainMessage)
		return nil
	}

	nodeName := l.GetStringOrDefault("resource.labels.node_name", "")
	if nodeName == "" {
		return fmt.Errorf("parser couldn't lookup the node name")
	}
	summary, err := parseSummary(l)
	if err != nil {
		return err
	}
	cs.RecordLogSummary(summary)
	severity, err := l.KLogField(k8s.KLogSeverityFieldAlias)
	if err == nil {
		cs.RecordLogSeverity(parseSeverity(severity))
	}

	supportsLifetimeParse := false
	syslogIdentifier := p.GetSyslogIdentifier(l)
	nodeComponentPath := resourcepath.NodeComponent(nodeName, syslogIdentifier)
	if syslogIdentifier == "Unknown" {
		// Check if the log is for kube-proxy. If it was true, the log event will be generated on the Pod resource.
		logName := l.GetStringOrDefault("logName", "")
		if strings.HasSuffix(logName, "kube-proxy") {
			kubeProxyPodPath := resourcepath.Pod("kube-system", fmt.Sprintf("kube-proxy-%s", nodeName))
			cs.RecordEvent(kubeProxyPodPath)
			return nil
		}
	}

	if syslogIdentifier == "containerd" {
		msg, err := l.KLogField("msg")
		if err != nil {
			return err
		}
		err = p.handleContainerdSandboxLogs(ctx, l, msg, builder, cs, summary)
		if err != nil {
			return err
		}
		if msg == ContainerdStartingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbCreate,
					State:      enum.RevisionStateExisting,
					Requestor:  syslogIdentifier,
					ChangeTime: l.Timestamp(),
				})
		}
		supportsLifetimeParse = true
	}
	if syslogIdentifier == "dockerd" {
		msg, err := l.KLogField("msg")
		if err != nil {
			return err
		}
		if msg == DockerdStartingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbCreate,
					State:      enum.RevisionStateExisting,
					Requestor:  syslogIdentifier,
					ChangeTime: l.Timestamp(),
				})
		}
		if msg == DockerdTerminatingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbDelete,
					State:      enum.RevisionStateDeleted,
					Requestor:  syslogIdentifier,
					ChangeTime: l.Timestamp(),
				})
		}
		supportsLifetimeParse = true
	}
	if syslogIdentifier == "configure.sh" {
		msg, err := l.KLogField("")
		if err != nil {
			return err
		}
		if msg == ConfigureShStartingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbCreate,
					State:      enum.RevisionStateExisting,
					Requestor:  syslogIdentifier,
					ChangeTime: l.Timestamp(),
				})
		}
		if msg == ConfigureShTerminatingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbDelete,
					State:      enum.RevisionStateDeleted,
					Requestor:  syslogIdentifier,
					ChangeTime: l.Timestamp(),
				})
		}
		supportsLifetimeParse = true
	}
	if syslogIdentifier == "configure-helper.sh" {
		msg, err := l.KLogField("")
		if err != nil {
			return err
		}
		if msg == ConfigureHelperShStartingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbCreate,
					State:      enum.RevisionStateExisting,
					Requestor:  syslogIdentifier,
					ChangeTime: l.Timestamp(),
				})
		}
		if msg == ConfigureHelperShTerminatingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbDelete,
					State:      enum.RevisionStateDeleted,
					Requestor:  syslogIdentifier,
					ChangeTime: l.Timestamp(),
				})
		}
		supportsLifetimeParse = true
	}
	if syslogIdentifier == "kubelet" {
		klogExitCode, err := l.KLogField("exitCode")
		if err == nil && klogExitCode != "" && klogExitCode != "0" {
			if klogExitCode == "137" {
				cs.RecordLogSeverity(enum.SeverityError)
			} else {
				cs.RecordLogSeverity(enum.SeverityWarning)
			}
		}
	}

	// Add inferred revision at the beginning when parse logics written before is not supporting lifetime visualization
	if !supportsLifetimeParse {
		tb := builder.GetTimelineBuilder(nodeComponentPath.Path)
		if tb.GetLatestRevision() == nil {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbCreate,
					State:      enum.RevisionStateInferred,
					Requestor:  syslogIdentifier,
					ChangeTime: l.Timestamp(),
				})
		}
	}

	cs.RecordEvent(nodeComponentPath)

	klognode, err := l.KLogField("node")
	if err == nil && klognode != "" {
		cs.RecordEvent(resourcepath.Node(klognode))
	}

	podNameWithNamespace, err := l.KLogField("pod")
	if err == nil && podNameWithNamespace != "" {
		podNameSplitted := strings.Split(podNameWithNamespace, "/")
		podNamespace := "unknown"
		podName := "unknown"
		if len(podNameSplitted) >= 2 {
			podNamespace = podNameSplitted[0]
			podName = podNameSplitted[1]
		}
		containerName, err := l.KLogField("containerName")
		if err == nil && containerName != "" {
			cs.RecordEvent(resourcepath.Container(podNamespace, podName, containerName))
			cs.RecordLogSummary(fmt.Sprintf("%s【%s】", summary, toReadableContainerName(podNamespace, podName, containerName)))
		} else {
			containerId, err := l.KLogField("containerID")
			if err == nil && containerId != "" {
				containerId := safeParseContainerId(containerId)
				containerIdLeaseHolder, err := builder.ClusterResource.ContainerIDs.GetResourceLeaseHolderAt(containerId, l.Timestamp())
				if err != nil {
					slog.DebugContext(ctx, fmt.Sprintf("container %s was not found. It would be created before the log query start time", containerId), logger.LogKind("container-not-found"))
				} else {
					podSandboxIdLeaseHolder, err := builder.ClusterResource.PodSandboxIDs.GetResourceLeaseHolderAt(containerIdLeaseHolder.Holder.PodSandboxId, l.Timestamp())
					if err != nil {
						slog.DebugContext(ctx, fmt.Sprintf("pod %s associated to %s was not found. It would be created before the log query start time", containerIdLeaseHolder.Holder.PodSandboxId, containerId))
					} else {
						containerResourcePath := resourcepath.Container(podSandboxIdLeaseHolder.Holder.Namespace, podSandboxIdLeaseHolder.Holder.Name, containerIdLeaseHolder.Holder.ContainerName)
						cs.RecordEvent(containerResourcePath)
						cs.RecordLogSummary(fmt.Sprintf("%s【%s】", summary, toReadableContainerName(podSandboxIdLeaseHolder.Holder.Namespace, podSandboxIdLeaseHolder.Holder.Name, containerIdLeaseHolder.Holder.ContainerName)))
					}
				}
			} else {
				cs.RecordEvent(resourcepath.Pod(podNamespace, podName))
				cs.RecordLogSummary(fmt.Sprintf("%s【%s】", summary, toReadablePodSandboxName(podNamespace, podName)))
			}
		}
	}
	return nil
}

func parseSummary(l *log.LogEntity) (string, error) {
	subinfo := ""
	klogmain, err := l.KLogField("")
	if err != nil {
		return "", err
	}
	eventMsg, err := l.KLogField("event")
	if err == nil && eventMsg != "" {
		if eventMsg[0] == '&' {
			if strings.Contains(eventMsg, "Type:") {
				subinfo = strings.Split(strings.Split(eventMsg, "Type:")[1], " ")[0]
			}
		} else {
			subinfo = eventMsg
		}
	}
	klogstatus, err := l.KLogField("status")
	if err == nil && klogstatus != "" {
		subinfo = fmt.Sprintf("status=%s", klogstatus)
	}
	klogExitCode, err := l.KLogField("exitCode")
	if err == nil && klogExitCode != "" {
		subinfo = fmt.Sprintf("exitCode=%s", klogExitCode)
	}
	klogGracePeriod, err := l.KLogField("gracePeriod")
	if err == nil && klogGracePeriod != "" {
		subinfo = fmt.Sprintf("gracePeriod=%ss", klogGracePeriod)
	}
	if subinfo == "" {
		return klogmain, nil
	} else {
		return fmt.Sprintf("%s(%s)", klogmain, subinfo), nil
	}
}

func (*k8sNodeParser) handleContainerdSandboxLogs(ctx context.Context, l *log.LogEntity, mainMessage string, builder *history.Builder, cs *history.ChangeSet, summary string) error {
	// Pod sandbox related logs
	if strings.HasPrefix(mainMessage, "RunPodSandbox") {
		podSandbox, err := parseRunPodSandboxLog(mainMessage)
		if err != nil {
			return err
		}
		cs.RecordEvent(resourcepath.Pod(podSandbox.PodNamespace, podSandbox.PodName))
		if podSandbox.PodSandboxId != "" {
			builder.ClusterResource.PodSandboxIDs.TouchResourceLease(podSandbox.PodSandboxId, l.Timestamp(),
				resourcelease.NewK8sResourceLeaseHolder("pod", podSandbox.PodNamespace, podSandbox.PodName))
			cs.RecordLogSummary(rewriteIdWithReadableName(podSandbox.PodSandboxId, toReadablePodSandboxName(podSandbox.PodNamespace, podSandbox.PodName), summary))
		}
		return nil
	}

	// Container related logs
	if strings.HasPrefix(mainMessage, "CreateContainer") {
		container, err := parseCreateContainerLog(mainMessage)

		if err != nil {
			return err
		}
		podSandboxIdLease, err := builder.ClusterResource.PodSandboxIDs.GetResourceLeaseHolderAt(container.PodSandboxId, l.Timestamp())
		if err != nil {
			slog.DebugContext(ctx, fmt.Sprintf("pod sandbox %s was not found. It would be created before the log query start time", container.PodSandboxId), logger.LogKind("pod-sandbox-not-found"))
			return nil
		}
		containerResourcePath := resourcepath.Container(podSandboxIdLease.Holder.Namespace, podSandboxIdLease.Holder.Name, container.ContainerName)
		cs.RecordEvent(containerResourcePath)
		cs.RecordLogSummary(rewriteIdWithReadableName(container.PodSandboxId, toReadableContainerName(podSandboxIdLease.Holder.Namespace, podSandboxIdLease.Holder.Name, container.ContainerName), summary))
		if container.ContainerId != "" {
			builder.ClusterResource.ContainerIDs.TouchResourceLease(container.ContainerId, l.Timestamp(), resourcelease.NewContainerLeaseHolder(container.PodSandboxId, container.ContainerName))
		}
		return nil
	}

	id := readNextQuotedString(mainMessage)
	idType := builder.ClusterResource.GetNodeResourceIDTypeFromID(id, l.Timestamp())
	if idType == resourceinfo.NodeResourceIDTypePodSandbox {
		podSandbox, err := builder.ClusterResource.PodSandboxIDs.GetResourceLeaseHolderAt(id, l.Timestamp())
		if err != nil {
			slog.DebugContext(ctx, fmt.Sprintf("pod sandbox %s was not found. It would be created before the log query start time", id), logger.LogKind("pod-sandbox-not-found"))
			return nil
		}
		cs.RecordEvent(resourcepath.Pod(podSandbox.Holder.Namespace, podSandbox.Holder.Name))
		cs.RecordLogSummary(rewriteIdWithReadableName(id, toReadablePodSandboxName(podSandbox.Holder.Namespace, podSandbox.Holder.Name), summary))
		return nil
	} else if idType == resourceinfo.NodeResourceIDTypeContainer {
		containerId := readNextQuotedString(mainMessage)
		if containerId != "" {
			containerIdLease, err := builder.ClusterResource.ContainerIDs.GetResourceLeaseHolderAt(containerId, l.Timestamp())
			if err != nil {
				slog.DebugContext(ctx, fmt.Sprintf("container %s was not found. It would be created before the log query start time", containerId), logger.LogKind("container-not-found"))
				return nil
			}
			podIdLease, err := builder.ClusterResource.PodSandboxIDs.GetResourceLeaseHolderAt(containerIdLease.Holder.PodSandboxId, l.Timestamp())
			if err != nil {
				slog.DebugContext(ctx, fmt.Sprintf("pod %s associated to container %s was not found. It would be created before the log query start time", containerIdLease.Holder.PodSandboxId, containerId))
				return nil
			}
			containerResourcePath := resourcepath.Container(podIdLease.Holder.Namespace, podIdLease.Holder.Name, containerIdLease.Holder.ContainerName)
			cs.RecordEvent(containerResourcePath)
			cs.RecordLogSummary(rewriteIdWithReadableName(containerId, toReadableContainerName(podIdLease.Holder.Namespace, podIdLease.Holder.Name, containerIdLease.Holder.ContainerName), summary))
		}
		return nil
	}
	return nil
}

type runPodSandboxLog struct {
	PodName      string
	PodNamespace string
	PodSandboxId string
}

func parseRunPodSandboxLog(msg string) (*runPodSandboxLog, error) {
	// RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"
	fields := readGoStructFromString(msg, "PodSandboxMetadata")
	sandboxId := ""
	splitted := strings.Split(msg, "returns sandbox id")
	if len(splitted) >= 2 {
		sandboxId = readNextQuotedString(splitted[1])
	}
	if fields["Name"] != "" && fields["Namespace"] != "" {
		return &runPodSandboxLog{
			PodName:      fields["Name"],
			PodNamespace: fields["Namespace"],
			PodSandboxId: sandboxId,
		}, nil
	}
	return nil, fmt.Errorf("not matched. igoreing")
}

type createContainerLog struct {
	ContainerId   string
	ContainerName string
	PodSandboxId  string
}

func parseCreateContainerLog(msg string) (*createContainerLog, error) {
	fields := readGoStructFromString(msg, "ContainerMetadata")
	sandboxId := ""
	splitted := strings.Split(msg, "within sandbox")
	if len(splitted) < 2 {
		return nil, fmt.Errorf("failed to read the sandbox Id from container starting log")
	}
	sandboxId = readNextQuotedString(splitted[1])
	containerId := ""
	splitted = strings.Split(msg, "returns container id")
	if len(splitted) >= 2 {
		containerId = readNextQuotedString(splitted[1])
	}
	if fields["Name"] != "" {
		return &createContainerLog{
			PodSandboxId:  sandboxId,
			ContainerName: fields["Name"],
			ContainerId:   containerId,
		}, nil
	}
	return nil, fmt.Errorf("not matched. ignoreing")
}

// Find the struct part of specific structName in given string and returns fields.
func readGoStructFromString(message string, structName string) map[string]string {
	splitted := strings.Split(message, structName)
	if len(splitted) > 1 {
		laterPart := splitted[1]
		if len(laterPart) == 0 {
			return map[string]string{}
		}
		if laterPart[0] == '{' {
			laterPart = laterPart[1:]
		}
		structPart := strings.Split(laterPart, "}")[0]
		fields := strings.Split(structPart, ",")
		result := map[string]string{}
		for _, field := range fields {
			keyValue := strings.Split(field, ":")
			if len(keyValue) == 2 {
				result[keyValue[0]] = keyValue[1]
			}
		}
		return result
	}
	return map[string]string{}
}

func safeParseContainerId(rawContainerId string) string {
	containerId := strings.TrimPrefix(rawContainerId, "containerd://")
	idBeginFrom := strings.Index(containerId, "ID")
	if idBeginFrom == -1 {
		return containerId
	} else {
		result := ""

		if rawContainerId[idBeginFrom+len("ID:\"")] == '"' {
			// Container ID in JSON like format with double quotes
			idBeginFrom += len("ID\":\"")
		} else {
			// Container ID in JSON like format but without double quotes
			idBeginFrom += len("ID:")
		}
		for i := idBeginFrom; i < len(containerId); i++ {
			if containerId[i] == '"' || containerId[i] == ' ' || containerId[i] == '}' {
				return result
			}
			result += string(containerId[i])
		}
		return result
	}
}

func readNextQuotedString(msg string) string {
	splitted := strings.Split(msg, "\"")
	if len(splitted) > 2 {
		return splitted[1]
	} else {
		return ""
	}
}

func rewriteIdWithReadableName(containerId string, readableName string, originalMessage string) string {
	if containerId == "" {
		return originalMessage
	}
	converted := fmt.Sprintf("%s...(%s)", containerId[:min(len(containerId), 7)], readableName)
	return strings.ReplaceAll(originalMessage, containerId, converted)
}

func toReadablePodSandboxName(namespace string, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func toReadableContainerName(namespace string, name string, container string) string {
	return fmt.Sprintf("%s in %s/%s", container, namespace, name)
}

func parseSeverity(severity string) enum.Severity {
	switch severity {
	case "info":
		return enum.SeverityInfo
	case "warning":
		return enum.SeverityWarning
	case "error":
		return enum.SeverityError
	case "fatal":
		return enum.SeverityFatal
	default:
		return enum.SeverityUnknown
	}
}

var _ parser.Parser = (*k8sNodeParser)(nil)
