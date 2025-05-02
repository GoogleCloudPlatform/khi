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

package structurev2

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/merger"
)

var _ MergeMapOrderStrategy = (*DefaultMergeMapOrderStrategy)(nil)

// MergeNode merge a previous node with patch node and generates a new Node.
// This patch supports strategic-merge patch https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md
// Refer the following mermaid graph to understand call hierarchy.
// ```mermaid
// flowchart TD
//
//	MergeNode --> mergeNode
//	mergeNode -->|when the node is scalar| mergeScalarNode
//	mergeNode -->|when the node is sequence| mergeSequenceNode
//	mergeNode -->|when the node is map| mergeMapNode
//	mergeSequenceNode -->|when the sequence items are scalar| mergeScalarSequenceNode
//	mergeSequenceNode -->|when the sequence items are sequence| mergeSequenceSequenceNode
//	mergeSequenceNode -->|when the sequence items are map| mergeMapSequenceNode
//	mergeMapSequenceNode -->|when the patch policy is replace| mergeMapSequenceNodeWithReplaceStrategy
//	mergeMapSequenceNode -->|when the patch policy is merge| mergeMapSequenceNodeWithMergeStrategy
//
//	mergeMapNode o-..->|for each items| mergeNode
//	mergeSequenceSequenceNode o-..->|for each sequences| mergeNode
//	mergeMapSequenceNodeWithReplaceStrategy o-..->|for each maps| mergeNode
//	mergeMapSequenceNodeWithMergeStrategy o-.->|for each maps| mergeNode
//
// ```
func MergeNode(prev Node, patch Node, config MergeConfiguration) (Node, error) {
	return mergeNode([]string{}, prev, patch, config)
}

func mergeNode(fieldPath []string, prev Node, patch Node, config MergeConfiguration) (Node, error) {
	if patch != nil {
		var err error
		patch, config, err = handleStrategicMergePatchDirectives(fieldPath, patch, config)
		if err != nil {
			return nil, err
		}
		if config.patchDirectiveDelete {
			return nil, nil
		}
	}

	if prev != nil && patch != nil {
		if prev.Type() != patch.Type() {
			// prev node type and patch node type is different, use replace strategy
			return cloneStandardNodeFromNode(patch)
		}
	}
	var nodeType NodeType
	if prev != nil {
		nodeType = prev.Type()
	} else if patch != nil {
		nodeType = patch.Type()
	}
	switch nodeType {
	case ScalarNodeType:
		return mergeScalarNode(prev, patch)
	case SequenceNodeType:
		return mergeSequenceNode(fieldPath, prev, patch, config)
	case MapNodeType:
		return mergeMapNode(fieldPath, prev, patch, config)
	default:
		return nil, fmt.Errorf("unknown node type %v", nodeType)
	}
}

func mergeScalarNode(prev Node, patch Node) (Node, error) {
	if patch == nil {
		if prev == nil {
			return nil, nil
		}
		return cloneStandardNodeFromNode(prev)
	}
	return cloneStandardNodeFromNode(patch) // replace policy
}

func mergeSequenceNode(fieldPath []string, prev Node, patch Node, config MergeConfiguration) (Node, error) {
	isFirstNode := true
	var sequenceChildNodeType NodeType
	if prev != nil {
		for _, value := range prev.Children() {
			nodeType := value.Type()
			if !isFirstNode && nodeType != sequenceChildNodeType {
				return nil, fmt.Errorf("child node type mismatch in a sequence node")
			}
			sequenceChildNodeType = nodeType
			isFirstNode = false
		}
	}
	if patch != nil {
		for _, value := range patch.Children() {
			nodeType := value.Type()
			if !isFirstNode && nodeType != sequenceChildNodeType {
				return nil, fmt.Errorf("child node type mismatch in a sequence node")
			}
			sequenceChildNodeType = nodeType
			isFirstNode = false
		}
	}

	fieldPath = append(fieldPath, "[]")
	defer func() {
		fieldPath = fieldPath[:len(fieldPath)-1]
	}()

	switch sequenceChildNodeType {
	case ScalarNodeType:
		return mergeScalarSequenceNode(fieldPath, prev, patch, config)
	case SequenceNodeType:
		return mergeSequenceSequenceNode(fieldPath, prev, patch, config)
	case MapNodeType:
		return mergeMapSequenceNode(fieldPath, prev, patch, config)
	default:
		return nil, fmt.Errorf("unknown node type %v", sequenceChildNodeType)
	}
}

