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

package googlecloudlogk8scontrolplane_contract

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

var (
	pathProjectID     = structured.CompileFieldPath("resource.labels.project_id")
	pathClusterName   = structured.CompileFieldPath("resource.labels.cluster_name")
	pathComponentName = structured.CompileFieldPath("resource.labels.component_name")
	pathMessage       = structured.CompileFieldPath("jsonPayload.message")
	pathSourceFile    = structured.CompileFieldPath("sourceLocation.file")

	pathAtomicRecommendation           = structured.CompileFieldPath("jsonPayload.atomicRecommendation")
	pathAtomicHPA                      = structured.CompileFieldPath("jsonPayload.atomicRecommendation.hpa")
	pathAtomicStartTime                = structured.CompileFieldPath("jsonPayload.atomicRecommendation.startTime")
	pathAtomicMetricType               = structured.CompileFieldPath("jsonPayload.atomicRecommendation.metric.type")
	pathAtomicMetricSpecName           = structured.CompileFieldPath("jsonPayload.atomicRecommendation.metric.spec.name")
	pathAtomicMetricSpecTargetAvgValue = structured.CompileFieldPath("jsonPayload.atomicRecommendation.metric.spec.target.averageValue")
	pathAtomicMetricSpecTargetAvgUtil  = structured.CompileFieldPath("jsonPayload.atomicRecommendation.metric.spec.target.averageUtilization")
	pathAtomicMetricSpecTargetValue    = structured.CompileFieldPath("jsonPayload.atomicRecommendation.metric.spec.target.value")
	pathAtomicMetricStatusAvgValue     = structured.CompileFieldPath("jsonPayload.atomicRecommendation.metric.status.averageValue")
	pathAtomicMetricStatusAvgUtil      = structured.CompileFieldPath("jsonPayload.atomicRecommendation.metric.status.averageUtilization")
	pathAtomicMetricStatusValue        = structured.CompileFieldPath("jsonPayload.atomicRecommendation.metric.status.value")
	pathAtomicMetricNewestSampleAge    = structured.CompileFieldPath("jsonPayload.atomicRecommendation.metric.newestSampleAgeSeconds")
	pathAtomicMetricNewestSampleTime   = structured.CompileFieldPath("jsonPayload.atomicRecommendation.metric.newestSampleTime")
	pathAtomicPodCountReady            = structured.CompileFieldPath("jsonPayload.atomicRecommendation.podCount.ready")
	pathAtomicPodCountTotal            = structured.CompileFieldPath("jsonPayload.atomicRecommendation.podCount.total")
	pathAtomicPodCountUnready          = structured.CompileFieldPath("jsonPayload.atomicRecommendation.podCount.unready")
	pathAtomicPodCountIgnored          = structured.CompileFieldPath("jsonPayload.atomicRecommendation.podCount.ignored")
	pathAtomicSummaryDampening         = structured.CompileFieldPath("jsonPayload.atomicRecommendation.summary.dampening")
	pathAtomicSummaryOverride          = structured.CompileFieldPath("jsonPayload.atomicRecommendation.summary.override")
	pathAtomicSummaryReplicas          = structured.CompileFieldPath("jsonPayload.atomicRecommendation.summary.replicas")

	pathFinalRecommendation      = structured.CompileFieldPath("jsonPayload.finalRecommendation")
	pathFinalHPA                 = structured.CompileFieldPath("jsonPayload.finalRecommendation.hpa")
	pathFinalStartTime           = structured.CompileFieldPath("jsonPayload.finalRecommendation.startTime")
	pathFinalActuationTime       = structured.CompileFieldPath("jsonPayload.finalRecommendation.actuationTime")
	pathFinalActuationLatency    = structured.CompileFieldPath("jsonPayload.finalRecommendation.actuationLatencySeconds")
	pathFinalConfiguredSize      = structured.CompileFieldPath("jsonPayload.finalRecommendation.configuredSize")
	pathFinalReplicas            = structured.CompileFieldPath("jsonPayload.finalRecommendation.replicas")
	pathFinalLeadingMetricIndex  = structured.CompileFieldPath("jsonPayload.finalRecommendation.leadingMetricIndex")
	pathFinalTargetRefAPIVersion = structured.CompileFieldPath("jsonPayload.finalRecommendation.targetRef.apiVersion")
	pathFinalTargetRefKind       = structured.CompileFieldPath("jsonPayload.finalRecommendation.targetRef.kind")
	pathFinalTargetRefName       = structured.CompileFieldPath("jsonPayload.finalRecommendation.targetRef.name")
	pathFinalTopLevelLimit       = structured.CompileFieldPath("jsonPayload.finalRecommendation.topLevelLimit")
	pathFinalTopLevelOverride    = structured.CompileFieldPath("jsonPayload.finalRecommendation.topLevelOverride")
)

