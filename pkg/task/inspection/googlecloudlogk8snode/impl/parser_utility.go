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

package googlecloudlogk8snode_impl

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
)

// readGoStructFromString finds the struct part of a specific structName in the given string and returns its fields as a map.
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

// readNextQuotedString extracts the content of the first double-quoted string found in the input message.
func readNextQuotedString(msg string) string {
	splitted := strings.Split(msg, "\"")
	if len(splitted) > 2 {
		return splitted[1]
	} else {
		return ""
	}
}

// checkStartingAndTerminationLog checks if the log message matches a predefined starting or termination log for a component and adds a corresponding revision to the ChangeSet.
func checkStartingAndTerminationLog(cs *history.ChangeSet, l *log.Log, startingLog string, terminationLog string) {
	commonFieldSet := log.MustGetFieldSet(l, &log.CommonFieldSet{})
	nodeLogFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
	mainMessage, _ := logutil.ExtractKLogField(nodeLogFieldSet.Message, "")
	switch mainMessage {
	case startingLog:
		if startingLog != "" {
			cs.AddRevision(nodeLogFieldSet.ResourcePath(), &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbCreate,
				State:      enum.RevisionStateExisting,
				Requestor:  nodeLogFieldSet.Component,
				ChangeTime: commonFieldSet.Timestamp,
			})
		}
	case terminationLog:
		if terminationLog != "" {
			cs.AddRevision(nodeLogFieldSet.ResourcePath(), &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbDelete,
				State:      enum.RevisionStateDeleted,
				Requestor:  nodeLogFieldSet.Component,
				ChangeTime: commonFieldSet.Timestamp,
			})
		}
	}
}

// slashSplittedPodNameToNamespaceAndName converts a slash-separated pod name (e.g., "kube-system/kube-dns-abcd") into its namespace and name components.
func slashSplittedPodNameToNamespaceAndName(name string) (string, string, error) {
	nameSplitted := strings.Split(name, "/")
	if len(nameSplitted) == 2 {
		return nameSplitted[0], nameSplitted[1], nil
	}
	return "", "", fmt.Errorf("invalid pod name format %q: %w", name, khierrors.ErrInvalidInput)
}

// toReadablePodSandboxName formats a pod sandbox's namespace and name into a human-readable string.
func toReadablePodSandboxName(namespace string, name string) string {
	return fmt.Sprintf("【%s (Namespace: %s)】", name, namespace)
}

// toReadableContainerName formats a container's name, its parent pod's name, and namespace into a human-readable string.
func toReadableContainerName(namespace string, name string, container string) string {
	return fmt.Sprintf("【%s (Pod:%s, Namespace:%s)】", container, name, namespace)
}

// parseDefaultSummary formats given klog message into a human readable message.
func parseDefaultSummary(msg string) (string, error) {
	subinfo := ""
	klogmain, err := logutil.ExtractKLogField(msg, "")
	if err != nil {
		return "", err
	}
	errorMsg, err := logutil.ExtractKLogField(msg, "error")
	if err == nil && errorMsg != "" {
		subinfo = fmt.Sprintf("error=%s", errorMsg)
	}
	probeType, err := logutil.ExtractKLogField(msg, "probeType")
	if err == nil && probeType != "" {
		subinfo = fmt.Sprintf("probeType=%s", probeType)
	}
	eventMsg, err := logutil.ExtractKLogField(msg, "event")
	if err == nil && eventMsg != "" {
		if eventMsg[0] == '&' || eventMsg[0] == '{' {
			if strings.Contains(eventMsg, "Type:") {
				subinfo = strings.Split(strings.Split(eventMsg, "Type:")[1], " ")[0]
			}
		} else {
			subinfo = eventMsg
		}
	}
	klogstatus, err := logutil.ExtractKLogField(msg, "status")
	if err == nil && klogstatus != "" {
		subinfo = fmt.Sprintf("status=%s", klogstatus)
	}
	klogExitCode, err := logutil.ExtractKLogField(msg, "exitCode")
	if err == nil && klogExitCode != "" {
		subinfo = fmt.Sprintf("exitCode=%s", klogExitCode)
	}
	klogGracePeriod, err := logutil.ExtractKLogField(msg, "gracePeriod")
	if err == nil && klogGracePeriod != "" {
		subinfo = fmt.Sprintf("gracePeriod=%ss", klogGracePeriod)
	}
	if subinfo == "" {
		return klogmain, nil
	} else {
		return fmt.Sprintf("%s(%s)", klogmain, subinfo), nil
	}
}
