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

package model

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
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

type KubernetesObjectOperation struct {
	APIVersion      string
	PluralKind      string
	Namespace       string
	Name            string
	SubResourceName string
	Verb            enum.RevisionVerb
}

func (o *KubernetesObjectOperation) CovertToResourcePath() string {
	if o.SubResourceName != "" {
		return strings.ToLower(strings.Join([]string{
			o.APIVersion,
			o.GetSingularKindName(),
			o.Namespace,
			o.Name,
			o.SubResourceName,
		}, "#"))
	} else {
		return strings.ToLower(strings.Join([]string{
			o.APIVersion,
			o.GetSingularKindName(),
			o.Namespace,
			o.Name,
		}, "#"))
	}
}

func (o *KubernetesObjectOperation) GetSingularKindName() string {
	if strings.HasSuffix(o.PluralKind, "ses") || strings.HasSuffix(o.PluralKind, "ies") {
		for pluralSuffix, singularSuffix := range irregularPluralToSingularSuffixMap {
			if strings.HasSuffix(o.PluralKind, pluralSuffix) {
				return strings.TrimSuffix(o.PluralKind, pluralSuffix) + singularSuffix
			}
		}
		slog.Warn(fmt.Sprintf("unknown singular name for %s", o.PluralKind))
		return o.PluralKind
	}
	if strings.HasSuffix(o.PluralKind, "s") {
		return strings.TrimSuffix(o.PluralKind, "s")
	}
	slog.Error(fmt.Sprintf("unknown plural form %s!", o.PluralKind))
	return o.PluralKind
}
