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

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

// ErrInvalidJSON indicates that invalid JSON syntax was encountered during scanning.
var ErrInvalidJSON = errors.New("invalid json format")

// LazyJSONNode represents structured data as an immutable JSON byte buffer and an offset index.
// Child nodes reuse the underlying byte buffer without allocating sub-slices, adjusting only the offset index.
type LazyJSONNode struct {
	data  []byte
	index int
}

var _ Node = (*LazyJSONNode)(nil)

// NewLazyJSONNode serializes a given Node to JSON and wraps it in a LazyJSONNode.
func NewLazyJSONNode(node Node) (Node, error) {
	if lazyNode, ok := node.(*LazyJSONNode); ok {
		return lazyNode, nil
	}
	serializer := JSONNodeSerializer{}
	data, err := serializer.Serialize(node)
	if err != nil {
		return nil, err
	}
	return NewLazyJSONNodeFromBytes(data), nil
}

// NewLazyJSONNodeFromBytes creates a LazyJSONNode from a JSON byte slice.
func NewLazyJSONNodeFromBytes(data []byte) Node {
	return &LazyJSONNode{
		data:  data,
		index: 0,
	}
}

// Type returns the NodeType of this node.
func (n *LazyJSONNode) Type() NodeType {
	idx := skipWhitespace(n.data, n.index)
	if idx >= len(n.data) {
		return InvalidNodeType
	}
	switch n.data[idx] {
	case '{':
		return MapNodeType
	case '[':
		return SequenceNodeType
	case '"', 't', 'f', 'n', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return ScalarNodeType
	default:
		return InvalidNodeType
	}
}

// NodeScalarValue returns the scalar value of this node.
func (n *LazyJSONNode) NodeScalarValue() (any, error) {
	if n.Type() != ScalarNodeType {
		return nil, ErrNonScalarNode
	}
	val, _, err := parseJSONScalar(n.data, n.index)
	return val, err
}

// Children returns an iterator function over the children of this node.
func (n *LazyJSONNode) Children() NodeChildrenIterator {
	nodeType := n.Type()
	switch nodeType {
	case MapNodeType:
		return func(yield func(key NodeChildrenKey, value Node) bool) {
			idx := skipWhitespace(n.data, n.index)
			if idx >= len(n.data) || n.data[idx] != '{' {
				return
			}
			idx++ // Skip '{'
			childIndex := 0

			for {
				idx = skipWhitespace(n.data, idx)
				if idx >= len(n.data) {
					return
				}
				if n.data[idx] == '}' {
					return
				}

				if n.data[idx] != '"' {
					return
				}

				keyStr, nextIdx, err := parseJSONString(n.data, idx)
				if err != nil {
					return
				}
				idx = skipWhitespace(n.data, nextIdx)
				if idx >= len(n.data) || n.data[idx] != ':' {
					return
				}
				idx++ // Skip ':'
				valStartIdx := skipWhitespace(n.data, idx)
				if valStartIdx >= len(n.data) {
					return
				}

				childNode := &LazyJSONNode{
					data:  n.data,
					index: valStartIdx,
				}
				if !yield(NodeChildrenKey{Index: childIndex, Key: keyStr}, childNode) {
					return
				}
				childIndex++

				valEndIdx, err := skipJSONValue(n.data, valStartIdx)
				if err != nil {
					return
				}
				idx = skipWhitespace(n.data, valEndIdx)
				if idx < len(n.data) && n.data[idx] == ',' {
					idx++
				}
			}
		}
	case SequenceNodeType:
		return func(yield func(key NodeChildrenKey, value Node) bool) {
			idx := skipWhitespace(n.data, n.index)
			if idx >= len(n.data) || n.data[idx] != '[' {
				return
			}
			idx++ // Skip '['
			childIndex := 0

			for {
				idx = skipWhitespace(n.data, idx)
				if idx >= len(n.data) {
					return
				}
				if n.data[idx] == ']' {
					return
				}

				elemStartIdx := idx
				childNode := &LazyJSONNode{
					data:  n.data,
					index: elemStartIdx,
				}
				if !yield(NodeChildrenKey{Index: childIndex, Key: ""}, childNode) {
					return
				}
				childIndex++

				elemEndIdx, err := skipJSONValue(n.data, elemStartIdx)
				if err != nil {
					return
				}
				idx = skipWhitespace(n.data, elemEndIdx)
				if idx < len(n.data) && n.data[idx] == ',' {
					idx++
				}
			}
		}
	default:
		return func(func(key NodeChildrenKey, value Node) bool) {}
	}
}

