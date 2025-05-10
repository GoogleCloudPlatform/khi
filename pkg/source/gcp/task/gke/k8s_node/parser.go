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
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/noderesource"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/inspectiontype"
	k8s_node_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_node/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var GKENodeLogParseJob = parser.NewParserTaskFromParser(k8s_node_taskid.GKENodeLogParserTaskID, &k8sNodeParser{}, false, inspectiontype.GCPK8sClusterInspectionTypes)

const ContainerdStartingMsg = "starting containerd"
const DockerdStartingMsg = "Starting up"
const DockerdTerminatingMsg = "Daemon shutdown complete"
const ConfigureShStartingMsg = "Start to install kubernetes files"
const ConfigureShTerminatingMsg = "Done for installing kubernetes files"
const ConfigureHelperShStartingMsg = "Start to configure instance for kubernetes"
const ConfigureHelperShTerminatingMsg = "Done for the configuration for kubernetes"

type k8sNodeParser struct {
}

// TargetLogType implements parser.Parser.
func (p *k8sNodeParser) TargetLogType() enum.LogType {
	return enum.LogTypeNode
}

// Description implements parser.Parser.
func (*k8sNodeParser) Description() string {
	return `Gather node components(e.g docker/container) logs. Log volume can be huge when the cluster has many nodes.`
}

// GetParserName implements parser.Parser.
func (*k8sNodeParser) GetParserName() string {
	return `Kubernetes Node Logs`
}

func (*k8sNodeParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (*k8sNodeParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return k8s_node_taskid.GKENodeLogQueryTaskID.Ref()
}

func (*k8sNodeParser) Grouper() grouper.LogGrouper {
	return grouper.NewSingleStringFieldKeyLogGrouper("resource.labels.node_name")
}

func (*k8sNodeParser) GetSyslogIdentifier(l *log.Log) string {
	syslogIdentiefier := l.ReadStringOrDefault("jsonPayload.SYSLOG_IDENTIFIER", "Unknown")
	if strings.HasPrefix(syslogIdentiefier, "(") && strings.HasSuffix(syslogIdentiefier, ")") { // dockerd can be "(dockerd)" in SYSLOG_IDENTIFIER field.
		syslogIdentiefier = strings.TrimPrefix(strings.TrimSuffix(syslogIdentiefier, ")"), "(")
	}
	return syslogIdentiefier
}

// Parse implements parser.Parser.
func (p *k8sNodeParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	commonMessageFieldSet := log.MustGetFieldSet(l, &log.CommonFieldSet{})
	mainMessageFieldSet := log.MustGetFieldSet(l, &log.MainMessageFieldSet{})
	if !mainMessageFieldSet.HasKLogField("") {
		cs.RecordLogSummary(mainMessageFieldSet.MainMessage)
		return nil
	}

	nodeName := l.ReadStringOrDefault("resource.labels.node_name", "")
	if nodeName == "" {
		return fmt.Errorf("parser couldn't lookup the node name")
	}
	summary, err := parseDefaultSummary(l)
	if err != nil {
		return err
	}
	cs.RecordLogSummary(summary)

	severity := mainMessageFieldSet.KLogSeverity()

	cs.RecordLogSeverity(severity)

	supportsLifetimeParse := false
	syslogIdentifier := p.GetSyslogIdentifier(l)
	nodeComponentPath := resourcepath.NodeComponent(nodeName, syslogIdentifier)
	if syslogIdentifier == "Unknown" {
		// Check if the log is for kube-proxy. If it was true, the log event will be generated on the Pod resource.
		logName := l.ReadStringOrDefault("logName", "")
		if strings.HasSuffix(logName, "kube-proxy") {
			kubeProxyPodPath := resourcepath.Pod("kube-system", fmt.Sprintf("kube-proxy-%s", nodeName))
			cs.RecordEvent(kubeProxyPodPath)
			return nil
		}
	}

	if syslogIdentifier == "containerd" {
		msg, err := mainMessageFieldSet.KLogField("msg")
		if err != nil {
			return err
		}
		err = p.handleContainerdSandboxLogs(ctx, l, nodeName, msg, builder, cs, summary)
		if err != nil {
			return err
		}
		if msg == ContainerdStartingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbCreate,
					State:      enum.RevisionStateExisting,
					Requestor:  syslogIdentifier,
					ChangeTime: commonMessageFieldSet.Timestamp,
				})
		}
		supportsLifetimeParse = true
	}
	if syslogIdentifier == "dockerd" {
		msg, err := mainMessageFieldSet.KLogField("msg")
		if err != nil {
			return err
		}
		if msg == DockerdStartingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbCreate,
					State:      enum.RevisionStateExisting,
					Requestor:  syslogIdentifier,
					ChangeTime: commonMessageFieldSet.Timestamp,
				})
		}
		if msg == DockerdTerminatingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbDelete,
					State:      enum.RevisionStateDeleted,
					Requestor:  syslogIdentifier,
					ChangeTime: commonMessageFieldSet.Timestamp,
				})
		}
		supportsLifetimeParse = true
	}
	if syslogIdentifier == "configure.sh" {
		msg, err := mainMessageFieldSet.KLogField("")
		if err != nil {
			return err
		}
		if msg == ConfigureShStartingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbCreate,
					State:      enum.RevisionStateExisting,
					Requestor:  syslogIdentifier,
					ChangeTime: commonMessageFieldSet.Timestamp,
				})
		}
		if msg == ConfigureShTerminatingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbDelete,
					State:      enum.RevisionStateDeleted,
					Requestor:  syslogIdentifier,
					ChangeTime: commonMessageFieldSet.Timestamp,
				})
		}
		supportsLifetimeParse = true
	}
	if syslogIdentifier == "configure-helper.sh" {
		msg, err := mainMessageFieldSet.KLogField("")
		if err != nil {
			return err
		}
		if msg == ConfigureHelperShStartingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbCreate,
					State:      enum.RevisionStateExisting,
					Requestor:  syslogIdentifier,
					ChangeTime: commonMessageFieldSet.Timestamp,
				})
		}
		if msg == ConfigureHelperShTerminatingMsg {
			cs.RecordRevision(nodeComponentPath,
				&history.StagingResourceRevision{
					Verb:       enum.RevisionVerbDelete,
					State:      enum.RevisionStateDeleted,
					Requestor:  syslogIdentifier,
					ChangeTime: commonMessageFieldSet.Timestamp,
				})
		}
		supportsLifetimeParse = true
	}
	if syslogIdentifier == "kubelet" {
		klogExitCode, err := mainMessageFieldSet.KLogField("exitCode")
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
					ChangeTime: commonMessageFieldSet.Timestamp,
				})
		}
	}

	cs.RecordEvent(nodeComponentPath)

	klognode, err := mainMessageFieldSet.KLogField("node")
	if err == nil && klognode != "" {
		cs.RecordEvent(resourcepath.Node(klognode))
	}

	resourceBindings := builder.ClusterResource.NodeResourceLogBinder.GetBoundResourcesForLogBody(nodeName, mainMessageFieldSet.MainMessage)
	for _, rb := range resourceBindings {
		cs.RecordEvent(rb.GetResourcePath())
		summary = rb.RewriteLogSummary(summary)
	}
	if len(resourceBindings) > 0 {
		cs.RecordLogSummary(summary)
	} else {
		// When this log can't be associated with resource by container id or pod sandbox id, try to get it from klog fields.
		podNameWithNamespace, err := mainMessageFieldSet.KLogField("pod")
		if err == nil && podNameWithNamespace != "" {
			podNameSplitted := strings.Split(podNameWithNamespace, "/")
			podNamespace := "unknown"
			podName := "unknown"
			if len(podNameSplitted) >= 2 {
				podNamespace = podNameSplitted[0]
				podName = podNameSplitted[1]
			}
			containerName, err := mainMessageFieldSet.KLogField("containerName")
			if err == nil && containerName != "" {
				cs.RecordEvent(resourcepath.Container(podNamespace, podName, containerName))
				cs.RecordLogSummary(fmt.Sprintf("%s【%s】", summary, toReadableContainerName(podNamespace, podName, containerName)))
			} else {
				cs.RecordEvent(resourcepath.Pod(podNamespace, podName))
				cs.RecordLogSummary(fmt.Sprintf("%s【%s】", summary, toReadablePodSandboxName(podNamespace, podName)))
			}
		}
	}
	return nil
}

