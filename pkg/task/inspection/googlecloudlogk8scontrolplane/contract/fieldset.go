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
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

type ControlplaneComponentParserType string

var (
	ComponentParserTypeScheduler         ControlplaneComponentParserType = "scheduler"
	ComponentParserTypeControllerManager ControlplaneComponentParserType = "controller-manager"
	ComponentParserTypeOther             ControlplaneComponentParserType = "other"
)

var componentNameToComponentParserTypeMap = map[string]ControlplaneComponentParserType{
	"scheduler":          ComponentParserTypeScheduler,
	"controller-manager": ComponentParserTypeControllerManager,
}

var itemsCaptureRegex = regexp.MustCompile(`\[(?P<apiVersionKind>[^,]+), namespace: (?P<namespace>[^,]*), name: (?P<name>[^,]+)`)

type K8sControlplaneComponentFieldSet struct {
	ClusterName   string
	ComponentName string
}

// Kind implements log.FieldSet.
func (k *K8sControlplaneComponentFieldSet) Kind() string {
	return "k8s_controlplane_component"
}

// ComponentParserType returns ControlplaneComponentParserType enum value which determine which control plane component parser process this log.
func (k *K8sControlplaneComponentFieldSet) ComponentParserType() ControlplaneComponentParserType {
	if parserType, found := componentNameToComponentParserTypeMap[k.ComponentName]; found {
		return parserType
	}
	return ComponentParserTypeOther
}

func (k *K8sControlplaneComponentFieldSet) ResourcePath() resourcepath.ResourcePath {
	return resourcepath.ControlplaneComponent(k.ClusterName, k.ComponentName)
}

var _ log.FieldSet = (*K8sControlplaneComponentFieldSet)(nil)

type K8sControlplaneComponentFieldSetReader struct {
}

// Read implements log.FieldSetReader.
func (k *K8sControlplaneComponentFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result K8sControlplaneComponentFieldSet
	result.ClusterName = reader.ReadStringOrDefault("resource.labels.cluster_name", "unknown")
	result.ComponentName = reader.ReadStringOrDefault("resource.labels.component_name", "")
	return &result, nil
}

// FieldSetKind implements log.FieldSetReader.
func (k *K8sControlplaneComponentFieldSetReader) FieldSetKind() string {
	return (&K8sControlplaneComponentFieldSet{}).Kind()
}

var _ log.FieldSetReader = (*K8sControlplaneComponentFieldSetReader)(nil)

type K8sControlplaneCommonMessageFieldSet struct {
	Message string
}

// Kind implements log.FieldSet.
func (k *K8sControlplaneCommonMessageFieldSet) Kind() string {
	return "k8s_controlplane_component_message"
}

var _ log.FieldSet = (*K8sControlplaneCommonMessageFieldSet)(nil)

type K8sControlplaneCommonMessageFieldSetReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (k *K8sControlplaneCommonMessageFieldSetReader) FieldSetKind() string {
	return (&K8sControlplaneCommonMessageFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (k *K8sControlplaneCommonMessageFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result K8sControlplaneCommonMessageFieldSet
	result.Message = reader.ReadStringOrDefault("jsonPayload.message", "")
	return &result, nil
}

var _ log.FieldSetReader = (*K8sControlplaneCommonMessageFieldSetReader)(nil)

type K8sSchedulerComponentFieldSet struct {
	PodName      string
	PodNamespace string
}

// Kind implements log.FieldSet.
func (k *K8sSchedulerComponentFieldSet) Kind() string {
	return "k8s_scheduler_component"
}

func (k *K8sSchedulerComponentFieldSet) HasPodField() bool {
	return k.PodName != "" && k.PodNamespace != ""
}

func (k *K8sSchedulerComponentFieldSet) ResourcePath() resourcepath.ResourcePath {
	return resourcepath.Pod(k.PodNamespace, k.PodName)
}

var _ log.FieldSet = (*K8sSchedulerComponentFieldSet)(nil)

type K8sSchedulerComponentFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (k *K8sSchedulerComponentFieldSetReader) FieldSetKind() string {
	return (&K8sSchedulerComponentFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (k *K8sSchedulerComponentFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result K8sSchedulerComponentFieldSet
	message := reader.ReadStringOrDefault("jsonPayload.message", "")
	podFQDN, err := logutil.ExtractKLogField(message, "pod")
	if err == nil {
		podNameFragments := strings.Split(podFQDN, "/")
		if len(podNameFragments) == 2 {
			result.PodNamespace = podNameFragments[0]
			result.PodName = podNameFragments[1]
		}
	}
	return &result, nil
}

var _ log.FieldSetReader = (*K8sSchedulerComponentFieldSetReader)(nil)

type K8sControllerManagerComponentFieldSet struct {
	Controller          string
	AssociatedResources []resourcepath.ResourcePath
}

func (k *K8sControllerManagerComponentFieldSet) ControlPlaneResourcePath(clusterName string) resourcepath.ResourcePath {
	if k.Controller == "" {
		return resourcepath.ControlplaneComponent(clusterName, "controller-manager")
	}
	return resourcepath.ControllerManagerControlplaneComponent(clusterName, k.Controller)
}

// Kind implements log.FieldSet.
func (k *K8sControllerManagerComponentFieldSet) Kind() string {
	return "k8s_controller_manager_component"
}

var _ log.FieldSet = (*K8sControllerManagerComponentFieldSet)(nil)

type KindToKLogFieldPairData struct {
	APIVersion   string
	KindName     string
	KLogField    string
	IsNamespaced bool
}

type K8sControllerManagerComponentFieldSetReader struct {
	WellKnownSourceLocationToControllerMap map[string]string
	WellKnownKindToKLogFieldPairs          []*KindToKLogFieldPairData
}

// FieldSetKind implements log.FieldSetReader.
func (k *K8sControllerManagerComponentFieldSetReader) FieldSetKind() string {
	return (&K8sControllerManagerComponentFieldSet{}).Kind()
}

// c Read implements log.FieldSetReader.
func (k *K8sControllerManagerComponentFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result K8sControllerManagerComponentFieldSet
	message := reader.ReadStringOrDefault("jsonPayload.message", "")
	sourceFile := reader.ReadStringOrDefault("sourceLocation.file", "")

	controller, _ := k.readController(message, sourceFile)
	result.Controller = controller
	result.AssociatedResources = k.readResourceAssociations(message)

	return &result, nil
}

func (k *K8sControllerManagerComponentFieldSetReader) readController(message string, sourceFile string) (string, error) {
	if logger, _ := logutil.ExtractKLogField(message, "logger"); logger != "" {
		return logger, nil
	}
	if controller, _ := logutil.ExtractKLogField(message, "controller"); controller != "" {
		return controller, nil
	}
	if controller, found := k.WellKnownSourceLocationToControllerMap[sourceFile]; found {
		return controller, nil
	}
	return "", nil
}

func (k *K8sControllerManagerComponentFieldSetReader) readResourceAssociations(message string) []resourcepath.ResourcePath {
	var result []resourcepath.ResourcePath
	fromKindField := k.readResourceAssociationFromKindField(message)
	result = append(result, fromKindField...)

	fromControllerSpecificField := k.readResourceAssociationFromControllerSpecificField(message)
	result = append(result, fromControllerSpecificField...)

	fromItems := k.readResourceAssociationFromItems(message)
	if fromItems.Path != "" {
		result = append(result, fromItems)
	}

	return result
}

// readResourceAssociationFromKindField reads the kind klog field to associate resource with this log.
// Example log: '"Finished syncing" kind="ReplicaSet" key="1-4-basic-ingresses/ready-repeat-app-554f6b9d95" duration="32.336593ms"'
func (k *K8sControllerManagerComponentFieldSetReader) readResourceAssociationFromKindField(message string) []resourcepath.ResourcePath {
	var result []resourcepath.ResourcePath
	kind, err := logutil.ExtractKLogField(message, "kind")
	if err == nil && kind != "" {
		kind = strings.ToLower(kind)
		key, err := logutil.ExtractKLogField(message, "key")
		if err == nil && kind != "" {
			for _, pair := range k.WellKnownKindToKLogFieldPairs {
				if pair.KindName == kind {
					if pair.IsNamespaced {
						splittedKey := strings.Split(key, "/")
						if len(splittedKey) != 2 {
							continue
						}
						result = append(result, resourcepath.NameLayerGeneralItem(pair.APIVersion, pair.KindName, splittedKey[0], splittedKey[1]))
					} else {
						result = append(result, resourcepath.NameLayerGeneralItem(pair.APIVersion, pair.KindName, "cluster-scope", key))
					}
				}
			}
		}
	}
	return result
}

// readResourceAssociationFromControllerSpecificField reads the associated resource of this log from controller specific key name.
// Example log: '"Error syncing deployment" deployment="1-4-basic-ingresses/ig-ready-repeat-app" err="Operation cannot be fulfilled on deployments.apps \"ig-ready-repeat-app\": the object has been modified; please apply your changes to the latest version and try again"'
func (k *K8sControllerManagerComponentFieldSetReader) readResourceAssociationFromControllerSpecificField(message string) []resourcepath.ResourcePath {
	var result []resourcepath.ResourcePath
	for _, pair := range k.WellKnownKindToKLogFieldPairs {
		field, err := logutil.ExtractKLogField(message, pair.KLogField)
		if err != nil || field == "" {
			continue
		}
		if pair.IsNamespaced {
			splittedField := strings.Split(field, "/")
			if len(splittedField) != 2 {
				continue
			}
			result = append(result, resourcepath.NameLayerGeneralItem(pair.APIVersion, pair.KindName, splittedField[0], splittedField[1]))
		} else {
			result = append(result, resourcepath.NameLayerGeneralItem(pair.APIVersion, pair.KindName, "cluster-scope", field))
		}
	}
	return result
}

// Example log: "Deleting item" logger="garbage-collector-controller" item="[coordination.k8s.io/v1/Lease, namespace: kube-node-lease, name: gke-p0-gke-basic-1-default-pool-4ca7ca8d-2k4v, uid: 8aba20bf-0392-40c9-ae35-240b7c099523]" propagationPolicy="Background"'
func (k *K8sControllerManagerComponentFieldSetReader) readResourceAssociationFromItems(message string) resourcepath.ResourcePath {
	var result resourcepath.ResourcePath
	item, err := logutil.ExtractKLogField(message, "item")
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
				result = resourcepath.NameLayerGeneralItem(apiVersion, kind, "cluster-scope", name)
			} else {
				result = resourcepath.NameLayerGeneralItem(apiVersion, kind, namespace, name)
			}
		}
	}
	return result
}

var _ log.FieldSetReader = (*K8sControllerManagerComponentFieldSetReader)(nil)
