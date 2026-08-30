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

var (
	pathProjectID        = structured.CompileFieldPath("resource.labels.project_id")
	pathOperationID      = structured.CompileFieldPath("operation.id")
	pathOperationFirst   = structured.CompileFieldPath("operation.first")
	pathOperationLast    = structured.CompileFieldPath("operation.last")
	pathMethodName       = structured.CompileFieldPath("protoPayload.methodName")
	pathResourceName     = structured.CompileFieldPath("protoPayload.resourceName")
	pathPrincipalEmail   = structured.CompileFieldPath("protoPayload.authenticationInfo.principalEmail")
	pathPrincipalSubject = structured.CompileFieldPath("protoPayload.authenticationInfo.principalSubject")
	pathStatusCode       = structured.CompileFieldPath("protoPayload.status.code")
	pathStatusMessage    = structured.CompileFieldPath("protoPayload.status.message")
	pathRequest          = structured.CompileFieldPath("protoPayload.request")
	pathResponse         = structured.CompileFieldPath("protoPayload.response")

	pathRequestMethod    = structured.CompileFieldPath("httpRequest.requestMethod")
	pathRequestURL       = structured.CompileFieldPath("httpRequest.requestUrl")
	pathRequestStatus    = structured.CompileFieldPath("httpRequest.status")
	pathRequestUserAgent = structured.CompileFieldPath("httpRequest.userAgent")
	pathRequestRemoteIP  = structured.CompileFieldPath("httpRequest.remoteIp")
	pathRequestServerIP  = structured.CompileFieldPath("httpRequest.serverIp")
	pathRequestReferer   = structured.CompileFieldPath("httpRequest.referer")
	pathRequestLatency   = structured.CompileFieldPath("httpRequest.latency")
	pathRequestProtocol  = structured.CompileFieldPath("httpRequest.protocol")
	pathRequestSize      = structured.CompileFieldPath("httpRequest.requestSize")
	pathResponseSize     = structured.CompileFieldPath("httpRequest.responseSize")

	pathSeverity = structured.CompileFieldPath("severity")

	pathProtoPayload = structured.CompileFieldPath("protoPayload")
	pathTextPayload  = structured.CompileFieldPath("textPayload")
	pathJSONPayload  = structured.CompileFieldPath("jsonPayload")
	pathLabels       = structured.CompileFieldPath("labels")

	pathJSONPayloadMessageFieldPaths = []structured.FieldPath{
		structured.CompileFieldPath("jsonPayload.MESSAGE"),
		structured.CompileFieldPath("jsonPayload.message"),
		structured.CompileFieldPath("jsonPayload.msg"),
		structured.CompileFieldPath("jsonPayload.log"),
	}
)

