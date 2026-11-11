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
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"gopkg.in/yaml.v3"
)

// InvalidJSONPathError reports a missing or syntactically invalid JSONPath selector.
type InvalidJSONPathError struct {
	// PathIndex is the 0-based index in Paths, or -1 when Paths itself is empty.
	PathIndex int
	// Path is the raw selector string that failed validation.
	Path string
	// Reason describes why the selector is invalid.
	Reason string
}

// Error formats the invalid JSONPath selector error message.
func (e *InvalidJSONPathError) Error() string {
	if e.PathIndex < 0 {
		return fmt.Sprintf("invalid JSONPath list: %s", e.Reason)
	}
	return fmt.Sprintf("invalid JSONPath %d (%q): %s", e.PathIndex, e.Path, e.Reason)
}

type jsonPathStepKind int

const (
	stepKindKeys jsonPathStepKind = iota
	stepKindWildcard
	stepKindIndices
	stepKindSlice
	stepKindFilter
)

type jsonPathStep struct {
	kind      jsonPathStepKind
	recursive bool
	keys      []string
	indices   []int
	sliceFrom *int
	sliceTo   *int
	filter    filterExpr
	raw       string
}

type compiledJSONPath struct {
	index int
	raw   string
	steps []jsonPathStep
}

type filterExpr interface {
	eval(candidate *indexedYAMLNode) (matched bool, highlight *indexedYAMLNode)
}

type filterOrExpr struct {
	left  filterExpr
	right filterExpr
}

func (e filterOrExpr) eval(candidate *indexedYAMLNode) (bool, *indexedYAMLNode) {
	if ok, hl := e.left.eval(candidate); ok {
		return true, hl
	}
	return e.right.eval(candidate)
}

type filterAndExpr struct {
	left  filterExpr
	right filterExpr
}

func (e filterAndExpr) eval(candidate *indexedYAMLNode) (bool, *indexedYAMLNode) {
	okL, hlL := e.left.eval(candidate)
	if !okL {
		return false, nil
	}
	okR, hlR := e.right.eval(candidate)
	if !okR {
		return false, nil
	}
	if hlL != nil {
		return true, hlL
	}
	return true, hlR
}

type filterOp int

const (
	filterOpExists filterOp = iota
	filterOpEq
	filterOpNe
	filterOpLt
	filterOpLe
	filterOpGt
	filterOpGe
)

type literalKind int

const (
	literalString literalKind = iota
	literalNumber
	literalBool
	literalNull
)

type filterLiteral struct {
	kind    literalKind
	strVal  string
	numVal  float64
	boolVal bool
}

type filterAtomExpr struct {
	relSteps []jsonPathStep
	op       filterOp
	rhs      filterLiteral
}

func (e filterAtomExpr) eval(candidate *indexedYAMLNode) (bool, *indexedYAMLNode) {
	targets := evalSteps([]*indexedYAMLNode{candidate}, e.relSteps)
	if e.op == filterOpExists {
		for _, m := range targets {
			if m.target != nil && m.target.node != nil && m.target.node.Tag != "!!null" {
				return true, m.target
			}
		}
		return false, nil
	}
	for _, m := range targets {
		if compareNodeWithLiteral(m.target.node, e.op, e.rhs) {
			return true, m.target
		}
	}
	return false, nil
}

