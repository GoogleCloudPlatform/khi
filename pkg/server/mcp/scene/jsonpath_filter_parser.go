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
	rem := p.src[p.pos:]
	p.pos += len(rem) - len(strings.TrimLeft(rem, " \t"))
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
	if p.src[p.pos] != '@' {
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
	op, err := p.parseComparisonOp()
	if err != nil {
		return nil, err
	}
	if op == filterOpExists {
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
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		if bracketDepth == 0 && isFilterAtomBoundary(ch) {
			break
		}
		switch ch {
		case '\'', '"':
			p.skipQuotedSpan()
		case '[':
			bracketDepth++
			p.pos++
		case ']':
			bracketDepth = max(0, bracketDepth-1)
			p.pos++
		default:
			p.pos++
		}
	}
	return p.src[pathStart:p.pos]
}

func (p *filterParser) skipQuotedSpan() {
	quote := p.src[p.pos]
	escaped := false
	for p.pos++; p.pos < len(p.src); {
		ch := p.src[p.pos]
		p.pos++
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == quote {
			return
		}
	}
}

func (p *filterParser) parseComparisonOp() (filterOp, error) {
	rem := p.src[p.pos:]
	switch {
	case strings.HasPrefix(rem, "=="):
		p.pos += 2
		return filterOpEq, nil
	case strings.HasPrefix(rem, "!="):
		p.pos += 2
		return filterOpNe, nil
	case strings.HasPrefix(rem, "<="):
		p.pos += 2
		return filterOpLe, nil
	case strings.HasPrefix(rem, ">="):
		p.pos += 2
		return filterOpGe, nil
	case strings.HasPrefix(rem, "<"):
		p.pos++
		return filterOpLt, nil
	case strings.HasPrefix(rem, ">"):
		p.pos++
		return filterOpGt, nil
	case strings.HasPrefix(rem, "="):
		return filterOpExists, fmt.Errorf("use '==' instead of '=' in filter expression")
	default:
		return filterOpExists, nil
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