// ControlplaneComponentParserType defines the parser type for control plane components.
type ControlplaneComponentParserType string

var (
	// ComponentParserTypeScheduler identifies scheduler logs.
	ComponentParserTypeScheduler ControlplaneComponentParserType = "scheduler"
	// ComponentParserTypeControllerManager identifies controller-manager logs.
	ComponentParserTypeControllerManager ControlplaneComponentParserType = "controller-manager"
	// ComponentParserTypeHPAController identifies hpa-controller logs.
	ComponentParserTypeHPAController ControlplaneComponentParserType = "hpa-controller"
	// ComponentParserTypeOther identifies other control plane component logs.
	ComponentParserTypeOther ControlplaneComponentParserType = "other"
)

var componentNameToComponentParserTypeMap = map[string]ControlplaneComponentParserType{
	"scheduler":          ComponentParserTypeScheduler,
	"controller-manager": ComponentParserTypeControllerManager,
	"hpa-controller":     ComponentParserTypeHPAController,
}

var itemsCaptureRegex = regexp.MustCompile(`\[(?P<apiVersionKind>[^,]+), namespace: (?P<namespace>[^,]*), name: (?P<name>[^,]+)`)

// K8sControlplaneComponentFieldSet contains component name and cluster info for a control plane log.
type K8sControlplaneComponentFieldSet struct {
	ProjectID     string
	ClusterName   string
	ComponentName string
}

// ComponentParserType returns ControlplaneComponentParserType enum value which determines which control plane component parser processes this log.
func (k *K8sControlplaneComponentFieldSet) ComponentParserType() ControlplaneComponentParserType {
	if parserType, found := componentNameToComponentParserTypeMap[k.ComponentName]; found {
		return parserType
	}
	return ComponentParserTypeOther
}

// ExtractK8sControlplaneComponent extracts K8sControlplaneComponentFieldSet from a NodeReader.
func ExtractK8sControlplaneComponent(reader *structured.NodeReader) (K8sControlplaneComponentFieldSet, error) {
	if mock, ok := structured.GetMock[K8sControlplaneComponentFieldSet](reader); ok {
		return mock, nil
	}
	var result K8sControlplaneComponentFieldSet
	result.ProjectID = reader.ReadStringOrDefault(pathProjectID, "unknown")
	result.ClusterName = reader.ReadStringOrDefault(pathClusterName, "unknown")
	result.ComponentName = reader.ReadStringOrDefault(pathComponentName, "")
	return result, nil
}

// K8sControlplaneCommonMessageFieldSet contains the common message string for a control plane log.
type K8sControlplaneCommonMessageFieldSet struct {
	Message string
}

// ExtractK8sControlplaneCommonMessage extracts the message from a NodeReader.
func ExtractK8sControlplaneCommonMessage(reader *structured.NodeReader) (string, error) {
	if mock, ok := structured.GetMock[K8sControlplaneCommonMessageFieldSet](reader); ok {
		return mock.Message, nil
	}
	return reader.ReadStringOrDefault(pathMessage, ""), nil
}

// K8sSchedulerComponentFieldSet contains scheduler-specific fields.
type K8sSchedulerComponentFieldSet struct {
	PodName      string
	PodNamespace string
}

// HasPodField returns whether the PodName and PodNamespace are present.
func (k *K8sSchedulerComponentFieldSet) HasPodField() bool {
	return k.PodName != "" && k.PodNamespace != ""
}

