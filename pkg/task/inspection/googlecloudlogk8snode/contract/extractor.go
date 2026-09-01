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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
)

var (
	pathUpperMESSAGE     = structured.CompileFieldPath("jsonPayload.MESSAGE")
	pathLowerMessage     = structured.CompileFieldPath("jsonPayload.message")
	pathSyslogIdentifier = structured.CompileFieldPath("jsonPayload.SYSLOG_IDENTIFIER")
	pathLogName          = structured.CompileFieldPath("logName")
	pathNodeName         = structured.CompileFieldPath("resource.labels.node_name")
)

// K8sNodeParserType defines the parser category for node logs.
type K8sNodeParserType string

var (
	// Containerd identifies containerd logs.
	Containerd K8sNodeParserType = "containerd"
	// Kubelet identifies kubelet logs.
	Kubelet K8sNodeParserType = "kubelet"
	// Other identifies other node component logs.
	Other K8sNodeParserType = "other"
)

// K8sNodeLogCommonFieldSet contains the common parsed metadata from a Kubernetes node log.
type K8sNodeLogCommonFieldSet struct {
	Message   *logutil.ParseStructuredLogResult
	Component string
	NodeName  string
}

// ParserType returns the K8sNodeParserType corresponding to the component.
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

// ExtractK8sNodeLogCommon extracts K8sNodeLogCommonFieldSet from a NodeReader.
func ExtractK8sNodeLogCommon(reader *structured.NodeReader, parser *logutil.SelectorLogParser[NodeLogContext]) (K8sNodeLogCommonFieldSet, error) {
	if mock, ok := structured.GetMock[K8sNodeLogCommonFieldSet](reader); ok {
		return mock, nil
	}
	var result K8sNodeLogCommonFieldSet
	message := reader.ReadStringOrDefault(pathUpperMESSAGE, "")
	if message == "" {
		message = reader.ReadStringOrDefault(pathLowerMessage, "")
	}

	result.Component = reader.ReadStringOrDefault(pathSyslogIdentifier, "")
	if result.Component == "" { // static pod log doesn't have SYSLOG_IDENTIFIER, use the name included in logName in the case.
		logName := reader.ReadStringOrDefault(pathLogName, "")
		lastSlash := strings.LastIndex(logName, "/")
		if lastSlash != -1 {
			result.Component = logName[lastSlash+1:]
		}
	}
	result.Component = strings.Trim(result.Component, "()") // Some component can have () around SYSLOG_IDENTIFIER. Remove them for consistency.
	result.NodeName = reader.ReadStringOrDefault(pathNodeName, "")

	if parser == nil {
		parser = DefaultNodeLogParser
	}

	ctx := NodeLogContext{
		SyslogIdentifier: result.Component,
		NodeName:         result.NodeName,
	}
	result.Message = parser.TryParse(ctx, message)

	return result, nil
}
