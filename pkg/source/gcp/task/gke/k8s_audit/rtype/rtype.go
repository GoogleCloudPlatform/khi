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

package rtype

// rtype.Type indicates the schema of log bodies under request or response.
type Type = int

const (
	RTypeUnknown       Type = 0
	RTypePatch         Type = 1
	RTypeDeleteOptions Type = 2
	RTypeSatus         Type = 3
	RTypeUnusedEnd
)

var Types map[string]Type = map[string]Type{
	"k8s.io/Patch":                         RTypePatch,
	"meta.k8s.io/__internal.DeleteOptions": RTypeDeleteOptions,
	"core.k8s.io/v1.Status":                RTypeSatus,
}