func parseDefaultSummary(l *log.Log) (string, error) {
	mainMessageFieldSet := log.MustGetFieldSet(l, &log.MainMessageFieldSet{})
	subinfo := ""
	klogmain, err := mainMessageFieldSet.KLogField("")
	if err != nil {
		return "", err
	}
	errorMsg, err := mainMessageFieldSet.KLogField("error")
	if err == nil && errorMsg != "" {
		subinfo = fmt.Sprintf("error=%s", errorMsg)
	}
	probeType, err := mainMessageFieldSet.KLogField("probeType")
	if err == nil && probeType != "" {
		subinfo = fmt.Sprintf("probeType=%s", probeType)
	}
	eventMsg, err := mainMessageFieldSet.KLogField("event")
	if err == nil && eventMsg != "" {
		if eventMsg[0] == '&' || eventMsg[0] == '{' {
			if strings.Contains(eventMsg, "Type:") {
				subinfo = strings.Split(strings.Split(eventMsg, "Type:")[1], " ")[0]
			}
		} else {
			subinfo = eventMsg
		}
	}
	klogstatus, err := mainMessageFieldSet.KLogField("status")
	if err == nil && klogstatus != "" {
		subinfo = fmt.Sprintf("status=%s", klogstatus)
	}
	klogExitCode, err := mainMessageFieldSet.KLogField("exitCode")
	if err == nil && klogExitCode != "" {
		subinfo = fmt.Sprintf("exitCode=%s", klogExitCode)
	}
	klogGracePeriod, err := mainMessageFieldSet.KLogField("gracePeriod")
	if err == nil && klogGracePeriod != "" {
		subinfo = fmt.Sprintf("gracePeriod=%ss", klogGracePeriod)
	}
	if subinfo == "" {
		return klogmain, nil
	} else {
		return fmt.Sprintf("%s(%s)", klogmain, subinfo), nil
	}
}