func mergeScalarSequenceNode(fieldPath []string, prev Node, patch Node, config MergeConfiguration) (Node, error) {
	sequenceNode := StandardSequenceNode{
		value: []Node{},
	}

	copyFrom := patch
	if copyFrom == nil {
		copyFrom = prev
	}

	if config.setElementOrderDirectiveList != nil { // When $setElementOrder is used for primitive list, the order list become the sequence itself. https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md#list-of-primitives
		for _, value := range config.setElementOrderDirectiveList {
			scalarChild := StandardScalarNode[string]{
				value: value,
			}
			sequenceNode.value = append(sequenceNode.value, &scalarChild)
		}
		return &sequenceNode, nil
	}

	for _, value := range copyFrom.Children() {
		// if the element is included in the parent $deleteFromPrimitiveList, then the element is ignored.
		if len(config.deleteFromPrimitiveListDirectiveList) > 0 {
			value, err := getScalarAs[string](value)
			if err != nil {
				return nil, err
			}
			if _, found := config.deleteFromPrimitiveListDirectiveList[value]; found {
				continue
			}
		}
		clonedPrimitive, err := cloneStandardNodeFromNode(value)
		if err != nil {
			return nil, err
		}
		sequenceNode.value = append(sequenceNode.value, clonedPrimitive)
	}

	return &sequenceNode, nil
}

func mergeSequenceSequenceNode(fieldPath []string, prev Node, patch Node, config MergeConfiguration) (Node, error) {
	sequenceNode := StandardSequenceNode{
		value: []Node{},
	}

	copyFrom := patch
	if copyFrom == nil {
		copyFrom = prev
	}

	for _, value := range copyFrom.Children() {
		// sequence children of children may have directives. It needs to be merged with nil.
		mergedNode, err := mergeNode(fieldPath, nil, value, config)
		if err != nil {
			return nil, err
		}
		if mergedNode != nil {
			sequenceNode.value = append(sequenceNode.value, value)
		}
	}
	return &sequenceNode, nil
}

func mergeMapSequenceNode(fieldPath []string, prev Node, patch Node, config MergeConfiguration) (Node, error) {
	strategy, mergeKey, err := config.GetArrayMergeStrategyAndKey(fieldPath)
	if err != nil {
		return nil, err
	}
	if strategy == merger.MergeStrategyReplace {
		return mergeMapSequenceNodeWithReplaceStrategy(fieldPath, prev, patch, config)
	} else {
		return mergeMapSequenceNodeWithMergeStrategy(fieldPath, mergeKey, prev, patch, config)
	}
}

func mergeMapSequenceNodeWithReplaceStrategy(fieldPath []string, prev Node, patch Node, config MergeConfiguration) (Node, error) {
	if patch == nil {
		return cloneStandardNodeFromNode(prev)
	}

	sequenceNode := StandardSequenceNode{
		value: []Node{},
	}
	for _, value := range patch.Children() {
		mergedNode, err := mergeNode(fieldPath, nil, value, config)
		if err != nil {
			return nil, err
		}
		if mergedNode == nil {
			continue
		}
		sequenceNode.value = append(sequenceNode.value, mergedNode)
	}
	return &sequenceNode, nil
}

