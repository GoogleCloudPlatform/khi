// Copyright 2026 Google LLC
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

package commonlogk8saudit_contract

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

var irregularPluralToSingularSuffixMap = map[string]string{
	"classes":      "class",
	"ingresses":    "ingress",
	"leases":       "lease",
	"dnses":        "dns",
	"identities":   "identity",
	"policies":     "policy",
	"topologies":   "topology",
	"statuses":     "status",
	"capabilities": "capability",
}

// GetSingularKindName converts a plural Kubernetes resource kind to its singular form.
func GetSingularKindName(pluralKind string) string {
	if strings.HasSuffix(pluralKind, "ses") || strings.HasSuffix(pluralKind, "ies") {
		for pluralSuffix, singularSuffix := range irregularPluralToSingularSuffixMap {
			if strings.HasSuffix(pluralKind, pluralSuffix) {
				return strings.TrimSuffix(pluralKind, pluralSuffix) + singularSuffix
			}
		}
		return pluralKind
	}
	if strings.HasSuffix(pluralKind, "s") {
		return strings.TrimSuffix(pluralKind, "s")
	}
	return pluralKind
}

// MutatingWebhookPatch represents a single JSON patch operation from a webhook.
type MutatingWebhookPatch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value"`
}

const (
	// MutatingWebhookMutationPrefix is the prefix for mutating webhook mutation labels.
	MutatingWebhookMutationPrefix = "mutation.webhook.admission.k8s.io/"
	// MutatingWebhookPatchPrefix is the prefix for mutating webhook patch labels.
	MutatingWebhookPatchPrefix = "patch.webhook.admission.k8s.io/"
	// MutatingWebhookFailedOpenPrefix is the prefix for mutating webhook failed-open labels.
	MutatingWebhookFailedOpenPrefix = "failed-open.mutation.webhook.admission.k8s.io/"
)

// MutatingWebhookMutationInfo contains information about webhook mutation configuration.
type MutatingWebhookMutationInfo struct {
	Configuration string `json:"configuration"`
	Webhook       string `json:"webhook"`
	Mutated       bool   `json:"mutated"`
}

// MutatingWebhookPatchInfo contains information about webhook mutation patch.
type MutatingWebhookPatchInfo struct {
	Configuration string                 `json:"configuration"`
	Webhook       string                 `json:"webhook"`
	Patch         []MutatingWebhookPatch `json:"patch"`
}

// MutatingWebhookResult holds the results of a mutating webhook invocation.
type MutatingWebhookResult struct {
	Round         int
	Index         int
	Configuration string
	Webhook       string
	Mutated       bool
	Patch         []MutatingWebhookPatch
	FailedOpen    bool
}

// K8sAuditLogFieldSet is the field set for k8s audit log.
type K8sAuditLogFieldSet struct {
	// OperationID is the ID of the operation.
	OperationID string
	// IsFirst is true if the log is the first log of the operation.
	IsFirst bool
	// IsLast is true if the log is the last log of the operation.
	IsLast bool

	// APIVersion is the API version of the resource (e.g., "apps/v1").
	APIVersion string
	// PluralKind is the plural resource kind name (e.g., "pods", "services").
	PluralKind string
	// Namespace is the namespace of the resource.
	Namespace string
	// ResourceName is the name of the resource.
	ResourceName string
	// SubresourceName is the subresource name if applicable (e.g., "status", "binding").
	SubresourceName string
	// ClusterName is the name of the Kubernetes cluster.
	ClusterName string
	// Verb is the styled operation revision verb.
	Verb *pb.Verb

	// RequestURI is the request URI.
	RequestURI string
	// Principal is the principal who issued the request.
	Principal string
	// StatusCode is the status code of the response.
	StatusCode int
	// StatusMessage is the status message of the response.
	StatusMessage string
	// IsError is true if the response is an error.
	IsError bool
	// Request is the request body.
	Request *structured.NodeReader
	// Response is the response body.
	Response *structured.NodeReader
	// MutatingWebhookResults are the assembled webhook results from audit annotations.
	MutatingWebhookResults []*MutatingWebhookResult
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
	if k.Verb == nil {
		return ""
	}
	return k.Verb.GetLabel()
}

var _ log.FieldSet = (*K8sAuditLogFieldSet)(nil)