// ExtractK8sSchedulerComponent extracts K8sSchedulerComponentFieldSet from a NodeReader.
func ExtractK8sSchedulerComponent(reader *structured.NodeReader, parser *logutil.SelectorLogParser[ControlplaneLogContext]) (K8sSchedulerComponentFieldSet, error) {
	if mock, ok := structured.GetMock[K8sSchedulerComponentFieldSet](reader); ok {
		return mock, nil
	}
	var result K8sSchedulerComponentFieldSet
	message := reader.ReadStringOrDefault(pathMessage, "")

	if parser == nil {
		parser = DefaultControlplaneLogParser
	}
	ctx := ControlplaneLogContext{
		ComponentName: "scheduler",
	}

	structuredRes := parser.TryParse(ctx, message)
	if structuredRes != nil {
		if podFQDN, err := structuredRes.StringField("pod"); err == nil && podFQDN != "" {
			podNameFragments := strings.Split(podFQDN, "/")
			if len(podNameFragments) == 2 {
				result.PodNamespace = podNameFragments[0]
				result.PodName = podNameFragments[1]
			}
		}
	}

	return result, nil
}

// K8sControllerManagerComponentFieldSet contains controller manager specific fields.
type K8sControllerManagerComponentFieldSet struct {
	Controller          string
	AssociatedResources []*commonlogk8saudit_contract.ResourceIdentity
}

// AssociatedResourceTimelines resolves and returns the timeline paths for all associated resources in the fieldset.
func (k *K8sControllerManagerComponentFieldSet) AssociatedResourceTimelines(ctx context.Context, clusterName string) []*khifilev6.TimelinePath {
	var result []*khifilev6.TimelinePath
	for _, resource := range k.AssociatedResources {
		result = append(result, commonlogk8saudit_contract.MustResourceTimeline(ctx, clusterName, resource))
	}
	return result
}

// KindToKLogFieldPairData maps kind name and API version to klog field name.
type KindToKLogFieldPairData struct {
	APIVersion   string
	KindName     string
	KLogField    string
	IsNamespaced bool
}

// K8sControllerManagerComponentExtractor extracts controller-manager component fields from log entries.
type K8sControllerManagerComponentExtractor struct {
	WellKnownSourceLocationToControllerMap map[string]string
	WellKnownKindToKLogFieldPairs          []*KindToKLogFieldPairData
	StructuredLogParser                    *logutil.SelectorLogParser[ControlplaneLogContext]
}

// Extract extracts K8sControllerManagerComponentFieldSet from a NodeReader.
func (k *K8sControllerManagerComponentExtractor) Extract(reader *structured.NodeReader) (K8sControllerManagerComponentFieldSet, error) {
	if mock, ok := structured.GetMock[K8sControllerManagerComponentFieldSet](reader); ok {
		return mock, nil
	}
	var result K8sControllerManagerComponentFieldSet
	message := reader.ReadStringOrDefault(pathMessage, "")
	sourceFile := reader.ReadStringOrDefault(pathSourceFile, "")

	parser := k.StructuredLogParser
	if parser == nil {
		parser = DefaultControlplaneLogParser
	}
	ctx := ControlplaneLogContext{
		ComponentName: "controller-manager",
	}

	structuredRes := parser.TryParse(ctx, message)
	controller, _ := k.ReadController(structuredRes, sourceFile)
	result.Controller = controller
	if structuredRes != nil {
		result.AssociatedResources = k.ReadResourceAssociations(structuredRes)
	}

	return result, nil
}

// ReadController reads the controller name from structured log or source file.
func (k *K8sControllerManagerComponentExtractor) ReadController(structuredRes *logutil.ParseStructuredLogResult, sourceFile string) (string, error) {
	if structuredRes != nil {
		if logger, _ := structuredRes.StringField("logger"); logger != "" {
			return logger, nil
		}
		if controller, _ := structuredRes.StringField("controller"); controller != "" {
			return controller, nil
		}
	}
	if controller, found := k.WellKnownSourceLocationToControllerMap[sourceFile]; found {
		return controller, nil
	}
	return "", nil
}

