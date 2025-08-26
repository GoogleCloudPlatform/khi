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

package resourcepath

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

func TestAPIVersionLayerGeneralItem(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		want       string
	}{
		{
			name:       "basic",
			apiVersion: "core/v1",
			want:       "core/v1",
		},
		{
			name:       "core package can omit the package domain",
			apiVersion: "v1",
			want:       "core/v1",
		},
		{
			name:       "empty",
			apiVersion: "",
			want:       "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourcePath := APIVersionLayerGeneralItem(tt.apiVersion)
			if resourcePath.Path != tt.want {
				t.Errorf("APIVersionLayerGeneralItem(%s).Path=%v, want %v", tt.apiVersion, resourcePath.Path, tt.want)
			}
			if resourcePath.ParentRelationship != enum.RelationshipChild {
				t.Errorf("APIVersionLayerGeneralItem(%s).ParentRelationship=%q, want %q", tt.apiVersion, resourcePath.ParentRelationship, enum.RelationshipChild)
			}
		})
	}
}

func TestKindLayerGeneralItem(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		kind       string
		want       string
	}{
		{
			name:       "basic",
			apiVersion: "v1",
			kind:       "Pod",
			want:       "core/v1#Pod",
		},
		{
			name:       "empty apiVersion",
			apiVersion: "",
			kind:       "Pod",
			want:       "unknown#Pod",
		},
		{
			name:       "empty kind",
			apiVersion: "v1",
			kind:       "",
			want:       "core/v1#unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourcePath := KindLayerGeneralItem(tt.apiVersion, tt.kind)
			if resourcePath.Path != tt.want {
				t.Errorf("KindLayerGeneralItem(%s,%s).Path=%v, want %v", tt.apiVersion, tt.kind, resourcePath.Path, tt.want)
			}
			if resourcePath.ParentRelationship != enum.RelationshipChild {
				t.Errorf("KindLayerGeneralItem(%s,%s).ParentRelationship=%q, want %q", tt.apiVersion, tt.kind, resourcePath.ParentRelationship, enum.RelationshipChild)
			}
		})
	}
}

func TestNamespaceLayerGeneralItem(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		kind       string
		namespace  string
		want       string
	}{
		{
			name:       "basic",
			apiVersion: "v1",
			kind:       "Pod",
			namespace:  "default",
			want:       "core/v1#Pod#default",
		},
		{
			name:       "empty apiVersion",
			apiVersion: "",
			kind:       "Pod",
			namespace:  "default",
			want:       "unknown#Pod#default",
		},
		{
			name:       "empty kind",
			apiVersion: "v1",
			kind:       "",
			namespace:  "default",
			want:       "core/v1#unknown#default",
		},
		{
			name:       "empty namespace",
			apiVersion: "v1",
			kind:       "Pod",
			namespace:  "",
			want:       "core/v1#Pod#unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourcePath := NamespaceLayerGeneralItem(tt.apiVersion, tt.kind, tt.namespace)
			if resourcePath.Path != tt.want {
				t.Errorf("NamespaceLayerGeneralItem(%s,%s,%s).Path=%v, want %v", tt.apiVersion, tt.kind, tt.namespace, resourcePath.Path, tt.want)
			}
			if resourcePath.ParentRelationship != enum.RelationshipChild {
				t.Errorf("NamespaceLayerGeneralItem(%s,%s,%s).ParentRelationship=%q, want %q", tt.apiVersion, tt.kind, tt.namespace, resourcePath.ParentRelationship, enum.RelationshipChild)
			}
		})
	}
}

