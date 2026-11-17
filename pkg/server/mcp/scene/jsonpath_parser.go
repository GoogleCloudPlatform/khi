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
	"strconv"
	"strings"
)

const (
	stepKindKeys jsonPathStepKind = iota
	stepKindWildcard
	stepKindIndices
	stepKindSlice
	stepKindFilter
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

type jsonPathStepKind int

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

type filterParser struct {
	src string
	pos int
}

// Error formats the invalid JSONPath selector error message.
func (e *InvalidJSONPathError) Error() string {
	if e.PathIndex < 0 {
		return fmt.Sprintf("invalid JSONPath list: %s", e.Reason)
	}
	return fmt.Sprintf("invalid JSONPath %d (%q): %s", e.PathIndex, e.Path, e.Reason)
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
	if s[0] == '$' {
		pos++
	}
	var steps []jsonPathStep
	// Allow leading bare field name or wildcard when '$' and '.' are omitted (e.g. "spec.replicas").
	if pos < len(s) && s[pos] != '.' && s[pos] != '[' {
		step, nextPos, err := parseFieldOrWildcardStep(s, pos, ".", false)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
		pos = nextPos
	}
	for pos < len(s) {
		step, nextPos, err := parseNextJSONPathStep(s, pos)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
		pos = nextPos
	}
	if len(steps) == 0 && !allowEmptySteps {
		return nil, fmt.Errorf("JSONPath must select a field or element beneath root '$' (e.g. '$.spec' or '$.status')")
	}
	return steps, nil
}

func parseNextJSONPathStep(s string, pos int) (jsonPathStep, int, error) {
	switch {
	case strings.HasPrefix(s[pos:], ".."):
		return parseRecursiveStep(s, pos)
	case s[pos] == '.':
		return parseDotStep(s, pos)
	case s[pos] == '[':
		step, nextPos, err := parseBracketStep(s, pos, false)
		if err != nil {
			return jsonPathStep{}, 0, err
		}
		step.raw = s[pos:nextPos]
		return step, nextPos, nil
	default:
		return jsonPathStep{}, 0, fmt.Errorf("unexpected character %q at offset %d", s[pos], pos)
	}
}

func parseRecursiveStep(s string, stepStart int) (jsonPathStep, int, error) {
	pos := stepStart + 2
	if pos >= len(s) {
		return jsonPathStep{}, 0, fmt.Errorf("trailing '..' in JSONPath")
	}
	if s[pos] == '[' {
		step, nextPos, err := parseBracketStep(s, pos, true)
		if err != nil {
			return jsonPathStep{}, 0, err
		}
		step.raw = s[stepStart:nextPos]
		return step, nextPos, nil
	}
	return parseFieldOrWildcardStep(s, pos, "..", true)
}

func parseDotStep(s string, stepStart int) (jsonPathStep, int, error) {
	pos := stepStart + 1
	if pos >= len(s) {
		return jsonPathStep{}, 0, fmt.Errorf("trailing '.' in JSONPath")
	}
	if s[pos] == '[' {
		return jsonPathStep{}, 0, fmt.Errorf("unexpected '[' after '.'; use '[...]' without a preceding dot")
	}
	return parseFieldOrWildcardStep(s, pos, ".", false)
}

func parseFieldOrWildcardStep(s string, pos int, rawPrefix string, recursive bool) (jsonPathStep, int, error) {
	if s[pos] == '*' {
		return jsonPathStep{
			kind:      stepKindWildcard,
			recursive: recursive,
			raw:       rawPrefix + "*",
		}, pos + 1, nil
	}
	key, nextPos, err := parseBareIdentifier(s, pos)
	if err != nil {
		return jsonPathStep{}, 0, err
	}
	return jsonPathStep{
		kind:      stepKindKeys,
		recursive: recursive,
		keys:      []string{key},
		raw:       rawPrefix + s[pos:nextPos],
	}, nextPos, nil
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
		return jsonPathStep{kind: stepKindFilter, recursive: recursive, filter: expr}, nextPos, nil
	}
	if inner[0] == '\'' || inner[0] == '"' {
		keys, err := parseQuotedKeyUnion(inner)
		if err != nil {
			return jsonPathStep{}, 0, err
		}
		return jsonPathStep{kind: stepKindKeys, recursive: recursive, keys: keys}, nextPos, nil
	}
	if strings.Contains(inner, ":") {
		from, to, err := parseSliceBounds(inner)
		if err != nil {
			return jsonPathStep{}, 0, err
		}
		return jsonPathStep{kind: stepKindSlice, recursive: recursive, sliceFrom: from, sliceTo: to}, nextPos, nil
	}
	indices, err := parseIndexUnion(inner)
	if err != nil {
		return jsonPathStep{}, 0, err
	}
	return jsonPathStep{kind: stepKindIndices, recursive: recursive, indices: indices}, nextPos, nil
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
	for {
		val, nextPos, err := parseQuotedString(inner, pos)
		if err != nil {
			return nil, err
		}
		keys = append(keys, val)
		pos = skipHorizontalWS(inner, nextPos)
		if pos == len(inner) {
			return keys, nil
		}
		if inner[pos] != ',' {
			return nil, fmt.Errorf("expected ',' or ']' after quoted key in bracket selector")
		}
		pos = skipHorizontalWS(inner, pos+1)
		if pos == len(inner) {
			return nil, fmt.Errorf("trailing comma in bracket key selector")
		}
	}
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

func skipHorizontalWS(s string, pos int) int {
	for pos < len(s) && (s[pos] == ' ' || s[pos] == '\t') {
		pos++
	}
	return pos
}

func (p *filterParser) skipWS() {
	p.pos = skipHorizontalWS(p.src, p.pos)
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
	relRaw := p.scanAtomRelativePath()
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

func (p *filterParser) scanAtomRelativePath() string {
	pathStart := p.pos
	bracketDepth := 0
	var quote byte
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		if quote != 0 {
			if ch == '\\' {
				p.pos = min(p.pos+2, len(p.src))
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
		if bracketDepth == 0 && isFilterAtomBoundary(ch) {
			break
		}
		p.pos++
	}
	return p.src[pathStart:p.pos]
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

func isFilterAtomBoundary(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '=' || ch == '!' || ch == '<' || ch == '>' || ch == '&' || ch == '|' || ch == ')'
}
