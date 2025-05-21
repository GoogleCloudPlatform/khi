// Copyright 2024 Google LLC
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

package componentparser

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/logger"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
)

type KindToKlogFieldPairData struct {
	APIVersion   string
	KindName     string
	KLogField    string
	IsNamespaced bool
}

func kindToKLogFieldPair(apiVersion string, kind string, klogField string, isNamespaced bool) *KindToKlogFieldPairData {
	return &KindToKlogFieldPairData{
		APIVersion:   apiVersion,
		KindName:     kind,
		KLogField:    klogField,
		IsNamespaced: isNamespaced,
	}
}

var kindToKLogFieldPairs = []*KindToKlogFieldPairData{
	kindToKLogFieldPair("apps/v1", "deployment", "deployment", true),
	kindToKLogFieldPair("apps/v1", "replicaset", "replicaSet", true),
	kindToKLogFieldPair("apps/v1", "statefulset", "statefulSet", true),
	kindToKLogFieldPair("apps/v1", "daemonset", "daemonSet", true),
	kindToKLogFieldPair("batch/v1", "cronjob", "cronjob", true),
	kindToKLogFieldPair("batch/v1", "job", "job", true),
	kindToKLogFieldPair("policy/v1", "poddisruptionbudget", "podDisruptionBudget", true),
	kindToKLogFieldPair("certificates.k8s.io/v1", "certificatesigningrequest", "csr", true),
	kindToKLogFieldPair("core/v1", "persistentvolumeclaim", "PVC", true),
	kindToKLogFieldPair("core/v1", "service", "service", true),
	kindToKLogFieldPair("core/v1", "node", "node", false),
	kindToKLogFieldPair("core/v1", "pod", "pod", true),
	kindToKLogFieldPair("core/v1", "namespace", "namespace", false),
}

// ControllerManagerComponentPatser handle logs from controller-manager,
type ControllerManagerComponentParser struct{}

// Process implements ControlPlaneComponentParser.
func (c *ControllerManagerComponentParser) Process(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) (bool, error) {
	mainMessageFieldSet := log.MustGetFieldSet(l, &log.MainMessageFieldSet{})
	mainMsg := mainMessageFieldSet.MainMessage
	// Event logs emitted from controller manager
	// The event message could start from Event occurred" or "Event occurred"
	if strings.HasPrefix(strings.TrimPrefix(mainMsg, "\""), "Event occurred\"") {
		path, err := c.eventLogToResourcePath(l)
		if err != nil {
			return true, nil
		}
		cs.RecordEvent(path)
	} else {
		kindLogResourcePath, err := c.kindLogToResourcePath(ctx, l)
		if err == nil {
			cs.RecordEvent(kindLogResourcePath)
		}
		paths, err := c.controllerLogToResourcePath(l)
		if err == nil {
			for _, path := range paths {
				cs.RecordEvent(path)
			}
		}
	}
	return true, nil
}

// ShouldProcess implements ControlPlaneComponentParser.
func (c *ControllerManagerComponentParser) ShouldProcess(component_name string) bool {
	return component_name == "controller-manager"
}

var _ ControlPlaneComponentParser = (*ControllerManagerComponentParser)(nil)

func (*ControllerManagerComponentParser) kindLogToResourcePath(ctx context.Context, l *log.Log) (resourcepath.ResourcePath, error) {
	mainMessageFieldSet := log.MustGetFieldSet(l, &log.MainMessageFieldSet{})
	if !mainMessageFieldSet.HasKLogField("kind") {
		return resourcepath.ResourcePath{}, fmt.Errorf("kind field wasn't found from the log")
	}
	kind, err := mainMessageFieldSet.KLogField("kind")
	if err != nil {
		return resourcepath.ResourcePath{}, fmt.Errorf("kind field not found from the log")
	}
	kind = strings.ToLower(kind)
	key, err := mainMessageFieldSet.KLogField("key")
	if err != nil || key == "" {
		return resourcepath.ResourcePath{}, fmt.Errorf("key field not found from the log")
	}
	for _, pair := range kindToKLogFieldPairs {
		if pair.KindName == kind {
			if pair.IsNamespaced {
				splittedField := strings.Split(key, "/")
				if len(splittedField) != 2 {
					continue
				}
				return resourcepath.NameLayerGeneralItem(pair.APIVersion, pair.KindName, splittedField[0], splittedField[1]), nil
			} else {
				return resourcepath.NameLayerGeneralItem(pair.APIVersion, pair.KindName, "cluster-scope", key), nil
			}
		}
	}
	slog.WarnContext(ctx, fmt.Sprintf("kind %s is not coverred in the parser", kind), logger.LogKind(fmt.Sprintf("controller-manager-component-missing-support-%s", kind)))
	return resourcepath.ResourcePath{}, fmt.Errorf("kind %s is not coverred in the parser", kind)
}

// controllerLogToResourcePath returns the list of resource path parsed by controller specific klog parser
// Example format: "Too few replicas" replicaSet="kube-system/kube-dns-68b67b4c6f" need=2 creating=1
func (*ControllerManagerComponentParser) controllerLogToResourcePath(l *log.Log) ([]resourcepath.ResourcePath, error) {
	mainMessageFieldSet := log.MustGetFieldSet(l, &log.MainMessageFieldSet{})
	result := []resourcepath.ResourcePath{}
	for _, pair := range kindToKLogFieldPairs {
		field, err := mainMessageFieldSet.KLogField(pair.KLogField)
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
	return result, nil
}

// eventLogToResourcePath returns the resource path with checking fields in KLog format of the given log entry.
// Example format: "Event occurred" object="gmp-system/collector" fieldPath="" kind="DaemonSet" apiVersion="apps/v1" type="Normal" reason="SuccessfulCreate" message="Created pod: collector-fwbmm"
func (*ControllerManagerComponentParser) eventLogToResourcePath(l *log.Log) (resourcepath.ResourcePath, error) {
	mainMessageFieldSet := log.MustGetFieldSet(l, &log.MainMessageFieldSet{})
	var namespace string
	var name string
	obj, err := mainMessageFieldSet.KLogField("object")
	if err != nil || obj == "" {
		return resourcepath.ResourcePath{}, fmt.Errorf("failed to read object from klog")
	}
	if strings.Contains(obj, "/") {
		parts := strings.Split(obj, "/")
		namespace = parts[0]
		name = parts[1]
	} else {
		namespace = "cluster-scope"
		name = obj
	}
	kind, err := mainMessageFieldSet.KLogField("kind")
	if err != nil || kind == "" {
		return resourcepath.ResourcePath{}, fmt.Errorf("failed to read kind from klog")
	}
	apiVersion, err := mainMessageFieldSet.KLogField("apiVersion")
	if err != nil || apiVersion == "" {
		return resourcepath.ResourcePath{}, fmt.Errorf("failed to read apiVersion from klog")
	}
	return resourcepath.NameLayerGeneralItem(apiVersion, strings.ToLower(kind), namespace, name), nil
}