func mergeMapSequenceNodeWithMergeStrategy(fieldPath []string, mergeKey string, prev Node, patch Node, config MergeConfiguration) (Node, error) {
	sequenceNode := StandardSequenceNode{
		value: []Node{},
	}

	var err error
	prevValues := map[string]Node{}
	prevItemKeys := []string{}
	if prev != nil {
		for _, value := range prev.Children() {
			var itemKey string
			for keyInChild, valueOfKeyInChild := range value.Children() {
				if keyInChild.Key == mergeKey {
					itemKey, err = getScalarAs[string](valueOfKeyInChild)
					if err != nil {
						return nil, err
					}
					break
				}
			}
			if itemKey == "" {
				return nil, fmt.Errorf("merge sequence key not found in array at %s (merge key %s)", strings.Join(fieldPath, "."), mergeKey)
			}
			prevValues[itemKey] = value
			prevItemKeys = append(prevItemKeys, itemKey)
		}
	}

	patchValues := map[string]Node{}
	patchItemKeys := []string{}
	if patch != nil {
		for _, value := range patch.Children() {
			var itemKey string
			for keyInChild, valueOfKeyInChild := range value.Children() {
				if keyInChild.Key == mergeKey {
					itemKey, err = getScalarAs[string](valueOfKeyInChild)
					if err != nil {
						return nil, err
					}
					break
				}
			}
			if itemKey == "" {
				return nil, fmt.Errorf("merge sequence key not found in array at %s (merge key %s)", strings.Join(fieldPath, "."), mergeKey)
			}
			patchValues[itemKey] = value
			patchItemKeys = append(patchItemKeys, itemKey)
		}
	}

	if config.setElementOrderDirectiveList != nil {
		for _, itemKey := range config.setElementOrderDirectiveList {
			var mergedNode Node
			prev := prevValues[itemKey]
			patch := patchValues[itemKey]
			if prev == nil && patch == nil {
				// if the item is not found in prev structure and patch but the order is given, add an object with the item key.
				mergedNode = &StandardMapNode{
					keys: []string{mergeKey},
					values: []Node{
						&StandardScalarNode[string]{
							value: itemKey,
						},
					},
				}
			} else {
				mergedNode, err = mergeNode(fieldPath, prev, patch, config)
				if err != nil {
					return nil, err
				}
				if mergedNode == nil {
					continue
				}
			}
			sequenceNode.value = append(sequenceNode.value, mergedNode)
		}
		return &sequenceNode, nil
	}

	for _, itemKey := range prevItemKeys {
		if _, found := patchValues[itemKey]; !found {
			mergedNode, err := mergeNode(fieldPath, prevValues[itemKey], nil, config)
			if err != nil {
				return nil, err
			}
			if mergedNode == nil {
				continue
			}
			sequenceNode.value = append(sequenceNode.value, mergedNode)
		}
	}
	for _, itemKey := range patchItemKeys {
		prev := prevValues[itemKey]
		patch := patchValues[itemKey]
		mergedNode, err := mergeNode(fieldPath, prev, patch, config)
		if err != nil {
			return nil, err
		}
		if mergedNode == nil {
			continue
		}
		sequenceNode.value = append(sequenceNode.value, mergedNode)
	}
	return &sequenceNode, nil
}