// Len returns the count of items in a map or sequence, or 0 for scalars.
func (n *LazyJSONNode) Len() int {
	nodeType := n.Type()
	switch nodeType {
	case MapNodeType:
		idx := skipWhitespace(n.data, n.index)
		if idx >= len(n.data) || n.data[idx] != '{' {
			return 0
		}
		idx++
		count := 0
		for {
			idx = skipWhitespace(n.data, idx)
			if idx >= len(n.data) || n.data[idx] == '}' {
				break
			}
			if n.data[idx] != '"' {
				break
			}
			_, nextIdx, err := parseJSONString(n.data, idx)
			if err != nil {
				break
			}
			idx = skipWhitespace(n.data, nextIdx)
			if idx >= len(n.data) || n.data[idx] != ':' {
				break
			}
			idx++
			valStartIdx := skipWhitespace(n.data, idx)
			valEndIdx, err := skipJSONValue(n.data, valStartIdx)
			if err != nil {
				break
			}
			count++
			idx = skipWhitespace(n.data, valEndIdx)
			if idx < len(n.data) && n.data[idx] == ',' {
				idx++
			}
		}
		return count
	case SequenceNodeType:
		idx := skipWhitespace(n.data, n.index)
		if idx >= len(n.data) || n.data[idx] != '[' {
			return 0
		}
		idx++
		count := 0
		for {
			idx = skipWhitespace(n.data, idx)
			if idx >= len(n.data) || n.data[idx] == ']' {
				break
			}
			elemEndIdx, err := skipJSONValue(n.data, idx)
			if err != nil {
				break
			}
			count++
			idx = skipWhitespace(n.data, elemEndIdx)
			if idx < len(n.data) && n.data[idx] == ',' {
				idx++
			}
		}
		return count
	default:
		return 0
	}
}

func skipWhitespace(data []byte, index int) int {
	for index < len(data) {
		c := data[index]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			index++
		} else {
			break
		}
	}
	return index
}

func parseJSONString(data []byte, index int) (string, int, error) {
	idx := skipWhitespace(data, index)
	if idx >= len(data) || data[idx] != '"' {
		return "", idx, ErrInvalidJSON
	}
	idx++ // Skip opening '"'

	start := idx
	hasEscapes := false
	escaped := false

stringScanLoop:
	for idx < len(data) {
		c := data[idx]
		switch {
		case escaped:
			escaped = false
		case c == '\\':
			hasEscapes = true
			escaped = true
		case c == '"':
			if !hasEscapes {
				if start == idx {
					return "", idx + 1, nil
				}
				return unsafe.String(&data[start], idx-start), idx + 1, nil
			}
			break stringScanLoop
		}
		idx++
	}

	if idx >= len(data) || data[idx] != '"' {
		return "", idx, ErrInvalidJSON
	}

	// Unescape string with escape sequences
	var sb strings.Builder
	sb.Grow(idx - start)
	idx = start
	for idx < len(data) {
		c := data[idx]
		if c == '"' {
			idx++
			return sb.String(), idx, nil
		}
		if c == '\\' {
			idx++
			if idx >= len(data) {
				return "", idx, ErrInvalidJSON
			}
			escapeChar := data[idx]
			switch escapeChar {
			case '"', '\\', '/':
				sb.WriteByte(escapeChar)
				idx++
			case 'b':
				sb.WriteByte('\b')
				idx++
			case 'f':
				sb.WriteByte('\f')
				idx++
			case 'n':
				sb.WriteByte('\n')
				idx++
			case 'r':
				sb.WriteByte('\r')
				idx++
			case 't':
				sb.WriteByte('\t')
				idx++
			case 'u':
				idx++
				if idx+4 > len(data) {
					return "", idx, ErrInvalidJSON
				}
				r, err := parseHex4(data[idx : idx+4])
				if err != nil {
					return "", idx, err
				}
				idx += 4

				// Check for UTF-16 surrogate pairs
				if utf16.IsSurrogate(r) {
					if idx+6 <= len(data) && data[idx] == '\\' && data[idx+1] == 'u' {
						r2, err := parseHex4(data[idx+2 : idx+6])
						if err == nil && utf16.IsSurrogate(r2) {
							combined := utf16.DecodeRune(r, r2)
							sb.WriteRune(combined)
							idx += 6
							continue
						}
					}
					sb.WriteRune(utf8.RuneError)
				} else {
					sb.WriteRune(r)
				}
			default:
				sb.WriteByte(escapeChar)
				idx++
			}
		} else {
			sb.WriteByte(c)
			idx++
		}
	}

	return "", idx, ErrInvalidJSON
}

