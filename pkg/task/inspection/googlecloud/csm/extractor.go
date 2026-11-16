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

package csm

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	commoncsmcp "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/csmcp"
)

const (
	// AccessLogTypeClient identifies an Envoy access log recorded by a client sidecar proxy.
	AccessLogTypeClient AccessLogType = "client"
	// AccessLogTypeServer identifies an Envoy access log recorded by a server sidecar proxy.
	AccessLogTypeServer AccessLogType = "server"
)

var (
	pathResponseFlag           = structured.CompileFieldPath("labels.response_flag")
	pathSourceNamespace        = structured.CompileFieldPath("labels.source_namespace")
	pathSourceName             = structured.CompileFieldPath("labels.source_name")
	pathDestinationNamespace   = structured.CompileFieldPath("labels.destination_namespace")
	pathDestinationName        = structured.CompileFieldPath("labels.destination_name")
	pathDestinationServiceName = structured.CompileFieldPath("labels.destination_service_name")
	pathDestinationServiceHost = structured.CompileFieldPath("labels.destination_service_host")
	pathPodName                = structured.CompileFieldPath("resource.labels.pod_name")
	pathNamespaceName          = structured.CompileFieldPath("resource.labels.namespace_name")
	pathContainerName          = structured.CompileFieldPath("resource.labels.container_name")
	pathLogName                = structured.CompileFieldPath("logName")
	pathProjectID              = structured.CompileFieldPath("resource.labels.project_id")
	pathLocation               = structured.CompileFieldPath("resource.labels.location")
	pathRevision               = structured.CompileFieldPath("resource.labels.revision")
	pathJSONPayloadMessage     = structured.CompileFieldPath("jsonPayload.message")
	pathJSONPayloadTimestamp   = structured.CompileFieldPath("jsonPayload.timestamp")
	pathTextPayload            = structured.CompileFieldPath("textPayload")
)

// AccessLogType identifies whether an Istio access log was emitted by a client or server proxy.
type AccessLogType string

// IstioAccessLogFieldSet holds structured fields for Istio access logs.
type IstioAccessLogFieldSet struct {
	// Type indicates whether the log was recorded by a client or server proxy.
	Type AccessLogType
	// ResponseFlags contains parsed Envoy response flags explaining the request outcome.
	ResponseFlags logutil.EnvoyResponseFlags

	// SourceNamespace is the Kubernetes namespace of the source workload.
	SourceNamespace string
	// SourceName is the name of the source workload or pod.
	SourceName string

	// DestinationNamespace is the Kubernetes namespace of the destination workload.
	DestinationNamespace string
	// DestinationName is the name of the destination workload or pod.
	DestinationName string
	// DestinationServiceName is the short name of the destination Kubernetes Service.
	DestinationServiceName string
	// DestinationServiceNamespace is the namespace parsed from the destination service host.
	DestinationServiceNamespace string

	// ReporterPodName is the name of the pod whose proxy reported this access log.
	ReporterPodName string
	// ReporterPodNamespace is the namespace of the reporting pod.
	ReporterPodNamespace string
	// ReporterContainerName is the container name of the reporting proxy.
	ReporterContainerName string
}

// CSMControlPlaneLogFieldSet holds structured fields extracted from CSM control plane logs.
type CSMControlPlaneLogFieldSet struct {
	// ProjectID is the GCP project ID where the control plane runs.
	ProjectID string
	// Location is the GCP region or zone of the control plane resource.
	Location string
	// Revision is the Istio control plane revision label.
	Revision string
	// Message is the log payload text extracted from jsonPayload or textPayload.
	Message string
	// Timestamp is the optional log entry timestamp parsed from jsonPayload.timestamp.
	Timestamp *time.Time
	// Pods lists Kubernetes pod identifiers referenced in the control plane log message.
	Pods []commoncsmcp.PodIdentifier
}

// ExtractIstioAccessLog extracts Istio access log fields from a NodeReader.
func ExtractIstioAccessLog(reader *structured.NodeReader) (IstioAccessLogFieldSet, error) {
	if mock, ok := structured.GetMock[IstioAccessLogFieldSet](reader); ok {
		return mock, nil
	}
	var result IstioAccessLogFieldSet
	result.ResponseFlags = logutil.ParseEnvoyResponseFlags(reader.ReadStringOrDefault(pathResponseFlag, ""))
	result.SourceNamespace = reader.ReadStringOrDefault(pathSourceNamespace, "")
	result.SourceName = reader.ReadStringOrDefault(pathSourceName, "")
	result.DestinationNamespace = reader.ReadStringOrDefault(pathDestinationNamespace, "")
	result.DestinationName = reader.ReadStringOrDefault(pathDestinationName, "")
	result.DestinationServiceName = reader.ReadStringOrDefault(pathDestinationServiceName, "")
	destinationServiceHost := reader.ReadStringOrDefault(pathDestinationServiceHost, "")
	if destinationServiceHost != "" {
		dotSplittedServiceHost := strings.Split(destinationServiceHost, ".")
		if len(dotSplittedServiceHost) >= 2 {
			result.DestinationServiceNamespace = dotSplittedServiceHost[1]
		}
	}

	result.ReporterPodName = reader.ReadStringOrDefault(pathPodName, "")
	result.ReporterPodNamespace = reader.ReadStringOrDefault(pathNamespaceName, "")
	result.ReporterContainerName = reader.ReadStringOrDefault(pathContainerName, "")

	logName, err := reader.ReadString(pathLogName)
	if err != nil {
		return result, err
	}
	switch {
	case strings.HasSuffix(logName, "server-accesslog-stackdriver"):
		result.Type = AccessLogTypeServer
	case strings.HasSuffix(logName, "client-accesslog-stackdriver"):
		result.Type = AccessLogTypeClient
	default:
		return result, fmt.Errorf("a log with unknown logName %q was given to IstioAccessLogLabelsFieldSetReader:%w", logName, khierrors.ErrInvalidInput)
	}

	return result, nil
}

// ExtractCSMControlPlaneLog extracts CSM control plane log fields from a NodeReader.
func ExtractCSMControlPlaneLog(reader *structured.NodeReader) (CSMControlPlaneLogFieldSet, error) {
	if mock, ok := structured.GetMock[CSMControlPlaneLogFieldSet](reader); ok {
		return mock, nil
	}
	message := reader.ReadStringOrDefault(pathJSONPayloadMessage, "")
	if message == "" {
		message = reader.ReadStringOrDefault(pathTextPayload, "")
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
	}
	return CSMControlPlaneLogFieldSet{
		ProjectID: reader.ReadStringOrDefault(pathProjectID, ""),
		Location:  reader.ReadStringOrDefault(pathLocation, ""),
		Revision:  reader.ReadStringOrDefault(pathRevision, ""),
		Message:   message,
		Timestamp: timestamp,
		Pods:      pods,
	}, nil
}