func compareNodeWithLiteral(node *yaml.Node, op filterOp, lit filterLiteral) bool {
	if node == nil || node.Kind != yaml.ScalarNode {
		return op == filterOpNe && lit.kind != literalNull
	}
	isNull := node.Tag == "!!null"
	if lit.kind == literalNull {
		if op == filterOpEq {
			return isNull
		}
		if op == filterOpNe {
			return !isNull
		}
		return false
	}
	if isNull {
		return op == filterOpNe
	}

	switch lit.kind {
	case literalBool:
		if b, err := strconv.ParseBool(node.Value); err == nil {
			if op == filterOpEq {
				return b == lit.boolVal
			}
			if op == filterOpNe {
				return b != lit.boolVal
			}
		}
		return false
	case literalNumber:
		if v, err := strconv.ParseFloat(node.Value, 64); err == nil {
			switch op {
			case filterOpEq:
				return v == lit.numVal
			case filterOpNe:
				return v != lit.numVal
			case filterOpLt:
				return v < lit.numVal
			case filterOpLe:
				return v <= lit.numVal
			case filterOpGt:
				return v > lit.numVal
			case filterOpGe:
				return v >= lit.numVal
			}
		}
		return false
	case literalString:
		switch op {
		case filterOpEq:
			return node.Value == lit.strVal
		case filterOpNe:
			return node.Value != lit.strVal
		case filterOpLt:
			return node.Value < lit.strVal
		case filterOpLe:
			return node.Value <= lit.strVal
		case filterOpGt:
			return node.Value > lit.strVal
		case filterOpGe:
			return node.Value >= lit.strVal
		}
	}
	return false
}

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

func compileJSONPaths(paths []string) ([]compiledJSONPath, error) {
	if len(paths) == 0 {
		return nil, &InvalidJSONPathError{
			PathIndex: -1,
			Reason:    "paths must contain at least one JSONPath selector",
		}
	}
	compiled := make([]compiledJSONPath, len(paths))
	for i, raw := range paths {
		steps, err := parseJSONPathSteps(raw, false)
		if err != nil {
			return nil, &InvalidJSONPathError{
				PathIndex: i,
				Path:      raw,
				Reason:    err.Error(),
			}
		}
		compiled[i] = compiledJSONPath{
			index: i,
			raw:   raw,
			steps: steps,
		}
	}
	return compiled, nil
}

func parseJSONPathSteps(raw string, allowEmptySteps bool) ([]jsonPathStep, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, fmt.Errorf("JSONPath must not be empty")
	}
	pos := 0
	if s[pos] == '$' {
		pos++
	}
	var steps []jsonPathStep
	for pos < len(s) {
		switch {
		case strings.HasPrefix(s[pos:], ".."):
			stepStart := pos
			pos += 2
			if pos >= len(s) {
				return nil, fmt.Errorf("trailing '..' in JSONPath")
			}
			switch s[pos] {
			case '[':
				step, nextPos, err := parseBracketStep(s, pos, true)
				if err != nil {
					return nil, err
				}
				step.raw = s[stepStart:nextPos]
				steps = append(steps, step)
				pos = nextPos
			case '*':
				pos++
				steps = append(steps, jsonPathStep{
					kind:      stepKindWildcard,
					recursive: true,
					raw:       s[stepStart:pos],
				})
			default:
				key, nextPos, err := parseBareIdentifier(s, pos)
				if err != nil {
					return nil, err
				}
				steps = append(steps, jsonPathStep{
					kind:      stepKindKeys,
					recursive: true,
					keys:      []string{key},
					raw:       s[stepStart:nextPos],
				})
				pos = nextPos
			}

		case s[pos] == '.':
			stepStart := pos
			pos++
			if pos >= len(s) {
				return nil, fmt.Errorf("trailing '.' in JSONPath")
			}
			if s[pos] == '[' {
				return nil, fmt.Errorf("unexpected '[' after '.'; use '[...]' without a preceding dot")
			}
			if s[pos] == '*' {
				pos++
				steps = append(steps, jsonPathStep{
					kind: stepKindWildcard,
					raw:  s[stepStart:pos],
				})
				continue
			}
			key, nextPos, err := parseBareIdentifier(s, pos)
			if err != nil {
				return nil, err
			}
			steps = append(steps, jsonPathStep{
				kind: stepKindKeys,
				keys: []string{key},
				raw:  s[stepStart:nextPos],
			})
			pos = nextPos

		case s[pos] == '[':
			stepStart := pos
			step, nextPos, err := parseBracketStep(s, pos, false)
			if err != nil {
				return nil, err
			}
			step.raw = s[stepStart:nextPos]
			steps = append(steps, step)
			pos = nextPos

		default:
			if len(steps) > 0 {
				return nil, fmt.Errorf("unexpected character %q at offset %d", s[pos], pos)
			}
			// Allow leading bare field name when '$' and '.' are omitted (e.g. "spec.replicas").
			stepStart := pos
			if s[pos] == '*' {
				pos++
				steps = append(steps, jsonPathStep{
					kind: stepKindWildcard,
					raw:  "." + s[stepStart:pos],
				})
				continue
			}
			key, nextPos, err := parseBareIdentifier(s, pos)
			if err != nil {
				return nil, err
			}
			steps = append(steps, jsonPathStep{
				kind: stepKindKeys,
				keys: []string{key},
				raw:  "." + s[stepStart:nextPos],
			})
			pos = nextPos
		}
	}

	if len(steps) == 0 && !allowEmptySteps {
		return nil, fmt.Errorf("JSONPath must select a field or element beneath root '$' (e.g. '$.spec' or '$.status')")
	}
	return steps, nil
}

