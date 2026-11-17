// Copyright 2026 Google LLC
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

package scene

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

type indexedYAMLDocument struct {
	lines []string
	root  *indexedYAMLNode
}

type indexedYAMLNode struct {
	node      *yaml.Node
	keyNode   *yaml.Node
	startLine int // 1-based inclusive line in lines.
	endLine   int // 1-based inclusive line in lines.
	byKey     map[string]*indexedYAMLNode
	keyOrder  []string
	seqItems  []*indexedYAMLNode
}

type matchedYAMLNode struct {
	target    *indexedYAMLNode
	highlight *indexedYAMLNode
}

type yamlLineSelection struct {
	startLine     int // 1-based inclusive.
	endLine       int // 1-based inclusive.
	highlightLine int // 1-based line (0 when none).
	highlightSpan byteSpan
}

func parseIndexedYAMLDocument(yamlText string) (*indexedYAMLDocument, error) {
	lines := workbench.SplitLines(yamlText)
	if len(lines) == 0 {
		return &indexedYAMLDocument{}, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(yamlText), &doc); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	if len(doc.Content) == 0 {
		return &indexedYAMLDocument{lines: lines}, nil
	}
	root := buildIndexedYAMLNode(doc.Content[0], nil, 1, len(lines)+1, lines)
	return &indexedYAMLDocument{
		lines: lines,
		root:  root,
	}, nil
}

func buildIndexedYAMLNode(
	node *yaml.Node,
	keyNode *yaml.Node,
	minStartLine int,
	nextBoundaryLine int,
	lines []string,
) *indexedYAMLNode {
	if node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	startLine := nodeFirstLine(node, keyNode, minStartLine, lines)
	endLine := min(len(lines), max(startLine, nextBoundaryLine-1))
	for endLine != startLine && (strings.TrimSpace(lines[endLine-1]) == "" || lines[endLine-1] == "...") {
		endLine--
	}

	idxNode := &indexedYAMLNode{
		node:      node,
		keyNode:   keyNode,
		startLine: startLine,
		endLine:   endLine,
	}
	switch node.Kind {
	case yaml.MappingNode:
		buildMappingYAMLNode(idxNode, nextBoundaryLine, lines)
	case yaml.SequenceNode:
		buildSequenceYAMLNode(idxNode, nextBoundaryLine, lines)
	}
	return idxNode
}

func buildMappingYAMLNode(idxNode *indexedYAMLNode, nextBoundaryLine int, lines []string) {
	node := idxNode.node
	pairCount := len(node.Content) / 2
	byKey := make(map[string]*indexedYAMLNode, pairCount)
	keyOrder := make([]string, 0, pairCount)
	for i := range pairCount {
		kNode := node.Content[2*i]
		vNode := node.Content[2*i+1]
		childBoundary := nextBoundaryLine
		if i+1 < pairCount {
			childBoundary = node.Content[2*(i+1)].Line
		}
		child := buildIndexedYAMLNode(vNode, kNode, idxNode.startLine, childBoundary, lines)
		if _, exists := byKey[kNode.Value]; !exists {
			keyOrder = append(keyOrder, kNode.Value)
		}
		byKey[kNode.Value] = child
	}
	idxNode.byKey = byKey
	idxNode.keyOrder = keyOrder
}

func buildSequenceYAMLNode(idxNode *indexedYAMLNode, nextBoundaryLine int, lines []string) {
	node := idxNode.node
	itemCount := len(node.Content)
	seqItems := make([]*indexedYAMLNode, itemCount)
	itemStarts := make([]int, itemCount)
	for i, item := range node.Content {
		itemStarts[i] = nodeFirstLine(item, nil, idxNode.startLine, lines)
	}
	for i, item := range node.Content {
		childBoundary := nextBoundaryLine
		if i+1 < itemCount {
			childBoundary = itemStarts[i+1]
		}
		child := buildIndexedYAMLNode(item, nil, itemStarts[i], childBoundary, lines)
		child.startLine = itemStarts[i]
		seqItems[i] = child
	}
	idxNode.seqItems = seqItems
}

