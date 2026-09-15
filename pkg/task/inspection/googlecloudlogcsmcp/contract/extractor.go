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

package googlecloudlogcsmcp_contract

import (
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	commoncsmcp "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/csmcp"
)

var (
	pathClusterName          = structured.CompileFieldPath("resource.labels.cluster_name")
	pathTextPayload          = structured.CompileFieldPath("textPayload")
	pathJSONPayloadTimestamp = structured.CompileFieldPath("jsonPayload.timestamp")
	pathJSONPayloadMessage   = structured.CompileFieldPath("jsonPayload.message")
	pathJSONPayloadMsg       = structured.CompileFieldPath("jsonPayload.msg")
	pathJSONPayloadMESSAGE   = structured.CompileFieldPath("jsonPayload.MESSAGE")
	pathJSONPayloadLog       = structured.CompileFieldPath("jsonPayload.log")
	pathTimestamp            = structured.CompileFieldPath("timestamp")
)

// FieldSet holds structured log data extracted from the log entry.
type FieldSet struct {
	Timestamp   *time.Time
	Message     string
	Pods        []commoncsmcp.PodIdentifier
	ClusterName string
}

// Extract extracts FieldSet from a NodeReader.
func Extract(reader *structured.NodeReader) (FieldSet, error) {
	if mock, ok := structured.GetMock[FieldSet](reader); ok {
		return mock, nil
	}

	clusterName := reader.ReadStringOrDefault(pathClusterName, "")

	var message string
	switch {
	case reader.Has(pathTextPayload):
		message = reader.ReadStringOrDefault(pathTextPayload, "")
	case reader.Has(pathJSONPayloadMessage):
		message = reader.ReadStringOrDefault(pathJSONPayloadMessage, "")
	case reader.Has(pathJSONPayloadMsg):
		message = reader.ReadStringOrDefault(pathJSONPayloadMsg, "")
	case reader.Has(pathJSONPayloadMESSAGE):
		message = reader.ReadStringOrDefault(pathJSONPayloadMESSAGE, "")
	case reader.Has(pathJSONPayloadLog):
		message = reader.ReadStringOrDefault(pathJSONPayloadLog, "")
	}

	var pods []commoncsmcp.PodIdentifier
	for _, word := range strings.Fields(message) {
		if p := commoncsmcp.ParsePodIdentifier(word); p != nil {
			pods = append(pods, *p)
		}
	}

	var timestamp *time.Time
	if ts, err := reader.ReadTimestamp(pathJSONPayloadTimestamp); err == nil {
		timestamp = &ts
	} else if ts, err := reader.ReadTimestamp(pathTimestamp); err == nil {
		timestamp = &ts
	}

	return FieldSet{
		Timestamp:   timestamp,
		Message:     message,
		Pods:        pods,
		ClusterName: clusterName,
	}, nil
}