func parseBareIdentifier(s string, start int) (string, int, error) {
	var sb strings.Builder
	pos := start
	for pos < len(s) {
		ch := s[pos]
		if ch == '\\' {
			if pos+1 < len(s) && s[pos+1] == '.' {
				sb.WriteByte('.')
				pos += 2
				continue
			}
			sb.WriteByte('\\')
			pos++
			continue
		}
		if ch == '.' || ch == '[' {
			break
		}
		if ch == ']' || ch == ' ' || ch == '\t' {
			return "", pos, fmt.Errorf("unexpected character %q in field name", ch)
		}
		sb.WriteByte(ch)
		pos++
	}
	if sb.Len() == 0 {
		return "", pos, fmt.Errorf("empty field name in JSONPath")
	}
	return sb.String(), pos, nil
}

func parseBracketStep(s string, openIdx int, recursive bool) (jsonPathStep, int, error) {
	closeIdx, err := findMatchingBracket(s, openIdx)
	if err != nil {
		return jsonPathStep{}, 0, err
	}
	inner := strings.TrimSpace(s[openIdx+1 : closeIdx])
	nextPos := closeIdx + 1
	if inner == "" {
		return jsonPathStep{}, 0, fmt.Errorf("empty bracket selector '[]'")
	}
	if inner == "*" {
		return jsonPathStep{kind: stepKindWildcard, recursive: recursive}, nextPos, nil
	}
	if inner[0] == '?' {
		expr, err := parseFilterExpression(strings.TrimSpace(inner[1:]))
		if err != nil {
			return jsonPathStep{}, 0, err
		}
		return jsonPathStep{
			kind:      stepKindFilter,
			recursive: recursive,
			filter:    expr,
		}, nextPos, nil
	}
	if inner[0] == '\'' || inner[0] == '"' {
		keys, err := parseQuotedKeyUnion(inner)
		if err != nil {
			return jsonPathStep{}, 0, err
		}
		return jsonPathStep{
			kind:      stepKindKeys,
			recursive: recursive,
			keys:      keys,
		}, nextPos, nil
	}
	if strings.Contains(inner, ":") {
		from, to, err := parseSliceBounds(inner)
		if err != nil {
			return jsonPathStep{}, 0, err
		}
		return jsonPathStep{
			kind:      stepKindSlice,
			recursive: recursive,
			sliceFrom: from,
			sliceTo:   to,
		}, nextPos, nil
	}
	indices, err := parseIndexUnion(inner)
	if err != nil {
		return jsonPathStep{}, 0, err
	}
	return jsonPathStep{
		kind:      stepKindIndices,
		recursive: recursive,
		indices:   indices,
	}, nextPos, nil
}