func nodeFirstLine(node *yaml.Node, keyNode *yaml.Node, minStartLine int, lines []string) int {
	if keyNode != nil {
		return min(keyNode.Line, len(lines))
	}
	line := clampInt(node.Line, 1, len(lines))
	// Capture a standalone sequence '-' line immediately above a block mapping item.
	if line-1 > minStartLine && strings.TrimSpace(lines[line-2]) == "-" {
		line--
	}
	return line
}

func evalCompiledJSONPath(doc *indexedYAMLDocument, path compiledJSONPath) []yamlLineSelection {
	if doc.root == nil {
		return nil
	}
	matched := evalSteps([]*indexedYAMLNode{doc.root}, path.steps)
	if len(matched) == 0 {
		return nil
	}
	selections := make([]yamlLineSelection, 0, len(matched))
	for _, m := range matched {
		hlLine, hlSpan := computeNodeHighlight(doc.lines, m)
		selections = append(selections, yamlLineSelection{
			startLine:     m.target.startLine,
			endLine:       m.target.endLine,
			highlightLine: hlLine,
			highlightSpan: hlSpan,
		})
	}
	return selections
}

func computeNodeHighlight(lines []string, m matchedYAMLNode) (int, byteSpan) {
	t := m.target
	switch t.node.Kind {
	case yaml.ScalarNode:
		return trimmedContentSpanOnLine(lines, t.startLine)
	case yaml.MappingNode, yaml.SequenceNode:
		if t.keyNode != nil {
			return trimmedContentSpanOnLine(lines, t.keyNode.Line)
		}
		if m.highlight != nil {
			return trimmedContentSpanOnLine(lines, m.highlight.startLine)
		}
	}
	return 0, byteSpan{}
}

func trimmedContentSpanOnLine(lines []string, lineNum int) (int, byteSpan) {
	if lineNum < 1 || lineNum > len(lines) {
		return 0, byteSpan{}
	}
	line := lines[lineNum-1]
	trimmedLeft := strings.TrimLeft(line, " \t")
	if trimmedLeft == "" {
		return 0, byteSpan{}
	}
	start := len(line) - len(trimmedLeft)
	if strings.HasPrefix(trimmedLeft, "- ") {
		start += 2
	}
	return lineNum, byteSpan{start: start, end: len(line)}
}

func evalSteps(roots []*indexedYAMLNode, steps []jsonPathStep) []matchedYAMLNode {
	current := make([]matchedYAMLNode, len(roots))
	for i, r := range roots {
		current[i] = matchedYAMLNode{target: r}
	}
	for _, step := range steps {
		current = advanceMatchedNodes(current, step)
		if len(current) == 0 {
			return nil
		}
	}
	return current
}

func advanceMatchedNodes(current []matchedYAMLNode, step jsonPathStep) []matchedYAMLNode {
	var next []matchedYAMLNode
	for _, item := range current {
		candidates := []*indexedYAMLNode{item.target}
		if step.recursive {
			candidates = collectSelfAndDescendants(item.target)
		}
		for _, cand := range candidates {
			next = append(next, applySingleStep(cand, step)...)
		}
	}
	return next
}

func collectSelfAndDescendants(n *indexedYAMLNode) []*indexedYAMLNode {
	out := []*indexedYAMLNode{n}
	for _, k := range n.keyOrder {
		out = append(out, collectSelfAndDescendants(n.byKey[k])...)
	}
	for _, item := range n.seqItems {
		out = append(out, collectSelfAndDescendants(item)...)
	}
	return out
}

func applySingleStep(cand *indexedYAMLNode, step jsonPathStep) []matchedYAMLNode {
	switch step.kind {
	case stepKindKeys:
		return applyKeyStep(cand, step.keys)
	case stepKindWildcard:
		return applyWildcardStep(cand)
	case stepKindIndices:
		return applyIndicesStep(cand, step.indices)
	case stepKindSlice:
		return applySliceStep(cand, step.sliceFrom, step.sliceTo)
	default:
		return applyFilterStep(cand, step.filter)
	}
}

func applyKeyStep(cand *indexedYAMLNode, keys []string) []matchedYAMLNode {
	if cand.node.Kind != yaml.MappingNode {
		return nil
	}
	var out []matchedYAMLNode
	for _, k := range keys {
		if child, ok := cand.byKey[k]; ok {
			out = append(out, matchedYAMLNode{target: child})
		}
	}
	return out
}

