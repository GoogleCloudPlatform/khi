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

package common

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/v2commonlogparse"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/v2logconvert"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/v2manifestgenerate"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/v2timelinegrouping"
)

func Register(i *inspection.InspectionTaskServer) error {
	err := i.AddTask(v2commonlogparse.Task)
	if err != nil {
		return err
	}

	err = i.AddTask(v2timelinegrouping.Task)
	if err != nil {
		return err
	}

	err = i.AddTask(v2manifestgenerate.Task)
	if err != nil {
		return err
	}

	err = i.AddTask(v2logconvert.Task)
	if err != nil {
		return err
	}
	return nil
}