// GCPAuditLogFieldSet represents the parsed fields of a GCP Cloud Audit Log entry.
type GCPAuditLogFieldSet struct {
	ProjectID      string
	OperationID    string
	OperationFirst bool
	OperationLast  bool
	MethodName     string
	ResourceName   string
	PrincipalEmail string
	Status         int
	StatusMessage  string
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

var _ log.FieldSet = (*GCPAuditLogFieldSet)(nil)

// ExtractGCPAuditLog extracts GCP Audit Log fields from a NodeReader.
func ExtractGCPAuditLog(reader *structured.NodeReader) (GCPAuditLogFieldSet, error) {
	if mock, ok := structured.GetMock[GCPAuditLogFieldSet](reader); ok {
		return mock, nil
	}
	var result GCPAuditLogFieldSet
	result.ProjectID = reader.ReadStringOrDefaultByPath(pathProjectID, "unknown")
	result.OperationID = reader.ReadStringOrDefaultByPath(pathOperationID, "")
	result.OperationFirst = reader.ReadBoolOrDefaultByPath(pathOperationFirst, false)
	result.OperationLast = reader.ReadBoolOrDefaultByPath(pathOperationLast, false)
	result.MethodName = reader.ReadStringOrDefaultByPath(pathMethodName, "unknown")
	result.ResourceName = reader.ReadStringOrDefaultByPath(pathResourceName, "unknown")
	result.PrincipalEmail = reader.ReadStringOrDefaultByPath(pathPrincipalEmail, "")
	if result.PrincipalEmail == "" {
		result.PrincipalEmail = reader.ReadStringOrDefaultByPath(pathPrincipalSubject, "unknown")
	}
	result.Status = reader.ReadIntOrDefaultByPath(pathStatusCode, -1)
	result.StatusMessage = reader.ReadStringOrDefaultByPath(pathStatusMessage, "")
	result.Request, _ = reader.GetReaderByPath(pathRequest)
	result.Response, _ = reader.GetReaderByPath(pathResponse)
	return result, nil
}

// GCPOperationAuditLogFieldSetReader parses GCPAuditLogFieldSet from logs.
type GCPOperationAuditLogFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (g *GCPOperationAuditLogFieldSetReader) FieldSetKind() string {
	return (&GCPAuditLogFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (g *GCPOperationAuditLogFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	result, err := ExtractGCPAuditLog(reader)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

var _ log.FieldSetReader = (*GCPOperationAuditLogFieldSetReader)(nil)

// GCPAccessLogFieldSet represents HTTP access log fields from Cloud Logging.
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

// ExtractGCPAccessLog extracts GCP Access Log fields from a NodeReader.
func ExtractGCPAccessLog(reader *structured.NodeReader) (GCPAccessLogFieldSet, error) {
	if mock, ok := structured.GetMock[GCPAccessLogFieldSet](reader); ok {
		return mock, nil
	}
	var result GCPAccessLogFieldSet
	result.Method = reader.ReadStringOrDefaultByPath(pathRequestMethod, "")
	result.RequestURL = reader.ReadStringOrDefaultByPath(pathRequestURL, "")
	result.Status = reader.ReadIntOrDefaultByPath(pathRequestStatus, 0)
	result.UserAgent = reader.ReadStringOrDefaultByPath(pathRequestUserAgent, "")
	result.RemoteIP = reader.ReadStringOrDefaultByPath(pathRequestRemoteIP, "")
	result.ServerIP = reader.ReadStringOrDefaultByPath(pathRequestServerIP, "")
	result.Referer = reader.ReadStringOrDefaultByPath(pathRequestReferer, "")
	result.Latency = reader.ReadStringOrDefaultByPath(pathRequestLatency, "")
	result.Protocol = reader.ReadStringOrDefaultByPath(pathRequestProtocol, "")

	requestSizeStr := reader.ReadStringOrDefaultByPath(pathRequestSize, "")
	responseSizeStr := reader.ReadStringOrDefaultByPath(pathResponseSize, "")
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

	return result, nil
}

// GCPAccessLogFieldSetReader parses GCPAccessLogFieldSet from logs.
type GCPAccessLogFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (g *GCPAccessLogFieldSetReader) FieldSetKind() string {
	return (&GCPAccessLogFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (g *GCPAccessLogFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	result, err := ExtractGCPAccessLog(reader)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

var _ log.FieldSetReader = (*GCPAccessLogFieldSetReader)(nil)

// ExtractGCPSeverity extracts severity from a GCP Cloud Logging entry.
func ExtractGCPSeverity(reader *structured.NodeReader) (*pb.Severity, error) {
	if mock, ok := structured.GetMock[inspectioncore_contract.DefaultSeverityFieldSet](reader); ok {
		return mock.Severity, nil
	}
	if mock, ok := structured.GetMock[*pb.Severity](reader); ok {
		return mock, nil
	}
	severityStr := reader.ReadStringOrDefaultByPath(pathSeverity, "")
	return ParseGCPSeverity(severityStr), nil
}

// GCPDefaultSeverityFieldSetReader reads and parses the severity field from a GCP Cloud Logging entry.
type GCPDefaultSeverityFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (g *GCPDefaultSeverityFieldSetReader) FieldSetKind() string {
	return (&inspectioncore_contract.DefaultSeverityFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (g *GCPDefaultSeverityFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	severity, err := ExtractGCPSeverity(reader)
	if err != nil {
		return nil, err
	}
	return &inspectioncore_contract.DefaultSeverityFieldSet{
		Severity: severity,
	}, nil
}

var _ log.FieldSetReader = (*GCPDefaultSeverityFieldSetReader)(nil)

// GCPMainMessageFieldSet represents the main message parsed from a GCP log.
type GCPMainMessageFieldSet struct {
	MainMessage string
}

// Kind implements log.FieldSet.
func (d *GCPMainMessageFieldSet) Kind() string {
	return "main_message"
}

var _ log.FieldSet = (*GCPMainMessageFieldSet)(nil)

// ExtractGCPMainMessage reads main message from the content of log stored on Cloud Logging.
// It treats fields as its main message in order: protoPayload > textPayload > jsonPayload.**** > jsonPayload > labels.
func ExtractGCPMainMessage(reader *structured.NodeReader) (string, error) {
	if mock, ok := structured.GetMock[GCPMainMessageFieldSet](reader); ok {
		return mock.MainMessage, nil
	}
	if reader.HasByPath(pathProtoPayload) {
		return "", nil
	}
	if reader.HasByPath(pathTextPayload) {
		return reader.ReadStringOrDefaultByPath(pathTextPayload, ""), nil
	}
	if reader.HasByPath(pathJSONPayload) {
		for _, pathMsg := range pathJSONPayloadMessageFieldPaths {
			jsonPayloadMessage, err := reader.ReadStringByPath(pathMsg)
			if err == nil {
				return jsonPayloadMessage, nil
			}
		}
		serialized, err := reader.Serialize("jsonPayload", &structured.JSONNodeSerializer{})
		if err != nil {
			return "", err
		}
		return string(serialized), nil
	}
	if reader.HasByPath(pathLabels) {
		serialized, err := reader.Serialize("labels", &structured.JSONNodeSerializer{})
		if err != nil {
			return "", err
		}
		return string(serialized), nil
	}
	return "", nil
}

// GCPMainMessageFieldSetReader reads main message from the content of log stored on Cloud Logging.
type GCPMainMessageFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (g *GCPMainMessageFieldSetReader) FieldSetKind() string {
	return (&GCPMainMessageFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (g *GCPMainMessageFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	msg, err := ExtractGCPMainMessage(reader)
	if err != nil {
		return nil, err
	}
	return &GCPMainMessageFieldSet{MainMessage: msg}, nil
}

var _ log.FieldSetReader = (*GCPMainMessageFieldSetReader)(nil)