// ReadResourceAssociations extracts resource associations from structured log.
func (k *K8sControllerManagerComponentExtractor) ReadResourceAssociations(structuredRes *logutil.ParseStructuredLogResult) []*commonlogk8saudit_contract.ResourceIdentity {
	var result []*commonlogk8saudit_contract.ResourceIdentity
	fromKindField := k.ReadResourceAssociationFromKindField(structuredRes)
	result = append(result, fromKindField...)

	fromControllerSpecificField := k.ReadResourceAssociationFromControllerSpecificField(structuredRes)
	result = append(result, fromControllerSpecificField...)

	fromItems := k.ReadResourceAssociationFromItems(structuredRes)
	if fromItems != nil {
		result = append(result, fromItems)
	}

	return result
}

// ReadResourceAssociationFromKindField reads the kind klog field to associate resource with this log.
// Example log: '"Finished syncing" kind="ReplicaSet" key="1-4-basic-ingresses/ready-repeat-app-554f6b9d95" duration="32.336593ms"'.
func (k *K8sControllerManagerComponentExtractor) ReadResourceAssociationFromKindField(structuredRes *logutil.ParseStructuredLogResult) []*commonlogk8saudit_contract.ResourceIdentity {
	var result []*commonlogk8saudit_contract.ResourceIdentity
	kind, err := structuredRes.StringField("kind")
	if err == nil && kind != "" {
		kind = strings.ToLower(kind)
		key, err := structuredRes.StringField("key")
		if err == nil && kind != "" {
			for _, pair := range k.WellKnownKindToKLogFieldPairs {
				if pair.KindName == kind {
					if pair.IsNamespaced {
						splittedKey := strings.Split(key, "/")
						if len(splittedKey) != 2 {
							continue
						}
						result = append(result, &commonlogk8saudit_contract.ResourceIdentity{
							APIVersion: pair.APIVersion,
							Kind:       pair.KindName,
							Namespace:  splittedKey[0],
							Name:       splittedKey[1],
						})
					} else {
						result = append(result, &commonlogk8saudit_contract.ResourceIdentity{
							APIVersion: pair.APIVersion,
							Kind:       pair.KindName,
							Namespace:  "cluster-scope",
							Name:       key,
						})
					}
				}
			}
		}
	}
	return result
}

// ReadResourceAssociationFromControllerSpecificField reads the associated resource of this log from controller specific key name.
// Example log: '"Error syncing deployment" deployment="1-4-basic-ingresses/ig-ready-repeat-app" err="Operation cannot be fulfilled on deployments.apps \"ig-ready-repeat-app\": the object has been modified; please apply your changes to the latest version and try again"'.
func (k *K8sControllerManagerComponentExtractor) ReadResourceAssociationFromControllerSpecificField(structuredRes *logutil.ParseStructuredLogResult) []*commonlogk8saudit_contract.ResourceIdentity {
	var result []*commonlogk8saudit_contract.ResourceIdentity
	for _, pair := range k.WellKnownKindToKLogFieldPairs {
		field, err := structuredRes.StringField(pair.KLogField)
		if err != nil || field == "" {
			continue
		}
		if pair.IsNamespaced {
			splittedField := strings.Split(field, "/")
			if len(splittedField) != 2 {
				continue
			}
			result = append(result, &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: pair.APIVersion,
				Kind:       pair.KindName,
				Namespace:  splittedField[0],
				Name:       splittedField[1],
			})
		} else {
			resourceName := field

			// Some resource may have longer name with slash e.g. PV volumeName="kubernetes.io/csi/pd.csi.storage.gke.io^projects/UNSPECIFIED/zones/us-central1-a/disks/pvc-fe42fc7f-7618-4d3b-94d1-a2490cfd009d".
			lastSlashIndex := strings.LastIndex(field, "/")
			if lastSlashIndex != -1 {
				resourceName = field[lastSlashIndex+1:]
			}

			result = append(result, &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: pair.APIVersion,
				Kind:       pair.KindName,
				Namespace:  "cluster-scope",
				Name:       resourceName,
			})
		}
	}
	return result
}