func findMatchingBracket(s string, openIdx int) (int, error) {
	depth := 0
	var quote byte
	for i := openIdx; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if ch == '\\' {
				i++
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unclosed '[' in JSONPath")
}

func parseQuotedKeyUnion(inner string) ([]string, error) {
	var keys []string
	pos := 0
	for pos < len(inner) {
		for pos < len(inner) && (inner[pos] == ' ' || inner[pos] == '\t') {
			pos++
		}
		if pos >= len(inner) {
			return nil, fmt.Errorf("trailing comma in bracket key selector")
		}
		val, nextPos, err := parseQuotedString(inner, pos)
		if err != nil {
			return nil, err
		}
		keys = append(keys, val)
		pos = nextPos
		for pos < len(inner) && (inner[pos] == ' ' || inner[pos] == '\t') {
			pos++
		}
		if pos < len(inner) {
			if inner[pos] != ',' {
				return nil, fmt.Errorf("expected ',' or ']' after quoted key in bracket selector")
			}
			pos++
			if pos >= len(inner) {
				return nil, fmt.Errorf("trailing comma in bracket key selector")
			}
		}
	}
	return keys, nil
}

func parseQuotedString(s string, start int) (string, int, error) {
	if start >= len(s) || (s[start] != '\'' && s[start] != '"') {
		return "", start, fmt.Errorf("expected quoted string")
	}
	quote := s[start]
	var sb strings.Builder
	for i := start + 1; i < len(s); i++ {
		ch := s[i]
		if ch == '\\' {
			i++
			if i >= len(s) {
				return "", i, fmt.Errorf("unclosed quoted string")
			}
			switch s[i] {
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			default:
				sb.WriteByte(s[i])
			}
			continue
		}
		if ch == quote {
			return sb.String(), i + 1, nil
		}
		sb.WriteByte(ch)
	}
	return "", len(s), fmt.Errorf("unclosed quoted string")
}

func parseSliceBounds(inner string) (*int, *int, error) {
	parts := strings.Split(inner, ":")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("slice selector %q supports only [start:end]", inner)
	}
	var from, to *int
	if left := strings.TrimSpace(parts[0]); left != "" {
		v, err := strconv.Atoi(left)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid slice start %q", left)
		}
		from = &v
	}
	if right := strings.TrimSpace(parts[1]); right != "" {
		v, err := strconv.Atoi(right)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid slice end %q", right)
		}
		to = &v
	}
	return from, to, nil
}

func parseIndexUnion(inner string) ([]int, error) {
	parts := strings.Split(inner, ",")
	indices := make([]int, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		v, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid array selector %q (quote string keys as ['%s'])", trimmed, trimmed)
		}
		indices = append(indices, v)
	}
	return indices, nil
}

type filterParser struct {
	src string
	pos int
}

func parseFilterExpression(raw string) (filterExpr, error) {
	if raw == "" {
		return nil, fmt.Errorf("empty filter expression after '?'")
	}
	p := &filterParser{src: raw}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	p.skipWS()
	if p.pos < len(p.src) {
		return nil, fmt.Errorf("unexpected trailing token %q in filter expression", p.src[p.pos:])
	}
	return expr, nil
}

func (p *filterParser) skipWS() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t') {
		p.pos++
	}
}

func (p *filterParser) parseOr() (filterExpr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		p.skipWS()
		if !strings.HasPrefix(p.src[p.pos:], "||") {
			return left, nil
		}
		p.pos += 2
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = filterOrExpr{left: left, right: right}
	}
}

func (p *filterParser) parseAnd() (filterExpr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		p.skipWS()
		if !strings.HasPrefix(p.src[p.pos:], "&&") {
			return left, nil
		}
		p.pos += 2
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		left = filterAndExpr{left: left, right: right}
	}
}

