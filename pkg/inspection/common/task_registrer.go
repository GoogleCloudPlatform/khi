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

package inspection_common

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
)

func PrepareInspectionServer(rootServer *inspection.InspectionTaskServer) error {
	err := rootServer.AddTask(inspection_task.InspectionTimeProducer)
	if err != nil {
		return err
	}
	return nil
}
