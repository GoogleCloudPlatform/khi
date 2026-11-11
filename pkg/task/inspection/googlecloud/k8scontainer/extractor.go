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

package k8scontainer

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
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

	pathCiliumSubsystem    = structured.CompileFieldPath("jsonPayload." + ciliumFieldSubsystem)
	pathCiliumModule       = structured.CompileFieldPath("jsonPayload." + ciliumFieldModule)
	pathCiliumLevel        = structured.CompileFieldPath("jsonPayload." + ciliumFieldLevel)
	pathCiliumError        = structured.CompileFieldPath("jsonPayload." + ciliumFieldError)
	pathCiliumK8sNamespace = structured.CompileFieldPath("jsonPayload." + ciliumFieldK8sNamespace)
	pathCiliumK8sPodName   = structured.CompileFieldPath("jsonPayload." + ciliumFieldK8sPodName)
	pathCiliumEndpointName = structured.CompileFieldPath("jsonPayload." + ciliumFieldEndpointName)
)

// CiliumAgentContainerName is the name of the Cilium agent container in the anetd DaemonSets of GKE Dataplane V2 and in Cilium DaemonSets.
const CiliumAgentContainerName = "cilium-agent"

// The following constants are the field names that cilium-agent writes in its structured logs.
const (
	ciliumFieldSubsystem    = "subsys"
	ciliumFieldModule       = "module"
	ciliumFieldLevel        = "level"
	ciliumFieldError        = "error"
	ciliumFieldK8sNamespace = "k8sNamespace"
	ciliumFieldK8sPodName   = "k8sPodName"
	ciliumFieldEndpointName = "ciliumEndpointName"
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

// CiliumAgentLogFieldSet holds the structured fields of a cilium-agent log.
type CiliumAgentLogFieldSet struct {
	// Subsystem is the logrus "subsys" field, or the last segment of the Hive slog "module" field (e.g., "endpoint-api" of "agent.controlplane.endpoint-api").
	Subsystem string
	// Error is the "error" field.
	Error string
	// Severity is the severity of the "level" field in a JSON payload. It is nil when the level is unknown or the log has a text payload,
	// because the logfmt parser already reads the level of text payloads.
	Severity *pb.Severity
	// PodNamespace is the namespace of the Pod that the log refers to.
	PodNamespace string
	// PodName is the name of the Pod that the log refers to.
	PodName string
}

// HasPodReference reports whether the log refers to a Pod.
func (c CiliumAgentLogFieldSet) HasPodReference() bool {
	return c.PodNamespace != "" && c.PodName != ""
}

// ExtractCiliumAgentLog extracts CiliumAgentLogFieldSet from a cilium-agent log.
// It reads the fields from jsonPayload, or from parsed when the log has a text payload in logfmt.
func ExtractCiliumAgentLog(reader *structured.NodeReader, parsed *logutil.ParseStructuredLogResult) (CiliumAgentLogFieldSet, error) {
	if mock, ok := structured.GetMock[CiliumAgentLogFieldSet](reader); ok {
		return mock, nil
	}
	if reader == nil {
		return CiliumAgentLogFieldSet{}, fmt.Errorf("failed to extract the cilium-agent log: reader is nil")
	}

	var subsystem, module, level, errorMessage, k8sNamespace, k8sPodName, endpointName string
	switch {
	case reader.Has(pathJsonPayload):
		subsystem = reader.ReadStringOrDefault(pathCiliumSubsystem, "")
		module = reader.ReadStringOrDefault(pathCiliumModule, "")
		level = reader.ReadStringOrDefault(pathCiliumLevel, "")
		errorMessage = reader.ReadStringOrDefault(pathCiliumError, "")
		k8sNamespace = reader.ReadStringOrDefault(pathCiliumK8sNamespace, "")
		k8sPodName = reader.ReadStringOrDefault(pathCiliumK8sPodName, "")
		endpointName = reader.ReadStringOrDefault(pathCiliumEndpointName, "")
	case parsed != nil:
		subsystem = readParsedStringField(parsed, ciliumFieldSubsystem)
		module = readParsedStringField(parsed, ciliumFieldModule)
		errorMessage = readParsedStringField(parsed, ciliumFieldError)
		k8sNamespace = readParsedStringField(parsed, ciliumFieldK8sNamespace)
		k8sPodName = readParsedStringField(parsed, ciliumFieldK8sPodName)
		endpointName = readParsedStringField(parsed, ciliumFieldEndpointName)
	}

	result := CiliumAgentLogFieldSet{
		Subsystem: subsystem,
		Error:     errorMessage,
		Severity:  parseCiliumLogLevel(level),
	}
	if result.Subsystem == "" && module != "" {
		result.Subsystem = module[strings.LastIndex(module, ".")+1:]
	}
	result.PodNamespace, result.PodName = resolveCiliumPodReference(k8sNamespace, k8sPodName, endpointName)
	return result, nil
}

// readParsedStringField returns the string field of a parsed log, or an empty string when the field is missing.
func readParsedStringField(parsed *logutil.ParseStructuredLogResult, field string) string {
	value, err := parsed.StringField(field)
	if err != nil {
		return ""
	}
	return value
}

// parseCiliumLogLevel returns the severity of a level written by cilium-agent, or nil for unknown levels.
func parseCiliumLogLevel(level string) *pb.Severity {
	switch strings.ToLower(level) {
	case "debug", "info":
		return inspectioncore.SeverityInfo
	case "warn", "warning":
		return inspectioncore.SeverityWarning
	case "error":
		return inspectioncore.SeverityError
	case "fatal", "panic":
		return inspectioncore.SeverityFatal
	default:
		return nil
	}
}

// resolveCiliumPodReference returns the namespace and the name of the Pod that a cilium-agent log refers to.
// Legacy logrus logs write "<namespace>/<pod>" in k8sPodName or ciliumEndpointName, while Hive slog logs write the namespace in k8sNamespace and the name in k8sPodName.
func resolveCiliumPodReference(k8sNamespace, k8sPodName, endpointName string) (string, string) {
	if namespace, name, found := splitNamespacedName(k8sPodName); found {
		return namespace, name
	}
	if k8sNamespace != "" && k8sPodName != "" && !strings.Contains(k8sPodName, "/") {
		return k8sNamespace, k8sPodName
	}
	if namespace, name, found := splitNamespacedName(endpointName); found {
		return namespace, name
	}
	return "", ""
}

// splitNamespacedName splits a "<namespace>/<name>" string into its namespace and name.
func splitNamespacedName(namespacedName string) (string, string, bool) {
	namespace, name, found := strings.Cut(namespacedName, "/")
	if !found || namespace == "" || name == "" || strings.Contains(name, "/") {
		return "", "", false
	}
	return namespace, name, true
}
