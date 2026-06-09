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

package googlecloudcommon_contract

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

type GCPAuditLogFieldSet struct {
	ProjectID      string
	OperationID    string
	OperationFirst bool
	OperationLast  bool
	MethodName     string
	ResourceName   string
	PrincipalEmail string
	Status         int
	Request        *structured.NodeReader
	Response       *structured.NodeReader
}

// Kind implements log.FieldSet.
func (g *GCPAuditLogFieldSet) Kind() string {
	return "gcp_operation"
}

// Starting returns true when the operation is long running operation and the log entry is for the starting timing.
func (g *GCPAuditLogFieldSet) Starting() bool {
	return g.OperationFirst && !g.OperationLast
}

// Ending returns true when the operation is long running operation and the log entry is for the ending timing.
func (g *GCPAuditLogFieldSet) Ending() bool {
	return g.OperationLast && !g.OperationFirst
}

// ImmediateOperation returns true when the log represents an operation completes immediately.
func (g *GCPAuditLogFieldSet) ImmediateOperation() bool {
	return (g.OperationFirst && g.OperationLast) || (!g.OperationFirst && !g.OperationLast)
}

// GuessRevisionVerb returns the guessed revision verb from the method name.
func (g *GCPAuditLogFieldSet) GuessRevisionVerb() *pb.Verb {
	methodNameSplitted := strings.Split(g.MethodName, ".")
	shortMethodName := "unknown"
	if len(methodNameSplitted) > 0 {
		shortMethodName = methodNameSplitted[len(methodNameSplitted)-1]
	}
	shortMethodName = strings.ToLower(shortMethodName)

	switch {
	case strings.HasPrefix(shortMethodName, "create"), strings.HasPrefix(shortMethodName, "insert"):
		return commonlogk8saudit_contract.VerbCreate
	case strings.HasPrefix(shortMethodName, "delete"):
		return commonlogk8saudit_contract.VerbDelete
	case strings.HasPrefix(shortMethodName, "update"), strings.HasPrefix(shortMethodName, "patch"):
		return commonlogk8saudit_contract.VerbUpdate
	default:
		return commonlogk8saudit_contract.VerbUpdate
	}
}

// RequestString returns the request body as a YAML string.
func (g *GCPAuditLogFieldSet) RequestString() (string, error) {
	if g.Request != nil {
		requestBodyRaw, err := g.Request.Serialize("", &structured.YAMLNodeSerializer{})
		if err != nil {
			return "", err
		}
		return string(requestBodyRaw), nil
	}
	return "", fmt.Errorf("protoPayload.request field is absent: %w", khierrors.ErrNotFound)
}

// ResponseString returns the response body as a YAML string.
func (g *GCPAuditLogFieldSet) ResponseString() (string, error) {
	if g.Response != nil {
		responseBodyRaw, err := g.Response.Serialize("", &structured.YAMLNodeSerializer{})
		if err != nil {
			return "", err
		}
		return string(responseBodyRaw), nil
	}
	return "", fmt.Errorf("protoPayload.response field is absent: %w", khierrors.ErrNotFound)
}

var _ (log.FieldSet) = (*GCPAuditLogFieldSet)(nil)

type GCPOperationAuditLogFieldSetReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (g *GCPOperationAuditLogFieldSetReader) FieldSetKind() string {
	return (&GCPAuditLogFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (g *GCPOperationAuditLogFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result GCPAuditLogFieldSet
	result.ProjectID = reader.ReadStringOrDefault("resource.labels.project_id", "unknown")
	result.OperationID = reader.ReadStringOrDefault("operation.id", "")
	result.OperationFirst = reader.ReadBoolOrDefault("operation.first", false)
	result.OperationLast = reader.ReadBoolOrDefault("operation.last", false)
	result.MethodName = reader.ReadStringOrDefault("protoPayload.methodName", "unknown")
	result.ResourceName = reader.ReadStringOrDefault("protoPayload.resourceName", "unknown")
	result.PrincipalEmail = reader.ReadStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")
	result.Status = reader.ReadIntOrDefault("protoPayload.status.code", -1)
	result.Request, _ = reader.GetReader("protoPayload.request")
	result.Response, _ = reader.GetReader("protoPayload.response")
	return &result, nil
}

var _ (log.FieldSetReader) = (*GCPOperationAuditLogFieldSetReader)(nil)

type GCPAccessLogFieldSet struct {
	Method       string
	RequestURL   string
	RequestSize  int64
	Status       int
	ResponseSize int64
	UserAgent    string
	RemoteIP     string
	ServerIP     string
	Referer      string
	Latency      string
	Protocol     string
}

// Kind implements log.FieldSet.
func (g *GCPAccessLogFieldSet) Kind() string {
	return "gcp_accesslog"
}

var _ log.FieldSet = (*GCPAccessLogFieldSet)(nil)

type GCPAccessLogFieldSetReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (g *GCPAccessLogFieldSetReader) FieldSetKind() string {
	return (&GCPAccessLogFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (g *GCPAccessLogFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result GCPAccessLogFieldSet
	result.Method = reader.ReadStringOrDefault("httpRequest.requestMethod", "")
	result.RequestURL = reader.ReadStringOrDefault("httpRequest.requestUrl", "")
	result.Status = reader.ReadIntOrDefault("httpRequest.status", 0)
	result.UserAgent = reader.ReadStringOrDefault("httpRequest.userAgent", "")
	result.RemoteIP = reader.ReadStringOrDefault("httpRequest.remoteIp", "")
	result.ServerIP = reader.ReadStringOrDefault("httpRequest.serverIp", "")
	result.Referer = reader.ReadStringOrDefault("httpRequest.referer", "")
	result.Latency = reader.ReadStringOrDefault("httpRequest.latency", "")
	result.Protocol = reader.ReadStringOrDefault("httpRequest.protocol", "")

	requestSizeStr := reader.ReadStringOrDefault("httpRequest.requestSize", "")
	responseSizeStr := reader.ReadStringOrDefault("httpRequest.responseSize", "")
	if requestSizeStr != "" {
		if size, err := strconv.ParseInt(requestSizeStr, 10, 64); err == nil {
			result.RequestSize = size
		}
	}
	if responseSizeStr != "" {
		if size, err := strconv.ParseInt(responseSizeStr, 10, 64); err == nil {
			result.ResponseSize = size
		}
	}

	return &result, nil
}

var _ log.FieldSetReader = (*GCPAccessLogFieldSetReader)(nil)

// GCPDefaultSeverityFieldSetReader reads and parses the severity field from a GCP Cloud Logging entry.
type GCPDefaultSeverityFieldSetReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (g *GCPDefaultSeverityFieldSetReader) FieldSetKind() string {
	return (&inspectioncore_contract.DefaultSeverityFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (g *GCPDefaultSeverityFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	severityStr := reader.ReadStringOrDefault("severity", "")
	parsedSeverity := ParseGCPSeverity(severityStr)
	return &inspectioncore_contract.DefaultSeverityFieldSet{
		Severity: parsedSeverity,
	}, nil
}

var _ log.FieldSetReader = (*GCPDefaultSeverityFieldSetReader)(nil)

type GCPMainMessageFieldSet struct {
	MainMessage string
}

// Kind implements FieldSet.
func (d *GCPMainMessageFieldSet) Kind() string {
	return "main_message"
}

var _ log.FieldSet = (*GCPMainMessageFieldSet)(nil)

// jsonPayloadMessageFieldNames is the field names to look for the main message in jsonPayload, in order of priority.
var jsonPayloadMessageFieldNames = []string{
	"MESSAGE",
	"message",
	"msg",
	"log",
}

// GCPMainMessageFieldSetReader read its main message from the content of log stored on Cloud Logging.
// It treats fields as its main message in the order: `textPayload` > `jsonPayload.****` (**** would be `message`, `msg`...etc) > jsonPayload > labels
type GCPMainMessageFieldSetReader struct{}

func (g *GCPMainMessageFieldSetReader) FieldSetKind() string {
	return (&GCPMainMessageFieldSet{}).Kind()
}

func (g *GCPMainMessageFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	result := &GCPMainMessageFieldSet{}
	switch {
	case reader.Has("protoPayload"):
		return result, nil
	case reader.Has("textPayload"):
		result.MainMessage = reader.ReadStringOrDefault("textPayload", "")
	case reader.Has("jsonPayload"):
		foundMessageField := false
		for _, fieldName := range jsonPayloadMessageFieldNames {
			jsonPayloadMessage, err := reader.ReadString(fmt.Sprintf("jsonPayload.%s", fieldName))
			if err == nil {
				result.MainMessage = jsonPayloadMessage
				foundMessageField = true
				break
			}
		}
		if !foundMessageField {
			serialized, err := reader.Serialize("jsonPayload", &structured.JSONNodeSerializer{})
			if err != nil {
				return nil, err
			}
			result.MainMessage = string(serialized)
		}
	case reader.Has("labels"):
		serialized, err := reader.Serialize("labels", &structured.JSONNodeSerializer{})
		if err != nil {
			return nil, err
		}
		result.MainMessage = string(serialized)
	}

	return result, nil
}

var _ log.FieldSetReader = (*GCPMainMessageFieldSetReader)(nil)