func applyWildcardStep(cand *indexedYAMLNode) []matchedYAMLNode {
	switch cand.node.Kind {
	case yaml.MappingNode:
		out := make([]matchedYAMLNode, 0, len(cand.keyOrder))
		for _, k := range cand.keyOrder {
			out = append(out, matchedYAMLNode{target: cand.byKey[k]})
		}
		return out
	case yaml.SequenceNode:
		out := make([]matchedYAMLNode, len(cand.seqItems))
		for i, item := range cand.seqItems {
			out[i] = matchedYAMLNode{target: item}
		}
		return out
	default:
		return nil
	}
}

func applyIndicesStep(cand *indexedYAMLNode, indices []int) []matchedYAMLNode {
	if cand.node.Kind != yaml.SequenceNode {
		return nil
	}
	n := len(cand.seqItems)
	var out []matchedYAMLNode
	for _, rawIdx := range indices {
		idx := rawIdx
		if idx < 0 {
			idx = n + idx
		}
		if idx >= 0 && idx < n {
			out = append(out, matchedYAMLNode{target: cand.seqItems[idx]})
		}
	}
	return out
}

func applySliceStep(cand *indexedYAMLNode, from, to *int) []matchedYAMLNode {
	if cand.node.Kind != yaml.SequenceNode {
		return nil
	}
	n := len(cand.seqItems)
	lo := resolveSliceBound(from, 0, n)
	hi := max(lo, resolveSliceBound(to, n, n))
	out := make([]matchedYAMLNode, 0, hi-lo)
	for i := lo; i < hi; i++ {
		out = append(out, matchedYAMLNode{target: cand.seqItems[i]})
	}
	return out
}

func resolveSliceBound(bound *int, defaultVal, seqLen int) int {
	v := defaultVal
	if bound != nil {
		v = *bound
		if v < 0 {
			v += seqLen
		}
	}
	return clampInt(v, 0, seqLen)
}

func applyFilterStep(cand *indexedYAMLNode, expr filterExpr) []matchedYAMLNode {
	var out []matchedYAMLNode
	switch cand.node.Kind {
	case yaml.SequenceNode:
		for _, item := range cand.seqItems {
			if ok, hl := expr.eval(item); ok {
				out = append(out, matchedYAMLNode{target: item, highlight: hl})
			}
		}
	case yaml.MappingNode:
		for _, k := range cand.keyOrder {
			child := cand.byKey[k]
			if ok, hl := expr.eval(child); ok {
				out = append(out, matchedYAMLNode{target: child, highlight: hl})
			}
		}
	}
	return out
}

// diagnoseUnmatchedJSONPath finds the deepest matched prefix across docs and returns its available child keys.
func diagnoseUnmatchedJSONPath(path compiledJSONPath, docs ...*indexedYAMLDocument) (matchedPrefix string, availableKeys []string) {
	var current []matchedYAMLNode
	for _, d := range docs {
		if d.root != nil {
			current = append(current, matchedYAMLNode{target: d.root})
		}
	}
	if len(current) == 0 {
		return "", nil
	}

	var prefixBuilder strings.Builder
	prefixBuilder.WriteByte('$')
	matchedDepth := 0
	for _, step := range path.steps {
		next := advanceMatchedNodes(current, step)
		if len(next) == 0 {
			break
		}
		prefixBuilder.WriteString(step.raw)
		current = next
		matchedDepth++
	}

	keys := collectAvailableKeys(current)
	if matchedDepth == 0 {
		return "", keys
	}
	return prefixBuilder.String(), keys
}

func collectAvailableKeys(matched []matchedYAMLNode) []string {
	keySet := make(map[string]struct{})
	for _, m := range matched {
		for _, k := range m.target.keyOrder {
			keySet[k] = struct{}{}
		}
		// When the deepest matched node is a sequence of mappings, surface element keys when no direct mapping keys exist.
		if len(m.target.keyOrder) == 0 {
			for _, item := range m.target.seqItems {
				for _, k := range item.keyOrder {
					keySet[k] = struct{}{}
				}
			}
		}
	}
	return slices.Sorted(maps.Keys(keySet))
}
