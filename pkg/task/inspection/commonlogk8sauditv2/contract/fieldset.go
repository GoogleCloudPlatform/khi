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

package commonlogk8sauditv2_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

type K8sAuditLogFieldSet struct {
	OperationID   string
	IsFirst       bool
	IsLast        bool
	K8sOperation  *model.KubernetesObjectOperation
	RequestURI    string
	Principal     string
	StatusCode    int
	StatusMessage string
	IsError       bool
	Request       *structured.NodeReader
	Response      *structured.NodeReader
}

// Kind implements log.FieldSet.
func (k *K8sAuditLogFieldSet) Kind() string {
	return "k8s_audit_log"
}

// LongRunning returns true if the log is a long-running operation.
func (k *K8sAuditLogFieldSet) LongRunning() bool {
	return (k.IsFirst && !k.IsLast) || (!k.IsFirst && k.IsLast)
}

// VerbString returns the string representation of the verb.
func (k *K8sAuditLogFieldSet) VerbString() string {
	if k.K8sOperation == nil {
		return ""
	}
	return enum.RevisionVerbs[k.K8sOperation.Verb].Label
}

var _ log.FieldSet = (*K8sAuditLogFieldSet)(nil)
