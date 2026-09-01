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

package googlecloudlogk8scontainer_contract

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
)

var (
	pathClusterName   = structured.CompileFieldPath("resource.labels.cluster_name")
	pathNamespaceName = structured.CompileFieldPath("resource.labels.namespace_name")
	pathPodName       = structured.CompileFieldPath("resource.labels.pod_name")
	pathContainerName = structured.CompileFieldPath("resource.labels.container_name")
	pathProtoPayload  = structured.CompileFieldPath("protoPayload")
	pathTextPayload   = structured.CompileFieldPath("textPayload")
	pathJsonPayload   = structured.CompileFieldPath("jsonPayload")
	pathLabels        = structured.CompileFieldPath("labels")
	pathResourceName  = structured.CompileFieldPath(`labels.compute\.googleapis\.com/resource_name`)

	pathJsonPayloadMessages = []structured.FieldPath{
		structured.CompileFieldPath("jsonPayload.MESSAGE"),
		structured.CompileFieldPath("jsonPayload.message"),
		structured.CompileFieldPath("jsonPayload.msg"),
		structured.CompileFieldPath("jsonPayload.log"),
	}
)

type K8sContainerLogFieldSet struct {
	ClusterName   string
	Namespace     string
	PodName       string
	ContainerName string
	Message       string
	ParsedMessage *logutil.ParseStructuredLogResult
}

// GroupKey returns the group key string for this container's Pod.
func (k *K8sContainerLogFieldSet) GroupKey() string {
	return fmt.Sprintf("%s/%s", k.Namespace, k.PodName)
}

// ExtractK8sContainerLog extracts container log fields from a NodeReader.
func ExtractK8sContainerLog(reader *structured.NodeReader, parser *logutil.SelectorLogParser[ContainerLogContext]) (K8sContainerLogFieldSet, error) {
	if mock, ok := structured.GetMock[K8sContainerLogFieldSet](reader); ok {
		return mock, nil
	}
	var result K8sContainerLogFieldSet
	result.ClusterName = reader.ReadStringOrDefault(pathClusterName, "unknown")
	result.Namespace = reader.ReadStringOrDefault(pathNamespaceName, "unknown")
	result.PodName = reader.ReadStringOrDefault(pathPodName, "unknown")
	result.ContainerName = reader.ReadStringOrDefault(pathContainerName, "unknown")

	rawMessage := ""
	switch {
	case reader.Has(pathProtoPayload):
		return result, nil
	case reader.Has(pathTextPayload):
		rawMessage = reader.ReadStringOrDefault(pathTextPayload, "")
	case reader.Has(pathJsonPayload):
		foundMessageField := false
		for _, p := range pathJsonPayloadMessages {
			jsonPayloadMessage, err := reader.ReadString(p)
			if err == nil {
				rawMessage = jsonPayloadMessage
				foundMessageField = true
				break
			}
		}
		if !foundMessageField {
			serialized, err := reader.Serialize(pathJsonPayload, &structured.JSONNodeSerializer{})
			if err != nil {
				return result, err
			}
			rawMessage = string(serialized)
		}
	case reader.Has(pathLabels):
		serialized, err := reader.Serialize(pathLabels, &structured.JSONNodeSerializer{})
		if err != nil {
			return result, err
		}
		rawMessage = string(serialized)
	}

	result.Message = rawMessage

	if parser == nil {
		parser = DefaultContainerLogParser
	}

	ctx := ContainerLogContext{
		Namespace:     result.Namespace,
		PodName:       result.PodName,
		ContainerName: result.ContainerName,
	}
	result.ParsedMessage = parser.TryParse(ctx, rawMessage)

	return result, nil
}

// GCPContainerLogNodeNameLabelFieldSet extracts the GCE resource name (Node name) if available.
type GCPContainerLogNodeNameLabelFieldSet struct {
	NodeName  string
	PodLabels map[string]string
}

// ExtractGCPContainerLogNodeNameLabel extracts NodeName and PodLabels from a NodeReader.
func ExtractGCPContainerLogNodeNameLabel(reader *structured.NodeReader) (GCPContainerLogNodeNameLabelFieldSet, error) {
	if mock, ok := structured.GetMock[GCPContainerLogNodeNameLabelFieldSet](reader); ok {
		return mock, nil
	}
	nodeName := reader.ReadStringOrDefault(pathResourceName, "")

	podLabels := make(map[string]string)
	labelsReader, err := reader.GetReader(pathLabels)
	if err == nil {
		for key, value := range labelsReader.Children() {
			if strings.HasPrefix(key.Key, "k8s-pod/") {
				valStr, err := value.ReadString(structured.EmptyFieldPath)
				if err == nil {
					trimmedKey := strings.TrimPrefix(key.Key, "k8s-pod/")
					podLabels[trimmedKey] = valStr
				}
			}
		}
	}

	return GCPContainerLogNodeNameLabelFieldSet{
		NodeName:  nodeName,
		PodLabels: podLabels,
	}, nil
}