// ReadResourceAssociationFromItems reads resource association from item klog field.
// Example log: "Deleting item" logger="garbage-collector-controller" item="[coordination.k8s.io/v1/Lease, namespace: kube-node-lease, name: gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v, uid: 8aba20bf-0392-40c9-ae35-240b7c099523]" propagationPolicy="Background"'.
func (k *K8sControllerManagerComponentExtractor) ReadResourceAssociationFromItems(structuredRes *logutil.ParseStructuredLogResult) *commonlogk8saudit_contract.ResourceIdentity {
	var result *commonlogk8saudit_contract.ResourceIdentity
	item, err := structuredRes.StringField("item")
	if item != "" && err == nil {
		matches := itemsCaptureRegex.FindStringSubmatch(item)
		if matches != nil {
			apiVersionKind := matches[1]
			slashIndex := strings.LastIndex(apiVersionKind, "/")
			if slashIndex == -1 {
				return result
			}
			apiVersion := apiVersionKind[:slashIndex]
			kind := apiVersionKind[slashIndex+1:]
			namespace := matches[2]
			name := matches[3]
			if apiVersion == "v1" {
				apiVersion = "core/v1"
			}
			kind = strings.ToLower(kind)
			if namespace == "" {
				result = &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: apiVersion,
					Kind:       kind,
					Namespace:  "cluster-scope",
					Name:       name,
				}
			} else {
				result = &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: apiVersion,
					Kind:       kind,
					Namespace:  namespace,
					Name:       name,
				}
			}
		}
	}
	return result
}

// HPATargetRef contains reference information for the scaling target workload.
type HPATargetRef struct {
	APIVersion string
	Kind       string
	Name       string
}

// HasTarget returns whether TargetRef has valid kind and name.
func (t HPATargetRef) HasTarget() bool {
	return t.Kind != "" && t.Name != ""
}

// HasValidResourceIdentity returns whether TargetRef has valid apiVersion, kind, and name for resource timeline mapping.
func (t HPATargetRef) HasValidResourceIdentity() bool {
	return t.APIVersion != "" && t.Kind != "" && t.Name != ""
}

// String returns the formatted resource representation "<Kind>/<Name>".
func (t HPATargetRef) String() string {
	return fmt.Sprintf("%s/%s", t.Kind, t.Name)
}

// HPAPodCount contains pod counts associated with an HPA recommendation.
type HPAPodCount struct {
	Ready   int
	Total   int
	Unready int
	Ignored int
}

// HPAAtomicRecommendation contains atomic recommendation details for a single metric.
type HPAAtomicRecommendation struct {
	HPA                    string
	HPANamespace           string
	HPAName                string
	StartTime              string
	MetricType             string
	MetricName             string
	SpecTargetAvgValue     string
	SpecTargetAvgUtil      int
	SpecTargetValue        string
	StatusAvgValue         string
	StatusAvgUtil          int
	StatusValue            string
	NewestSampleAgeSeconds float64
	NewestSampleTime       string
	PodCount               HPAPodCount
	Dampening              string
	Override               string
	Replicas               int
}

// HPAFinalRecommendation contains merged final recommendation details for an HPA.
type HPAFinalRecommendation struct {
	HPA                     string
	HPANamespace            string
	HPAName                 string
	StartTime               string
	ActuationTime           string
	ActuationLatencySeconds float64
	ConfiguredSize          int
	Replicas                int
	LeadingMetricIndex      int
	TargetRef               HPATargetRef
	TopLevelLimit           string
	TopLevelOverride        string
}

// K8sHPAControllerFieldSet contains extracted fields from an HPA controller log entry.
type K8sHPAControllerFieldSet struct {
	AtomicRecommendation *HPAAtomicRecommendation
	FinalRecommendation  *HPAFinalRecommendation
	Message              string
}