func (p *filterParser) parsePrimary() (filterExpr, error) {
	p.skipWS()
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("unexpected end of filter expression")
	}
	if p.src[p.pos] == '(' {
		p.pos++
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		p.skipWS()
		if p.pos >= len(p.src) || p.src[p.pos] != ')' {
			return nil, fmt.Errorf("unclosed '(' in filter expression")
		}
		p.pos++
		return expr, nil
	}
	return p.parseAtom()
}

func (p *filterParser) parseAtom() (filterExpr, error) {
	p.skipWS()
	if p.pos >= len(p.src) || p.src[p.pos] != '@' {
		return nil, fmt.Errorf("filter predicate must start with '@' (e.g. [?(@.type=='Ready')])")
	}
	p.pos++ // Consume '@'.
	pathStart := p.pos
	bracketDepth := 0
	var quote byte
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		if quote != 0 {
			if ch == '\\' {
				p.pos += 2
				continue
			}
			if ch == quote {
				quote = 0
			}
			p.pos++
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			p.pos++
			continue
		}
		if ch == '[' {
			bracketDepth++
			p.pos++
			continue
		}
		if ch == ']' && bracketDepth > 0 {
			bracketDepth--
			p.pos++
			continue
		}
		if bracketDepth == 0 && (ch == ' ' || ch == '\t' || ch == '=' || ch == '!' || ch == '<' || ch == '>' || ch == '&' || ch == '|' || ch == ')') {
			break
		}
		p.pos++
	}

	relRaw := p.src[pathStart:p.pos]
	var relSteps []jsonPathStep
	if relRaw != "" {
		var err error
		relSteps, err = parseJSONPathSteps("$"+relRaw, true)
		if err != nil {
			return nil, fmt.Errorf("invalid relative path '@%s': %w", relRaw, err)
		}
	}

	p.skipWS()
	op, hasOp, err := p.parseComparisonOp()
	if err != nil {
		return nil, err
	}
	if !hasOp {
		return filterAtomExpr{relSteps: relSteps, op: filterOpExists}, nil
	}

	p.skipWS()
	lit, err := p.parseLiteral()
	if err != nil {
		return nil, err
	}
	return filterAtomExpr{relSteps: relSteps, op: op, rhs: lit}, nil
}

func (p *filterParser) parseComparisonOp() (filterOp, bool, error) {
	if p.pos >= len(p.src) {
		return 0, false, nil
	}
	rem := p.src[p.pos:]
	switch {
	case strings.HasPrefix(rem, "=="):
		p.pos += 2
		return filterOpEq, true, nil
	case strings.HasPrefix(rem, "!="):
		p.pos += 2
		return filterOpNe, true, nil
	case strings.HasPrefix(rem, "<="):
		p.pos += 2
		return filterOpLe, true, nil
	case strings.HasPrefix(rem, ">="):
		p.pos += 2
		return filterOpGe, true, nil
	case strings.HasPrefix(rem, "<"):
		p.pos++
		return filterOpLt, true, nil
	case strings.HasPrefix(rem, ">"):
		p.pos++
		return filterOpGt, true, nil
	case strings.HasPrefix(rem, "="):
		return 0, false, fmt.Errorf("use '==' instead of '=' in filter expression")
	default:
		return 0, false, nil
	}
}

