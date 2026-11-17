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

package k8scontainer_impl

import (
	"fmt"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontainer"
)

// enrichCiliumAgentLog sets the severity of a cilium-agent log from its level and returns its summary with the subsystem, the error and the Pod that the log refers to.
// The summary of a JSON log otherwise only contains its message because the generic container log parser reads the message field alone.
func enrichCiliumAgentLog(cs *khifilev6.LogChangeSet, l *log.Log, containerFields k8scontainer.K8sContainerLogFieldSet, summary string) string {
	ciliumFields := k8scontainer.ExtractCiliumAgentLog(l.NodeReader, containerFields.ParsedMessage)
	if ciliumFields.Severity != nil {
		cs.SetSeverity(ciliumFields.Severity)
	}
	return formatCiliumAgentSummary(summary, ciliumFields)
}

// formatCiliumAgentSummary formats the summary of a cilium-agent log like "[endpoint] Identity of endpoint changed 【kube-dns-abcde (Namespace: kube-system)】".
func formatCiliumAgentSummary(message string, ciliumFields k8scontainer.CiliumAgentLogFieldSet) string {
	summary := message
	if ciliumFields.Error != "" {
		summary = fmt.Sprintf("%s(error=%s)", summary, ciliumFields.Error)
	}
	if ciliumFields.Subsystem != "" {
		summary = fmt.Sprintf("[%s] %s", ciliumFields.Subsystem, summary)
	}
	if ciliumFields.HasPodReference() {
		summary = fmt.Sprintf("%s 【%s (Namespace: %s)】", summary, ciliumFields.PodName, ciliumFields.PodNamespace)
	}
	return summary
}
