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

package structured

// orderedMapNode wraps a map Node and customizes its Children() iteration order.
type orderedMapNode struct {
	inner        Node
	priorityKeys []string
}

// WithKeyOrder wraps a map Node to customize its Children() iteration order.
// It yields children with keys matching priorityKeys in the specified order first,
// followed by any remaining children in their original order.
// If the node is not a MapNodeType or priorityKeys is empty, it returns the original node.
func WithKeyOrder(node Node, priorityKeys ...string) Node {
	if node == nil || node.Type() != MapNodeType || len(priorityKeys) == 0 {
		return node
	}
	if ordered, ok := node.(*orderedMapNode); ok {
		return &orderedMapNode{
			inner:        ordered.inner,
			priorityKeys: priorityKeys,
		}
	}
	return &orderedMapNode{
		inner:        node,
		priorityKeys: priorityKeys,
	}
}

// Type implements Node.
func (o *orderedMapNode) Type() NodeType {
	return o.inner.Type()
}

// NodeScalarValue implements Node.
func (o *orderedMapNode) NodeScalarValue() (any, error) {
	return o.inner.NodeScalarValue()
}

// Len implements Node.
func (o *orderedMapNode) Len() int {
	return o.inner.Len()
}

// Children implements Node.
func (o *orderedMapNode) Children() NodeChildrenIterator {
	return func(callback func(key NodeChildrenKey, value Node) bool) {
		type entry struct {
			key   NodeChildrenKey
			value Node
		}
		entries := make([]entry, 0, o.inner.Len())
		keyToIndex := make(map[string]int, o.inner.Len())

		for k, v := range o.inner.Children() {
			keyToIndex[k.Key] = len(entries)
			entries = append(entries, entry{key: k, value: v})
		}

		visited := make([]bool, len(entries))

		// 1. Yield priority keys in specified order with O(1) lookup
		for _, pk := range o.priorityKeys {
			if idx, ok := keyToIndex[pk]; ok && !visited[idx] {
				visited[idx] = true
				if !callback(entries[idx].key, entries[idx].value) {
					return
				}
			}
		}

		// 2. Yield remaining keys in original order
		for i, e := range entries {
			if !visited[i] {
				if !callback(e.key, e.value) {
					return
				}
			}
		}
	}
}

var _ Node = (*orderedMapNode)(nil)