func (p *filterParser) parseLiteral() (filterLiteral, error) {
	if p.pos >= len(p.src) {
		return filterLiteral{}, fmt.Errorf("missing comparison value in filter expression")
	}
	if p.src[p.pos] == '\'' || p.src[p.pos] == '"' {
		val, nextPos, err := parseQuotedString(p.src, p.pos)
		if err != nil {
			return filterLiteral{}, err
		}
		p.pos = nextPos
		return filterLiteral{kind: literalString, strVal: val}, nil
	}
	start := p.pos
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		if ch == ' ' || ch == '\t' || ch == ')' || ch == '&' || ch == '|' {
			break
		}
		p.pos++
	}
	token := p.src[start:p.pos]
	switch token {
	case "":
		return filterLiteral{}, fmt.Errorf("missing comparison value in filter expression")
	case "true":
		return filterLiteral{kind: literalBool, boolVal: true}, nil
	case "false":
		return filterLiteral{kind: literalBool, boolVal: false}, nil
	case "null":
		return filterLiteral{kind: literalNull}, nil
	default:
		num, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return filterLiteral{}, fmt.Errorf("invalid filter literal %q (quote strings with single or double quotes)", token)
		}
		return filterLiteral{kind: literalNumber, numVal: num}, nil
	}
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
	if node != nil && node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}
	totalLines := len(lines)
	startLine := nodeFirstLine(node, keyNode, minStartLine, lines)
	endLine := min(totalLines, max(startLine, nextBoundaryLine-1))
	for endLine > startLine && (strings.TrimSpace(lines[endLine-1]) == "" || lines[endLine-1] == "...") {
		endLine--
	}

	idxNode := &indexedYAMLNode{
		node:      node,
		keyNode:   keyNode,
		startLine: startLine,
		endLine:   endLine,
	}
	if node == nil {
		return idxNode
	}

	switch node.Kind {
	case yaml.MappingNode:
		pairCount := len(node.Content) / 2
		idxNode.byKey = make(map[string]*indexedYAMLNode, pairCount)
		idxNode.keyOrder = make([]string, 0, pairCount)
		for i := range pairCount {
			kNode := node.Content[2*i]
			vNode := node.Content[2*i+1]
			childBoundary := nextBoundaryLine
			if i+1 < pairCount && node.Content[2*(i+1)].Line > 0 {
				childBoundary = node.Content[2*(i+1)].Line
			}
			child := buildIndexedYAMLNode(vNode, kNode, startLine, childBoundary, lines)
			if _, exists := idxNode.byKey[kNode.Value]; !exists {
				idxNode.keyOrder = append(idxNode.keyOrder, kNode.Value)
			}
			idxNode.byKey[kNode.Value] = child
		}

	case yaml.SequenceNode:
		itemCount := len(node.Content)
		idxNode.seqItems = make([]*indexedYAMLNode, itemCount)
		itemStarts := make([]int, itemCount)
		prevMin := startLine
		for i, item := range node.Content {
			itemStarts[i] = nodeFirstLine(item, nil, prevMin, lines)
			prevMin = itemStarts[i]
		}
		for i, item := range node.Content {
			childBoundary := nextBoundaryLine
			if i+1 < itemCount {
				childBoundary = itemStarts[i+1]
			}
			child := buildIndexedYAMLNode(item, nil, itemStarts[i], childBoundary, lines)
			child.startLine = itemStarts[i]
			idxNode.seqItems[i] = child
		}
	}
	return idxNode
}

func nodeFirstLine(node *yaml.Node, keyNode *yaml.Node, minStartLine int, lines []string) int {
	if keyNode != nil && keyNode.Line > 0 {
		return clampInt(keyNode.Line, 1, max(1, len(lines)))
	}
	line := 1
	if node != nil && node.Line > 0 {
		line = node.Line
	}
	line = clampInt(line, 1, max(1, len(lines)))
	// Capture a standalone sequence '-' line immediately above a block mapping item.
	if keyNode == nil && line > 1 && line-1 > minStartLine && strings.TrimSpace(lines[line-2]) == "-" {
		line--
	}
	return line
}

