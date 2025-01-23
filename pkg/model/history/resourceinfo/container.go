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

package resourceinfo

import (
	"fmt"
	"reflect"
	"sync"

	corev1 "k8s.io/api/core/v1"
)

type ContainerStatuses struct {
	lastObservedStatus map[string]corev1.ContainerStatus
	statusMapLock      sync.Mutex
}

func (c *ContainerStatuses) IsNewChange(namespace string, podname string, containerName string, status corev1.ContainerStatus) bool {
	c.statusMapLock.Lock()
	defer c.statusMapLock.Unlock()
	path := fmt.Sprintf("%s#%s#%s", namespace, podname, containerName)
	if last, found := c.lastObservedStatus[path]; !found {
		c.lastObservedStatus[path] = last
		return true
	} else {
		if !reflect.DeepEqual(last, status) {
			c.lastObservedStatus[path] = last
			return true
		} else {
			return false
		}
	}
}