// Summary returns a human-readable summary of the HPA controller log entry.
func (k *K8sHPAControllerFieldSet) Summary() string {
	if k.FinalRecommendation != nil {
		hpa := k.FinalRecommendation.HPA
		replicas := k.FinalRecommendation.Replicas
		configuredSize := k.FinalRecommendation.ConfiguredSize
		targetRef := k.FinalRecommendation.TargetRef

		var extra []string
		if k.FinalRecommendation.TopLevelLimit != "" && k.FinalRecommendation.TopLevelLimit != "none" {
			extra = append(extra, fmt.Sprintf("limit: %s", k.FinalRecommendation.TopLevelLimit))
		}
		if k.FinalRecommendation.TopLevelOverride != "" && k.FinalRecommendation.TopLevelOverride != "none" {
			extra = append(extra, fmt.Sprintf("override: %s", k.FinalRecommendation.TopLevelOverride))
		}
		extraStr := ""
		if len(extra) > 0 {
			extraStr = fmt.Sprintf(" (%s)", strings.Join(extra, ", "))
		}

		if targetRef.HasTarget() {
			if configuredSize != replicas {
				return fmt.Sprintf("[HPA Final] %s: %s %d -> %d replicas%s", hpa, targetRef.String(), configuredSize, replicas, extraStr)
			}
			return fmt.Sprintf("[HPA Final] %s: %s = %d replicas%s", hpa, targetRef.String(), replicas, extraStr)
		}
		if configuredSize != replicas {
			return fmt.Sprintf("[HPA Final] %s: %d -> %d replicas%s", hpa, configuredSize, replicas, extraStr)
		}
		return fmt.Sprintf("[HPA Final] %s = %d replicas%s", hpa, replicas, extraStr)
	}

	if k.AtomicRecommendation != nil {
		hpa := k.AtomicRecommendation.HPA
		replicas := k.AtomicRecommendation.Replicas
		metricName := k.AtomicRecommendation.MetricName
		if metricName == "" {
			metricName = k.AtomicRecommendation.MetricType
		}

		currentVal := ""
		switch {
		case k.AtomicRecommendation.StatusAvgUtil > 0:
			currentVal = fmt.Sprintf("%d%%", k.AtomicRecommendation.StatusAvgUtil)
		case k.AtomicRecommendation.StatusAvgValue != "":
			currentVal = k.AtomicRecommendation.StatusAvgValue
		case k.AtomicRecommendation.StatusValue != "":
			currentVal = k.AtomicRecommendation.StatusValue
		}

		targetVal := ""
		switch {
		case k.AtomicRecommendation.SpecTargetAvgUtil > 0:
			targetVal = fmt.Sprintf("%d%%", k.AtomicRecommendation.SpecTargetAvgUtil)
		case k.AtomicRecommendation.SpecTargetAvgValue != "":
			targetVal = k.AtomicRecommendation.SpecTargetAvgValue
		case k.AtomicRecommendation.SpecTargetValue != "":
			targetVal = k.AtomicRecommendation.SpecTargetValue
		}

		var extra []string
		if k.AtomicRecommendation.Dampening != "" && k.AtomicRecommendation.Dampening != "none" {
			extra = append(extra, fmt.Sprintf("dampened: %s", k.AtomicRecommendation.Dampening))
		}
		if k.AtomicRecommendation.Override != "" && k.AtomicRecommendation.Override != "none" {
			extra = append(extra, fmt.Sprintf("override: %s", k.AtomicRecommendation.Override))
		}
		extraStr := ""
		if len(extra) > 0 {
			extraStr = fmt.Sprintf(" (%s)", strings.Join(extra, ", "))
		}

		if metricName != "" && currentVal != "" && targetVal != "" {
			return fmt.Sprintf("[HPA Metric] %s: %s (%s / target %s) -> %d replicas%s", hpa, metricName, currentVal, targetVal, replicas, extraStr)
		}
		if metricName != "" && currentVal != "" {
			return fmt.Sprintf("[HPA Metric] %s: %s (%s) -> %d replicas%s", hpa, metricName, currentVal, replicas, extraStr)
		}
		if metricName != "" {
			return fmt.Sprintf("[HPA Metric] %s: %s -> %d replicas%s", hpa, metricName, replicas, extraStr)
		}
		return fmt.Sprintf("[HPA Metric] %s -> %d replicas%s", hpa, replicas, extraStr)
	}

	return k.Message
}