func mergeMapNode(fieldPath []string, prev Node, patch Node, config MergeConfiguration) (Node, error) {
	if config.patchDirectiveReplace {
		return cloneStandardNodeFromNode(patch)
	}
	prevKeys := []string{}
	prevValues := map[string]Node{}
	if prev != nil {
		for key, value := range prev.Children() {
			prevKeys = append(prevKeys, key.Key)
			prevValues[key.Key] = value
		}
	}

	patchKeys := []string{}
	patchValues := map[string]Node{}
	if patch != nil {
		for key, value := range patch.Children() {
			patchKeys = append(patchKeys, key.Key)
			patchValues[key.Key] = value
		}
	}

	// find keys only existing in the strategic patch-merge directives
	directiveKeysForChildren := []string{}
	defaultPrevForDirectiveOnlyChildren := map[string]Node{} // default node structure for the nodes not included in patch or prev.
	if config.setElementOrderListForChildren != nil {
		for key, itemKeyValues := range config.setElementOrderListForChildren {
			_, foundInPrev := prevValues[key]
			_, foundInPatch := patchValues[key]
			if foundInPatch || foundInPrev {
				continue
			}
			fieldPath = append(fieldPath, key)
			fieldPath = append(fieldPath, "[]")
			directiveKeysForChildren = append(directiveKeysForChildren, key)
			_, itemKey, err := config.GetArrayMergeStrategyAndKey(fieldPath)
			if err != nil {
				return nil, err
			}
			sequenceNodeInferredFromDirective := &StandardSequenceNode{
				value: []Node{},
			}
			if itemKey == "" { // the sequence is primitive list
				for _, itemKeyValue := range itemKeyValues {
					sequenceNodeInferredFromDirective.value = append(sequenceNodeInferredFromDirective.value, &StandardScalarNode[string]{
						value: itemKeyValue,
					})
				}
			} else {
				for _, itemKeyValue := range itemKeyValues {
					sequenceNodeInferredFromDirective.value = append(sequenceNodeInferredFromDirective.value, &StandardMapNode{
						keys: []string{itemKey},
						values: []Node{
							&StandardScalarNode[string]{
								value: itemKeyValue,
							},
						},
					})
				}
			}
			fieldPath = fieldPath[:len(fieldPath)-2]
			defaultPrevForDirectiveOnlyChildren[key] = sequenceNodeInferredFromDirective
		}
	}

	orderedKeys, err := config.MergeMapOrderStrategy.GetMergedKeyOrder(prevKeys, patchKeys, directiveKeysForChildren)
	if err != nil {
		return nil, err
	}

	mapNode := StandardMapNode{
		keys:   []string{},
		values: []Node{},
	}

	for _, key := range orderedKeys {
		childConfig := config
		prevNode := prevValues[key]
		patchNode := patchValues[key]
		childConfig.deleteFromPrimitiveListDirectiveListForChildren = nil
		childConfig.retainKeysDirectiveListForChildren = nil
		childConfig.setElementOrderListForChildren = nil

		if config.deleteFromPrimitiveListDirectiveListForChildren != nil && config.deleteFromPrimitiveListDirectiveListForChildren[key] != nil {
			childConfig.deleteFromPrimitiveListDirectiveList = config.deleteFromPrimitiveListDirectiveListForChildren[key]
		}

		if config.retainKeysDirectiveListForChildren != nil && config.retainKeysDirectiveListForChildren[key] != nil {
			childConfig.retainKeysDirectiveList = config.retainKeysDirectiveListForChildren[key]
		}

		if config.setElementOrderListForChildren != nil && config.setElementOrderListForChildren[key] != nil {
			childConfig.setElementOrderDirectiveList = config.setElementOrderListForChildren[key]
		}

		if config.retainKeysDirectiveList != nil {
			if _, found := config.retainKeysDirectiveList[key]; !found {
				continue
			}
		}

		fieldPath = append(fieldPath, key)
		if prevNode == nil && patchNode == nil {
			prevNode = defaultPrevForDirectiveOnlyChildren[key]
		}
		mergedNode, err := mergeNode(fieldPath, prevNode, patchNode, childConfig)
		if err != nil {
			return nil, err
		}
		fieldPath = fieldPath[:len(fieldPath)-1]

		if mergedNode == nil {
			continue
		}
		mapNode.keys = append(mapNode.keys, key)
		mapNode.values = append(mapNode.values, mergedNode)
	}

	return &mapNode, nil
}

