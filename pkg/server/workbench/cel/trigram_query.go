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

package cel

import (
	"fmt"
	"regexp/syntax"
	"strings"
)

// TrigramQuery represents a node in the Boolean AST of trigram search requirements.
type TrigramQuery interface {
	String() string
	Simplify() TrigramQuery
}

// AllQuery represents an unconstrained query matching all candidates (Universe).
type AllQuery struct{}

var _ TrigramQuery = (*AllQuery)(nil)

// String returns a human-readable representation of AllQuery.
func (q *AllQuery) String() string {
	return "ALL"
}

// Simplify returns the AllQuery unchanged.
func (q *AllQuery) Simplify() TrigramQuery {
	return q
}

// NoneQuery represents an impossible query matching no candidates (Empty).
type NoneQuery struct{}

var _ TrigramQuery = (*NoneQuery)(nil)

// String returns a human-readable representation of NoneQuery.
func (q *NoneQuery) String() string {
	return "NONE"
}

// Simplify returns the NoneQuery unchanged.
func (q *NoneQuery) Simplify() TrigramQuery {
	return q
}

// TermQuery represents a single 3-gram term constraint.
type TermQuery struct {
	Term string
}

var _ TrigramQuery = (*TermQuery)(nil)

// String returns a human-readable representation of TermQuery.
func (q *TermQuery) String() string {
	return fmt.Sprintf("TERM(%q)", q.Term)
}

// Simplify returns the TermQuery unchanged.
func (q *TermQuery) Simplify() TrigramQuery {
	return q
}

// AndQuery represents a conjunction (AND) of sub-queries.
type AndQuery struct {
	Children []TrigramQuery
}

var _ TrigramQuery = (*AndQuery)(nil)

// String returns a human-readable representation of AndQuery.
func (q *AndQuery) String() string {
	parts := make([]string, len(q.Children))
	for i, c := range q.Children {
		parts[i] = c.String()
	}
	return fmt.Sprintf("AND(%s)", strings.Join(parts, ", "))
}

// Simplify reduces the AndQuery by eliminating neutral elements and flattening nested ANDs.
func (q *AndQuery) Simplify() TrigramQuery {
	var simplified []TrigramQuery
	seenTerms := make(map[string]struct{})

	for _, child := range q.Children {
		sChild := child.Simplify()
		switch c := sChild.(type) {
		case *NoneQuery:
			return &NoneQuery{}
		case *AllQuery:
			continue
		case *AndQuery:
			for _, subChild := range c.Children {
				if t, ok := subChild.(*TermQuery); ok {
					if _, seen := seenTerms[t.Term]; seen {
						continue
					}
					seenTerms[t.Term] = struct{}{}
				}
				simplified = append(simplified, subChild)
			}
		case *TermQuery:
			if _, seen := seenTerms[c.Term]; seen {
				continue
			}
			seenTerms[c.Term] = struct{}{}
			simplified = append(simplified, c)
		default:
			simplified = append(simplified, sChild)
		}
	}

	if len(simplified) == 0 {
		return &AllQuery{}
	}
	if len(simplified) == 1 {
		return simplified[0]
	}
	return &AndQuery{Children: simplified}
}

// OrQuery represents a disjunction (OR) of sub-queries.
type OrQuery struct {
	Children []TrigramQuery
}

var _ TrigramQuery = (*OrQuery)(nil)

// String returns a human-readable representation of OrQuery.
func (q *OrQuery) String() string {
	parts := make([]string, len(q.Children))
	for i, c := range q.Children {
		parts[i] = c.String()
	}
	return fmt.Sprintf("OR(%s)", strings.Join(parts, ", "))
}

// Simplify reduces the OrQuery by eliminating neutral elements and flattening nested ORs.
func (q *OrQuery) Simplify() TrigramQuery {
	var simplified []TrigramQuery
	seenTerms := make(map[string]struct{})

	for _, child := range q.Children {
		sChild := child.Simplify()
		switch c := sChild.(type) {
		case *AllQuery:
			return &AllQuery{}
		case *NoneQuery:
			continue
		case *OrQuery:
			for _, subChild := range c.Children {
				if t, ok := subChild.(*TermQuery); ok {
					if _, seen := seenTerms[t.Term]; seen {
						continue
					}
					seenTerms[t.Term] = struct{}{}
				}
				simplified = append(simplified, subChild)
			}
		case *TermQuery:
			if _, seen := seenTerms[c.Term]; seen {
				continue
			}
			seenTerms[c.Term] = struct{}{}
			simplified = append(simplified, c)
		default:
			simplified = append(simplified, sChild)
		}
	}

	if len(simplified) == 0 {
		return &NoneQuery{}
	}
	if len(simplified) == 1 {
		return simplified[0]
	}
	return &OrQuery{Children: simplified}
}

// RegexToTrigramQuery translates a parsed syntax.Regexp into a TrigramQuery Boolean AST.
func RegexToTrigramQuery(s *syntax.Regexp) TrigramQuery {
	if s == nil {
		return &AllQuery{}
	}

	switch s.Op {
	case syntax.OpLiteral:
		str := string(s.Rune)
		lowerStr := strings.ToLower(str)
		runes := []rune(lowerStr)
		if len(runes) < 3 {
			return &AllQuery{}
		}
		if len(runes) == 3 {
			return &TermQuery{Term: string(runes)}
		}
		children := make([]TrigramQuery, 0, len(runes)-2)
		for i := 0; i <= len(runes)-3; i++ {
			children = append(children, &TermQuery{Term: string(runes[i : i+3])})
		}
		return &AndQuery{Children: children}

	case syntax.OpConcat:
		children := make([]TrigramQuery, 0, len(s.Sub))
		for _, sub := range s.Sub {
			child := RegexToTrigramQuery(sub)
			children = append(children, child)
		}
		return &AndQuery{Children: children}

	case syntax.OpAlternate:
		children := make([]TrigramQuery, 0, len(s.Sub))
		for _, sub := range s.Sub {
			child := RegexToTrigramQuery(sub)
			children = append(children, child)
		}
		return &OrQuery{Children: children}

	case syntax.OpCapture, syntax.OpPlus:
		if len(s.Sub) > 0 {
			return RegexToTrigramQuery(s.Sub[0])
		}
		return &AllQuery{}

	case syntax.OpRepeat:
		if s.Min >= 1 && len(s.Sub) > 0 {
			return RegexToTrigramQuery(s.Sub[0])
		}
		return &AllQuery{}

	default:
		// OpStar, OpQuest, OpCharClass, OpAnyChar, OpBeginLine, etc.
		return &AllQuery{}
	}
}
