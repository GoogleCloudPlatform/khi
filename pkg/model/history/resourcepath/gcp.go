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

// NetworkEndpointGroup returns the ResourcePath of timeline for NEG.
func NetworkEndpointGroup(negNamespace string, negName string) ResourcePath {
	if negNamespace == "" {
		negNamespace = nonSpecifiedPlaceholder
	}
	if negName == "" {
		negName = nonSpecifiedPlaceholder
	}
	return NameLayerGeneralItem("networking.gke.io/v1beta1", "servicenetworkendpointgroup", negNamespace, negName)
}