// handleStrategicMergePatchDirectives reads the strategic patch directives like $patch, $deleteFromPrimitiveList, $setElementOrder ...etc defined in https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md#list-of-maps-2
// It reads a structured data representing the patch request and merge configuration, and returns new patch structured data omitting these specific fields and updated merge configuration with these directives.
func handleStrategicMergePatchDirectives(fieldPath []string, patch Node, parentConfig MergeConfiguration) (newPatch Node, newConfig MergeConfiguration, err error) {
	if patch.Type() != MapNodeType {
		return patch, parentConfig, nil
	}
	newConfig = parentConfig
	mapNode := &StandardMapNode{
		keys:   []string{},
		values: []Node{},
	}

	for key, value := range patch.Children() {
		keySlashSeparatedSegments := strings.Split(key.Key, "/")
		switch keySlashSeparatedSegments[0] {
		case "$patch":
			patchDirective, err := getScalarAs[string](value)
			if err != nil {
				return nil, MergeConfiguration{}, err
			}
			switch patchDirective {
			case "replace":
				newConfig.patchDirectiveReplace = true
			case "delete":
				newConfig.patchDirectiveDelete = true
			case "merge": // It's default. ignore.
				continue
			default:
				return nil, MergeConfiguration{}, fmt.Errorf("unknown patch directive %s", patchDirective)
			}
		case "$deleteFromPrimitiveList":
			if value.Type() != SequenceNodeType {
				return nil, MergeConfiguration{}, fmt.Errorf("$deleteFromPrimitiveList must be a sequence node")
			}
			primitiveList := map[string]struct{}{}
			for _, child := range value.Children() {
				value, err := getScalarAs[string](child)
				if err != nil {
					return nil, MergeConfiguration{}, err
				}
				primitiveList[value] = struct{}{}
			}
			if newConfig.deleteFromPrimitiveListDirectiveList == nil {
				newConfig.deleteFromPrimitiveListDirectiveListForChildren = map[string]map[string]struct{}{}
			}
			newConfig.deleteFromPrimitiveListDirectiveListForChildren[strings.TrimPrefix(key.Key, "$deleteFromPrimitiveList/")] = primitiveList
		case "$retainKeys":
			if value.Type() != SequenceNodeType {
				return nil, MergeConfiguration{}, fmt.Errorf("$retainKeys must be a sequence node")
			}
			retainKeysList := map[string]struct{}{}
			for _, child := range value.Children() {
				value, err := getScalarAs[string](child)
				if err != nil {
					return nil, MergeConfiguration{}, err
				}
				retainKeysList[value] = struct{}{}
			}
			if newConfig.retainKeysDirectiveListForChildren == nil {
				newConfig.retainKeysDirectiveListForChildren = map[string]map[string]struct{}{}
			}
			newConfig.retainKeysDirectiveListForChildren[strings.TrimPrefix(key.Key, "$retainKeys/")] = retainKeysList
		case "$setElementOrder":
			if value.Type() != SequenceNodeType {
				return nil, MergeConfiguration{}, fmt.Errorf("$retainKeys must be a sequence node")
			}
			setElementOrderList := []string{}
			for _, child := range value.Children() {
				switch child.Type() {
				case ScalarNodeType: // https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md#list-of-primitives
					value, err := getScalarAs[string](child)
					if err != nil {
						return nil, MergeConfiguration{}, err
					}
					setElementOrderList = append(setElementOrderList, value)
				case MapNodeType: // https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md#list-of-maps-2
					var keyValue string
					for _, value := range child.Children() {
						keyValue, err = getScalarAs[string](value)
						if err != nil {
							return nil, MergeConfiguration{}, err
						}
						break
					}
					setElementOrderList = append(setElementOrderList, keyValue)
				default:
					return nil, MergeConfiguration{}, fmt.Errorf("$setElementOrder must be a sequence node of maps or scalars")
				}
			}
			if newConfig.setElementOrderListForChildren == nil {
				newConfig.setElementOrderListForChildren = map[string][]string{}
			}
			newConfig.setElementOrderListForChildren[strings.TrimPrefix(key.Key, "$setElementOrder/")] = setElementOrderList
		default:
			mapNode.keys = append(mapNode.keys, key.Key)
			mapNode.values = append(mapNode.values, value)
		}
	}
	newPatch = mapNode
	return
}
