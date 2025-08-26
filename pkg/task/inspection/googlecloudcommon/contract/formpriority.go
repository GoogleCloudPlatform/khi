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

package googlecloudcommon_contract

var (
	// FormBasePriority is the base priority for Google Cloud common forms.
	FormBasePriority = 100000
	// PriorityForQueryTimeGroup is the priority for the query time group.
	PriorityForQueryTimeGroup = FormBasePriority + 50000
	// PriorityForResourceIdentifierGroup is the priority for the resource identifier group.
	PriorityForResourceIdentifierGroup = FormBasePriority + 40000
	// PriorityForK8sResourceFilterGroup is the priority for the k8s resource filter group.
	PriorityForK8sResourceFilterGroup = FormBasePriority + 30000
)
