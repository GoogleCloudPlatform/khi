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
	"cmp"
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	filterOpExists filterOp = iota
	filterOpEq
	filterOpNe
	filterOpLt
	filterOpLe
	filterOpGt
	filterOpGe
)

const (
	literalString literalKind = iota
	literalNumber
	literalBool
	literalNull
)

type filterOp int

type literalKind int

type filterLiteral struct {
	kind    literalKind
	strVal  string
	numVal  float64
	boolVal bool
}

type filterExpr interface {
	eval(candidate *indexedYAMLNode) (matched bool, highlight *indexedYAMLNode)
}

type filterOrExpr struct {
	left  filterExpr
	right filterExpr
}

type filterAndExpr struct {
	left  filterExpr
	right filterExpr
}

type filterAtomExpr struct {
	relSteps []jsonPathStep
	op       filterOp
	rhs      filterLiteral
}

func (e filterOrExpr) eval(candidate *indexedYAMLNode) (bool, *indexedYAMLNode) {
	if ok, hl := e.left.eval(candidate); ok {
		return true, hl
	}
	return e.right.eval(candidate)
}

func (e filterAndExpr) eval(candidate *indexedYAMLNode) (bool, *indexedYAMLNode) {
	okL, hlL := e.left.eval(candidate)
	if !okL {
		return false, nil
	}
	okR, _ := e.right.eval(candidate)
	if !okR {
		return false, nil
	}
	return true, hlL
}

func (e filterAtomExpr) eval(candidate *indexedYAMLNode) (bool, *indexedYAMLNode) {
	targets := evalSteps([]*indexedYAMLNode{candidate}, e.relSteps)
	if e.op == filterOpExists {
		for _, m := range targets {
			if m.target.node.Tag != "!!null" {
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
	if node.Kind != yaml.ScalarNode {
		return op == filterOpNe && lit.kind != literalNull
	}
	isNull := node.Tag == "!!null"
	if lit.kind == literalNull {
		return compareNullLiteral(isNull, op)
	}
	if isNull {
		return op == filterOpNe
	}
	switch lit.kind {
	case literalBool:
		return compareBoolLiteral(node.Value, op, lit.boolVal)
	case literalNumber:
		v, err := strconv.ParseFloat(node.Value, 64)
		return err == nil && compareOrdered(v, op, lit.numVal)
	default:
		return compareOrdered(node.Value, op, lit.strVal)
	}
}

func compareNullLiteral(isNull bool, op filterOp) bool {
	switch op {
	case filterOpEq:
		return isNull
	case filterOpNe:
		return !isNull
	default:
		return false
	}
}

func compareBoolLiteral(raw string, op filterOp, want bool) bool {
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	switch op {
	case filterOpEq:
		return b == want
	case filterOpNe:
		return b != want
	default:
		return false
	}
}

func compareOrdered[T cmp.Ordered](lhs T, op filterOp, rhs T) bool {
	switch op {
	case filterOpEq:
		return lhs == rhs
	case filterOpNe:
		return lhs != rhs
	case filterOpLt:
		return lhs < rhs
	case filterOpLe:
		return lhs <= rhs
	case filterOpGt:
		return lhs > rhs
	default:
		return lhs >= rhs
	}
}
