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

package noderesource

import (
	"strings"
	"sync"
)

// LogBinder finds ids/names in log body and returns associated resource associattion.
type LogBinder struct {
	nodeLogBinderMutex sync.RWMutex
	nodeLogBinders     map[string]*nodeLogBinder
}

// NewLogBinder returns a new instance of LogBinder
func NewLogBinder() *LogBinder {
	return &LogBinder{
		nodeLogBinderMutex: sync.RWMutex{},
		nodeLogBinders:     map[string]*nodeLogBinder{},
	}
}

// AddResourceBinding adds a new resource related logs on the specific node.
func (n *LogBinder) AddResourceBinding(nodeName string, ra ResourceBinding) {
	n.nodeLogBinderMutex.Lock()
	defer n.nodeLogBinderMutex.Unlock()
	if _, ok := n.nodeLogBinders[nodeName]; !ok {
		n.nodeLogBinders[nodeName] = newNodeLogBinder()
	}
	n.nodeLogBinders[nodeName].AddResourceBinding(ra)
}

// GetBoundResourcesForLogBody returns the array of ResourceBinding bound to the given log on the node.
func (n *LogBinder) GetBoundResourcesForLogBody(nodeName string, logBody string) []ResourceBinding {
	n.nodeLogBinderMutex.RLock()
	defer n.nodeLogBinderMutex.RUnlock()
	if _, ok := n.nodeLogBinders[nodeName]; !ok {
		return []ResourceBinding{}
	}
	return n.nodeLogBinders[nodeName].GetBoundResourcesForLogBody(logBody)
}

type nodeLogBinder struct {
	nodeResourceBindingsMutex sync.RWMutex
	nodeResourceBindings      []ResourceBinding
}

func newNodeLogBinder() *nodeLogBinder {
	return &nodeLogBinder{
		nodeResourceBindingsMutex: sync.RWMutex{},
		nodeResourceBindings:      []ResourceBinding{},
	}
}

// GetBoundResourcesForLogBody returns the array of ResourceBinding bound to the given log.
func (n *nodeLogBinder) GetBoundResourcesForLogBody(logBody string) []ResourceBinding {
	n.nodeResourceBindingsMutex.RLock()
	defer n.nodeResourceBindingsMutex.RUnlock()

	result := []ResourceBinding{}
	for _, ra := range n.nodeResourceBindings {
		uniqueIdentifier := ra.GetUniqueIdentifier()
		if strings.Contains(logBody, uniqueIdentifier) {
			result = append(result, ra)
		}
	}
	return result
}

// AddResourceBinding adds the given ResourceBinding when it's not already registered.
func (n *nodeLogBinder) AddResourceBinding(ra ResourceBinding) {
	resourcePath := ra.GetResourcePath()
	uniqueIdentifier := ra.GetUniqueIdentifier()
	n.nodeResourceBindingsMutex.Lock()
	defer n.nodeResourceBindingsMutex.Unlock()

	// ignore adding it when it found in the list already.
	for _, ra := range n.nodeResourceBindings {
		if ra.GetResourcePath() == resourcePath && ra.GetUniqueIdentifier() == uniqueIdentifier {
			return
		}
	}
	n.nodeResourceBindings = append(n.nodeResourceBindings, ra)
}