func splitNamespaceAndName(fqdn string) (string, string) {
	parts := strings.Split(fqdn, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", fqdn
}

// ExtractK8sHPAControllerComponent extracts K8sHPAControllerFieldSet from a NodeReader.
func ExtractK8sHPAControllerComponent(reader *structured.NodeReader) (K8sHPAControllerFieldSet, error) {
	if mock, ok := structured.GetMock[K8sHPAControllerFieldSet](reader); ok {
		return mock, nil
	}
	var result K8sHPAControllerFieldSet

	if reader.Has(pathFinalRecommendation) {
		hpa := reader.ReadStringOrDefault(pathFinalHPA, "")
		ns, name := splitNamespaceAndName(hpa)
		finalRec := &HPAFinalRecommendation{
			HPA:                     hpa,
			HPANamespace:            ns,
			HPAName:                 name,
			StartTime:               reader.ReadStringOrDefault(pathFinalStartTime, ""),
			ActuationTime:           reader.ReadStringOrDefault(pathFinalActuationTime, ""),
			ActuationLatencySeconds: reader.ReadFloatOrDefault(pathFinalActuationLatency, 0),
			ConfiguredSize:          reader.ReadIntOrDefault(pathFinalConfiguredSize, 0),
			Replicas:                reader.ReadIntOrDefault(pathFinalReplicas, 0),
			LeadingMetricIndex:      reader.ReadIntOrDefault(pathFinalLeadingMetricIndex, 0),
			TargetRef: HPATargetRef{
				APIVersion: reader.ReadStringOrDefault(pathFinalTargetRefAPIVersion, ""),
				Kind:       reader.ReadStringOrDefault(pathFinalTargetRefKind, ""),
				Name:       reader.ReadStringOrDefault(pathFinalTargetRefName, ""),
			},
			TopLevelLimit:    reader.ReadStringOrDefault(pathFinalTopLevelLimit, ""),
			TopLevelOverride: reader.ReadStringOrDefault(pathFinalTopLevelOverride, ""),
		}
		result.FinalRecommendation = finalRec
		return result, nil
	}

	if reader.Has(pathAtomicRecommendation) {
		hpa := reader.ReadStringOrDefault(pathAtomicHPA, "")
		ns, name := splitNamespaceAndName(hpa)
		atomicRec := &HPAAtomicRecommendation{
			HPA:                    hpa,
			HPANamespace:           ns,
			HPAName:                name,
			StartTime:              reader.ReadStringOrDefault(pathAtomicStartTime, ""),
			MetricType:             reader.ReadStringOrDefault(pathAtomicMetricType, ""),
			MetricName:             reader.ReadStringOrDefault(pathAtomicMetricSpecName, ""),
			SpecTargetAvgValue:     reader.ReadStringOrDefault(pathAtomicMetricSpecTargetAvgValue, ""),
			SpecTargetAvgUtil:      reader.ReadIntOrDefault(pathAtomicMetricSpecTargetAvgUtil, 0),
			SpecTargetValue:        reader.ReadStringOrDefault(pathAtomicMetricSpecTargetValue, ""),
			StatusAvgValue:         reader.ReadStringOrDefault(pathAtomicMetricStatusAvgValue, ""),
			StatusAvgUtil:          reader.ReadIntOrDefault(pathAtomicMetricStatusAvgUtil, 0),
			StatusValue:            reader.ReadStringOrDefault(pathAtomicMetricStatusValue, ""),
			NewestSampleAgeSeconds: reader.ReadFloatOrDefault(pathAtomicMetricNewestSampleAge, 0),
			NewestSampleTime:       reader.ReadStringOrDefault(pathAtomicMetricNewestSampleTime, ""),
			PodCount: HPAPodCount{
				Ready:   reader.ReadIntOrDefault(pathAtomicPodCountReady, 0),
				Total:   reader.ReadIntOrDefault(pathAtomicPodCountTotal, 0),
				Unready: reader.ReadIntOrDefault(pathAtomicPodCountUnready, 0),
				Ignored: reader.ReadIntOrDefault(pathAtomicPodCountIgnored, 0),
			},
			Dampening: reader.ReadStringOrDefault(pathAtomicSummaryDampening, ""),
			Override:  reader.ReadStringOrDefault(pathAtomicSummaryOverride, ""),
			Replicas:  reader.ReadIntOrDefault(pathAtomicSummaryReplicas, 0),
		}
		result.AtomicRecommendation = atomicRec
		return result, nil
	}

	result.Message = reader.ReadStringOrDefault(pathMessage, "")
	return result, nil
}
