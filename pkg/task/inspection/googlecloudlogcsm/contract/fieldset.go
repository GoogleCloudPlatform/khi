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
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
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

// Kind implements log.FieldSet.
func (i *IstioAccessLogFieldSet) Kind() string {
	return "istio_accesslog"
}

var _ log.FieldSet = (*IstioAccessLogFieldSet)(nil)

type IstioAccessLogFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (i *IstioAccessLogFieldSetReader) FieldSetKind() string {
	return (&IstioAccessLogFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (i *IstioAccessLogFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result IstioAccessLogFieldSet
	result.ResponseFlags = logutil.ParseEnvoyResponseFlags(reader.ReadStringOrDefault("labels.response_flag", ""))
	result.SourceNamespace = reader.ReadStringOrDefault("labels.source_namespace", "")
	result.SourceName = reader.ReadStringOrDefault("labels.source_name", "")
	result.DestinationNamespace = reader.ReadStringOrDefault("labels.destination_namespace", "")
	result.DestinationName = reader.ReadStringOrDefault("labels.destination_name", "")
	result.DestinationServiceName = reader.ReadStringOrDefault("labels.destination_service_name", "")
	destinationServiceHost := reader.ReadStringOrDefault("labels.destination_service_host", "")
	if destinationServiceHost != "" {
		dotSplittedServiceHost := strings.Split(destinationServiceHost, ".")
		if len(dotSplittedServiceHost) >= 2 {
			result.DestinationServiceNamespace = dotSplittedServiceHost[1]
		}
	}

	result.ReporterPodName = reader.ReadStringOrDefault("resource.labels.pod_name", "")
	result.ReporterPodNamespace = reader.ReadStringOrDefault("resource.labels.namespace_name", "")
	result.ReporterContainerName = reader.ReadStringOrDefault("resource.labels.container_name", "")

	logName, err := reader.ReadString("logName")
	if err != nil {
		return nil, err
	}
	switch {
	case strings.HasSuffix(logName, "server-accesslog-stackdriver"):
		result.Type = AccessLogTypeServer
	case strings.HasSuffix(logName, "client-accesslog-stackdriver"):
		result.Type = AccessLogTypeClient
	default:
		return nil, fmt.Errorf("a log with unknown logName %q was given to IstioAccessLogLabelsFieldSetReader:%w", logName, khierrors.ErrInvalidInput)
	}

	return &result, nil

}

var _ log.FieldSetReader = (*IstioAccessLogFieldSetReader)(nil)
