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

package googlecloudlogcsm_contract

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
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
)

type AccessLogType string

const (
	AccessLogTypeClient AccessLogType = "client"
	AccessLogTypeServer AccessLogType = "server"
)

// IstioAccessLogFieldSet holds structured fields for Istio access logs.
type IstioAccessLogFieldSet struct {
	Type          AccessLogType
	ResponseFlags logutil.EnvoyResponseFlags

	SourceNamespace string
	SourceName      string

	DestinationNamespace        string
	DestinationName             string
	DestinationServiceName      string
	DestinationServiceNamespace string

	ReporterPodName       string
	ReporterPodNamespace  string
	ReporterContainerName string
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