func TestNameLayerGeneralItem(t *testing.T) {
	tests := []struct {
		tname      string
		apiVersion string
		kind       string
		namespace  string
		name       string
		want       string
	}{
		{
			tname:      "basic",
			apiVersion: "v1",
			kind:       "Pod",
			namespace:  "default",
			name:       "my-pod",
			want:       "core/v1#Pod#default#my-pod",
		},
		{
			tname:      "empty apiVersion",
			apiVersion: "",
			kind:       "Pod",
			namespace:  "default",
			name:       "my-pod",
			want:       "unknown#Pod#default#my-pod",
		},
		{
			tname:      "empty kind",
			apiVersion: "v1",
			kind:       "",
			namespace:  "default",
			name:       "my-pod",
			want:       "core/v1#unknown#default#my-pod",
		},
		{
			tname:      "empty namespace",
			apiVersion: "v1",
			kind:       "Pod",
			namespace:  "",
			name:       "my-pod",
			want:       "core/v1#Pod#unknown#my-pod",
		},
		{
			tname:      "empty name",
			apiVersion: "v1",
			kind:       "Pod",
			namespace:  "default",
			name:       "",
			want:       "core/v1#Pod#default#unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.tname, func(t *testing.T) {
			resourcePath := NameLayerGeneralItem(tt.apiVersion, tt.kind, tt.namespace, tt.name)
			if resourcePath.Path != tt.want {
				t.Errorf("NameLayerGeneralItem(%s,%s,%s,%s).Path=%v, want %v", tt.apiVersion, tt.kind, tt.namespace, tt.name, resourcePath.Path, tt.want)
			}
			if resourcePath.ParentRelationship != enum.RelationshipChild {
				t.Errorf("NameLayerGeneralItem(%s,%s,%s,%s).ParentRelationship=%q, want %q", tt.apiVersion, tt.kind, tt.namespace, tt.name, resourcePath.ParentRelationship, enum.RelationshipChild)
			}
		})
	}
}

func TestSubresourceLayerGeneralItem(t *testing.T) {
	tests := []struct {
		tname       string
		apiVersion  string
		kind        string
		namespace   string
		name        string
		subresource string
		want        string
	}{
		{
			tname:       "basic",
			apiVersion:  "v1",
			kind:        "Pod",
			namespace:   "default",
			name:        "my-pod",
			subresource: "status",
			want:        "core/v1#Pod#default#my-pod#status",
		},
		{
			tname:       "empty apiVersion",
			apiVersion:  "",
			kind:        "Pod",
			namespace:   "default",
			name:        "my-pod",
			subresource: "status",
			want:        "unknown#Pod#default#my-pod#status",
		},
		{
			tname:       "empty kind",
			apiVersion:  "v1",
			kind:        "",
			namespace:   "default",
			name:        "my-pod",
			subresource: "status",
			want:        "core/v1#unknown#default#my-pod#status",
		},
		{
			tname:       "empty namespace",
			apiVersion:  "v1",
			kind:        "Pod",
			namespace:   "",
			name:        "my-pod",
			subresource: "status",
			want:        "core/v1#Pod#unknown#my-pod#status",
		},
		{
			tname:       "empty name",
			apiVersion:  "v1",
			kind:        "Pod",
			namespace:   "default",
			name:        "",
			subresource: "status",
			want:        "core/v1#Pod#default#unknown#status",
		},
		{
			tname:       "empty subresource",
			apiVersion:  "v1",
			kind:        "Pod",
			namespace:   "default",
			name:        "my-pod",
			subresource: "",
			want:        "core/v1#Pod#default#my-pod#unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.tname, func(t *testing.T) {
			resourcePath := SubresourceLayerGeneralItem(tt.apiVersion, tt.kind, tt.namespace, tt.name, tt.subresource)
			if resourcePath.Path != tt.want {
				t.Errorf("SubresourceLayerGeneralItem(%s,%s,%s,%s,%s).Path=%v, want %v", tt.apiVersion, tt.kind, tt.namespace, tt.name, tt.subresource, resourcePath.Path, tt.want)
			}
			if resourcePath.ParentRelationship != enum.RelationshipChild {
				t.Errorf("SubresourceLayerGeneralItem(%s,%s,%s,%s,%s).ParentRelationship=%q, want %q", tt.apiVersion, tt.kind, tt.namespace, tt.name, tt.subresource, resourcePath.ParentRelationship, enum.RelationshipChild)
			}
		})
	}
}