func parseHex4(b []byte) (rune, error) {
	var val rune
	for _, c := range b {
		val <<= 4
		switch {
		case c >= '0' && c <= '9':
			val |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			val |= rune(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			val |= rune(c - 'A' + 10)
		default:
			return 0, ErrInvalidJSON
		}
	}
	return val, nil
}

func parseJSONScalar(data []byte, index int) (any, int, error) {
	idx := skipWhitespace(data, index)
	if idx >= len(data) {
		return nil, idx, ErrInvalidJSON
	}

	switch data[idx] {
	case '"':
		return parseJSONString(data, idx)
	case 't':
		if idx+4 <= len(data) && data[idx] == 't' && data[idx+1] == 'r' && data[idx+2] == 'u' && data[idx+3] == 'e' {
			return true, idx + 4, nil
		}
		return nil, idx, ErrInvalidJSON
	case 'f':
		if idx+5 <= len(data) && data[idx] == 'f' && data[idx+1] == 'a' && data[idx+2] == 'l' && data[idx+3] == 's' && data[idx+4] == 'e' {
			return false, idx + 5, nil
		}
		return nil, idx, ErrInvalidJSON
	case 'n':
		if idx+4 <= len(data) && data[idx] == 'n' && data[idx+1] == 'u' && data[idx+2] == 'l' && data[idx+3] == 'l' {
			return nil, idx + 4, nil
		}
		return nil, idx, ErrInvalidJSON
	default:
		// Number
		start := idx
		hasFloatChar := false
	numberLoop:
		for idx < len(data) {
			c := data[idx]
			switch {
			case c == '.' || c == 'e' || c == 'E':
				hasFloatChar = true
				idx++
			case (c >= '0' && c <= '9') || c == '-' || c == '+':
				idx++
			default:
				break numberLoop
			}
		}
		if start == idx {
			return nil, idx, ErrInvalidJSON
		}
		numStr := unsafe.String(&data[start], idx-start)
		if hasFloatChar {
			floatVal, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return nil, idx, err
			}
			return floatVal, idx, nil
		}
		intVal, err := strconv.Atoi(numStr)
		if err == nil {
			return intVal, idx, nil
		}
		// Fallback to int64 or float64 if integer overflows standard int
		int64Val, err := strconv.ParseInt(numStr, 10, 64)
		if err == nil {
			return int64Val, idx, nil
		}
		floatVal, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return nil, idx, fmt.Errorf("failed to parse number %q: %w", numStr, err)
		}
		return floatVal, idx, nil
	}
}

func skipJSONValue(data []byte, index int) (int, error) {
	idx := skipWhitespace(data, index)
	if idx >= len(data) {
		return idx, ErrInvalidJSON
	}

	switch data[idx] {
	case '{':
		idx++
		depth := 1
		inString := false
		escaped := false
		for idx < len(data) {
			c := data[idx]
			if inString {
				switch {
				case escaped:
					escaped = false
				case c == '\\':
					escaped = true
				case c == '"':
					inString = false
				}
			} else {
				switch c {
				case '"':
					inString = true
				case '{':
					depth++
				case '}':
					depth--
					if depth == 0 {
						return idx + 1, nil
					}
				}
			}
			idx++
		}
		return idx, ErrInvalidJSON
	case '[':
		idx++
		depth := 1
		inString := false
		escaped := false
		for idx < len(data) {
			c := data[idx]
			if inString {
				switch {
				case escaped:
					escaped = false
				case c == '\\':
					escaped = true
				case c == '"':
					inString = false
				}
			} else {
				switch c {
				case '"':
					inString = true
				case '[':
					depth++
				case ']':
					depth--
					if depth == 0 {
						return idx + 1, nil
					}
				}
			}
			idx++
		}
		return idx, ErrInvalidJSON
	case '"':
		_, nextIdx, err := parseJSONString(data, idx)
		return nextIdx, err
	default:
		// Scalar (bool, null, number)
		for idx < len(data) {
			c := data[idx]
			if c == ',' || c == '}' || c == ']' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				return idx, nil
			}
			idx++
		}
		return idx, nil
	}
}