func (*k8sNodeParser) handleContainerdSandboxLogs(ctx context.Context, l *log.Log, nodeName string, mainMessage string, builder *history.Builder, cs *history.ChangeSet, summary string) error {
	// Pod sandbox related logs
	if strings.HasPrefix(mainMessage, "RunPodSandbox") {
		podSandbox, err := parseRunPodSandboxLog(mainMessage)
		if err != nil {
			return err
		}
		if podSandbox.PodSandboxID != "" {
			builder.ClusterResource.NodeResourceLogBinder.AddResourceBinding(nodeName, noderesource.NewPodResourceBinding(
				podSandbox.PodSandboxID,
				podSandbox.PodNamespace,
				podSandbox.PodName,
			))
		}
		return nil
	}

	// Container related logs
	if strings.HasPrefix(mainMessage, "CreateContainer") {
		container, err := parseCreateContainerLog(mainMessage)
		if err != nil {
			return err
		}
		if container.ContainerID == "" {
			slog.DebugContext(ctx, fmt.Sprintf("container ID is empty string for container %s. This is ignored because it would be kube-proxy container.", container.ContainerName), logger.LogKind("empty-container-id"))
			return nil
		}
		if container.ContainerName == "" {
			slog.WarnContext(ctx, fmt.Sprintf("container name is empty for pod sandbox id %s", container.PodSandboxID), logger.LogKind("empty-container-name"))
			return nil
		}
		bindingsForPodSandboxID := builder.ClusterResource.NodeResourceLogBinder.GetBoundResourcesForLogBody(nodeName, container.PodSandboxID)
		if len(bindingsForPodSandboxID) == 0 {
			slog.DebugContext(ctx, fmt.Sprintf("pod sandbox %s was not found. It would be created before the log query start time", container.PodSandboxID), logger.LogKind("pod-sandbox-not-found"))
			return nil
		}
		if len(bindingsForPodSandboxID) > 1 {
			return fmt.Errorf("multiple pod sandboxes were found associated to pod sandbox id %s. This is unexpected behavior. Please check the log", container.PodSandboxID)
		}
		podResourceBinding, casted := bindingsForPodSandboxID[0].(*noderesource.PodResourceBinding)
		if !casted {
			return fmt.Errorf("pod sandbox ID %s is not associated with a PodResourceBinding reference. %v was given", container.PodSandboxID, bindingsForPodSandboxID[0])
		}
		containerResourceBinding := podResourceBinding.NewContainerResourceBinding(container.ContainerID, container.ContainerName)
		builder.ClusterResource.NodeResourceLogBinder.AddResourceBinding(nodeName, containerResourceBinding)
		return nil
	}
	return nil
}

type runPodSandboxLog struct {
	PodName      string
	PodNamespace string
	PodSandboxID string
}

func parseRunPodSandboxLog(msg string) (*runPodSandboxLog, error) {
	// RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"
	fields := readGoStructFromString(msg, "PodSandboxMetadata")
	sandboxID := ""
	splitted := strings.Split(msg, "returns sandbox id")
	if len(splitted) >= 2 {
		sandboxID = readNextQuotedString(splitted[1])
	}
	if fields["Name"] != "" && fields["Namespace"] != "" {
		return &runPodSandboxLog{
			PodName:      fields["Name"],
			PodNamespace: fields["Namespace"],
			PodSandboxID: sandboxID,
		}, nil
	}
	return nil, fmt.Errorf("not matched. igoreing")
}

type createContainerLog struct {
	ContainerID   string
	ContainerName string
	PodSandboxID  string
}

func parseCreateContainerLog(msg string) (*createContainerLog, error) {
	fields := readGoStructFromString(msg, "ContainerMetadata")
	sandboxID := ""
	splitted := strings.Split(msg, "within sandbox")
	if len(splitted) < 2 {
		return nil, fmt.Errorf("failed to read the sandbox Id from container starting log")
	}
	sandboxID = readNextQuotedString(splitted[1])
	containerID := ""
	splitted = strings.Split(msg, "returns container id")
	if len(splitted) >= 2 {
		containerID = readNextQuotedString(splitted[1])
	}
	if fields["Name"] != "" {
		return &createContainerLog{
			PodSandboxID:  sandboxID,
			ContainerName: fields["Name"],
			ContainerID:   containerID,
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

func readNextQuotedString(msg string) string {
	splitted := strings.Split(msg, "\"")
	if len(splitted) > 2 {
		return splitted[1]
	} else {
		return ""
	}
}

func toReadablePodSandboxName(namespace string, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func toReadableContainerName(namespace string, name string, container string) string {
	return fmt.Sprintf("%s in %s/%s", container, namespace, name)
}

var _ parser.Parser = (*k8sNodeParser)(nil)
