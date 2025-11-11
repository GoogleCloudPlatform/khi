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

package googlecloudlogk8snode_contract

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

type K8sNodeParserType string

var (
	Containerd K8sNodeParserType = "containerd"
	Kubelet    K8sNodeParserType = "kubelet"
	Other      K8sNodeParserType = "other"
)

type K8sNodeLogCommonFieldSet struct {
	Message   string
	Component string
	NodeName  string
}

func (k *K8sNodeLogCommonFieldSet) ParserType() K8sNodeParserType {
	switch k.Component {
	case "containerd":
		return Containerd
	case "kubelet":
		return Kubelet
	default:
		return Other
	}
}

func (k *K8sNodeLogCommonFieldSet) ResourcePath() resourcepath.ResourcePath {
	if k.Component == "kube-proxy" {
		return resourcepath.Pod("kube-system", fmt.Sprintf("kube-proxy-%s", k.NodeName))
	}
	return resourcepath.NodeComponent(k.NodeName, k.Component)
}

// Kind implements log.FieldSet.
func (k *K8sNodeLogCommonFieldSet) Kind() string {
	return "k8s_node_common"
}

var _ log.FieldSet = (*K8sNodeLogCommonFieldSet)(nil)

type K8sNodeLogCommonFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (k *K8sNodeLogCommonFieldSetReader) FieldSetKind() string {
	return (&K8sNodeLogCommonFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (k *K8sNodeLogCommonFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result K8sNodeLogCommonFieldSet
	result.Message = reader.ReadStringOrDefault("jsonPayload.MESSAGE", "")
	if result.Message == "" {
		result.Message = reader.ReadStringOrDefault("jsonPayload.message", "")
	}
	result.Component = reader.ReadStringOrDefault("jsonPayload.SYSLOG_IDENTIFIER", "")
	if result.Component == "" { // static pod log doesn't have SYSLOG_IDENTIFIER, use the name included in logName in the case.
		logName := reader.ReadStringOrDefault("logName", "")
		lastSlash := strings.LastIndex(logName, "/")
		if lastSlash != -1 {
			result.Component = logName[lastSlash+1:]
		}
	}
	result.Component = strings.Trim(result.Component, "()") // Some component can have () around SYSLOG_IDENTIFIER. Remove them for consistency.
	result.NodeName = reader.ReadStringOrDefault("resource.labels.node_name", "")
	return &result, nil
}

var _ log.FieldSetReader = (*K8sNodeLogCommonFieldSetReader)(nil)