func evalCompiledJSONPath(doc *indexedYAMLDocument, path compiledJSONPath) []yamlLineSelection {
	if doc == nil || doc.root == nil {
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
	if t == nil || t.node == nil || len(lines) == 0 {
		return 0, byteSpan{}
	}
	switch t.node.Kind {
	case yaml.ScalarNode:
		return trimmedContentSpanOnLine(lines, t.startLine)
	case yaml.MappingNode, yaml.SequenceNode:
		if t.keyNode != nil && t.keyNode.Line >= 1 && t.keyNode.Line <= len(lines) {
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
	if strings.HasPrefix(trimmedLeft, "- ") && len(trimmedLeft) > 2 {
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
		if len(next) == 0 {
			return nil
		}
		current = next
	}
	return current
}

func collectSelfAndDescendants(n *indexedYAMLNode) []*indexedYAMLNode {
	if n == nil {
		return nil
	}
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
	if cand == nil || cand.node == nil {
		return nil
	}
	switch step.kind {
	case stepKindKeys:
		if cand.node.Kind != yaml.MappingNode {
			return nil
		}
		var out []matchedYAMLNode
		for _, k := range step.keys {
			if child, ok := cand.byKey[k]; ok {
				out = append(out, matchedYAMLNode{target: child})
			}
		}
		return out

	case stepKindWildcard:
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
		}
		return nil

	case stepKindIndices:
		if cand.node.Kind != yaml.SequenceNode {
			return nil
		}
		n := len(cand.seqItems)
		var out []matchedYAMLNode
		for _, rawIdx := range step.indices {
			idx := rawIdx
			if idx < 0 {
				idx = n + idx
			}
			if idx >= 0 && idx < n {
				out = append(out, matchedYAMLNode{target: cand.seqItems[idx]})
			}
		}
		return out

	case stepKindSlice:
		if cand.node.Kind != yaml.SequenceNode {
			return nil
		}
		n := len(cand.seqItems)
		lo := 0
		if step.sliceFrom != nil {
			lo = *step.sliceFrom
			if lo < 0 {
				lo = n + lo
			}
		}
		hi := n
		if step.sliceTo != nil {
			hi = *step.sliceTo
			if hi < 0 {
				hi = n + hi
			}
		}
		lo = clampInt(lo, 0, n)
		hi = clampInt(hi, lo, n)
		out := make([]matchedYAMLNode, 0, hi-lo)
		for i := lo; i < hi; i++ {
			out = append(out, matchedYAMLNode{target: cand.seqItems[i]})
		}
		return out

	case stepKindFilter:
		var out []matchedYAMLNode
		switch cand.node.Kind {
		case yaml.SequenceNode:
			for _, item := range cand.seqItems {
				if ok, hl := step.filter.eval(item); ok {
					out = append(out, matchedYAMLNode{target: item, highlight: hl})
				}
			}
		case yaml.MappingNode:
			for _, k := range cand.keyOrder {
				child := cand.byKey[k]
				if ok, hl := step.filter.eval(child); ok {
					out = append(out, matchedYAMLNode{target: child, highlight: hl})
				}
			}
		}
		return out
	}
	return nil
}

// diagnoseUnmatchedJSONPath finds the deepest matched prefix across docs and returns its available child keys.
func diagnoseUnmatchedJSONPath(path compiledJSONPath, docs ...*indexedYAMLDocument) (matchedPrefix string, availableKeys []string) {
	var validRoots []*indexedYAMLNode
	for _, d := range docs {
		if d != nil && d.root != nil {
			validRoots = append(validRoots, d.root)
		}
	}
	if len(validRoots) == 0 {
		return "", nil
	}

	current := make([]matchedYAMLNode, len(validRoots))
	for i, r := range validRoots {
		current[i] = matchedYAMLNode{target: r}
	}
	var prefixBuilder strings.Builder
	prefixBuilder.WriteByte('$')
	matchedDepth := 0

	for _, step := range path.steps {
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
		if len(next) == 0 {
			break
		}
		prefixBuilder.WriteString(step.raw)
		current = next
		matchedDepth++
	}

	keySet := make(map[string]struct{})
	for _, m := range current {
		if m.target == nil {
			continue
		}
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
	keys := slices.Sorted(maps.Keys(keySet))
	if matchedDepth == 0 {
		return "", keys
	}
	return prefixBuilder.String(), keys
}
