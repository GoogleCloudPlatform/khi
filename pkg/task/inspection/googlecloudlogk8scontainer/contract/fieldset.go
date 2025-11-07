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

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

var jsonPayloadMessageFieldNames = []string{
	"MESSAGE",
	"message",
	"msg",
	"log",
}

type K8sContainerLogFieldSet struct {
	Namespace     string
	PodName       string
	ContainerName string
	Message       string
}

// Kind implements log.FieldSet.
func (k *K8sContainerLogFieldSet) Kind() string {
	return "k8s_container"
}

func (k *K8sContainerLogFieldSet) ResourcePath() resourcepath.ResourcePath {
	return resourcepath.Container(k.Namespace, k.PodName, k.ContainerName)
}

var _ log.FieldSet = (*K8sContainerLogFieldSet)(nil)

type K8sContainerLogFieldSetReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (k *K8sContainerLogFieldSetReader) FieldSetKind() string {
	return (&K8sContainerLogFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (k *K8sContainerLogFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result K8sContainerLogFieldSet
	result.Namespace = reader.ReadStringOrDefault("resource.labels.namespace_name", "unknown")
	result.PodName = reader.ReadStringOrDefault("resource.labels.pod_name", "unknown")
	result.ContainerName = reader.ReadStringOrDefault("resource.labels.container_name", "unknown")
	switch {
	case reader.Has("protoPayload"):
		return &result, nil
	case reader.Has("textPayload"):
		result.Message = reader.ReadStringOrDefault("textPayload", "")
	case reader.Has("jsonPayload"):
		foundMessageField := false
		for _, fieldName := range jsonPayloadMessageFieldNames {
			jsonPayloadMessage, err := reader.ReadString(fmt.Sprintf("jsonPayload.%s", fieldName))
			if err == nil {
				result.Message = jsonPayloadMessage
				foundMessageField = true
				break
			}
		}
		if !foundMessageField {
			serialized, err := reader.Serialize("jsonPayload", &structured.JSONNodeSerializer{})
			if err != nil {
				return nil, err
			}
			result.Message = string(serialized)
		}
	case reader.Has("labels"):
		serialized, err := reader.Serialize("labels", &structured.JSONNodeSerializer{})
		if err != nil {
			return nil, err
		}
		result.Message = string(serialized)
	}
	return &result, nil
}

var _ log.FieldSetReader = (*K8sContainerLogFieldSetReader)(nil)
