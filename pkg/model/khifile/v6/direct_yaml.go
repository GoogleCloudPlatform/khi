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

package khifilev6

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
)

// FieldOp represents a precomputed formatting operation for a single field in a FieldPathSet.
type FieldOp struct {
	// PrefixYAML contains the YAML opening headers for newly entered nested parent maps.
	PrefixYAML string
	// Indent is the whitespace indentation preceding the key name.
	Indent string
	// KeyName is the formatted leaf key string (quoted if containing special characters like @type).
	KeyName string
	// IndentLevel is the numeric nesting level used to indent child items such as lists and nested structs.
	IndentLevel int
}

// FieldPathSchema caches the compiled FieldOp sequence for a specific FieldPathSet.
type FieldPathSchema struct {
	Ops []FieldOp
}

// DirectYAMLSerializer serializes InternedStruct directly to YAML text without constructing intermediate ASTs.
// It reuses an internal buffer across calls for performance and is not safe for concurrent use by multiple goroutines.
type DirectYAMLSerializer struct {
	buf         bytes.Buffer
	schemaCache sync.Map // map[uint32]*FieldPathSchema
}

// NewDirectYAMLSerializer creates a new DirectYAMLSerializer instance with an empty schema cache.
func NewDirectYAMLSerializer() *DirectYAMLSerializer {
	return &DirectYAMLSerializer{}
}

// SerializeStruct converts an InternedStruct into its YAML string representation.
func (s *DirectYAMLSerializer) SerializeStruct(structObj *pb.InternedStruct, pool *InternPool) (string, error) {
	s.buf.Reset()
	if structObj == nil || structObj.FieldPathSetId == nil {
		return "{}\n", nil
	}

	schema := s.getSchema(*structObj.FieldPathSetId, pool)
	if len(schema.Ops) == 0 {
		return "{}\n", nil
	}

	s.serializeStructWithIndent(&s.buf, structObj, pool, schema, 0)
	return s.buf.String(), nil
}

// StringAndFieldSetResolver provides lookup of strings and field path sets by ID.
type StringAndFieldSetResolver interface {
	ResolveStringFromID(id uint32) string
	ResolveFieldSetFromID(id uint32) []uint32
}

// SerializeFlatStruct converts an interned struct identified by structID into its YAML string representation
// directly from FlatStructStore without allocating intermediate Protobuf messages.
func (s *DirectYAMLSerializer) SerializeFlatStruct(structID uint32, pool *ReadonlyInternPool) (string, error) {
	s.buf.Reset()
	if pool == nil || pool.flatStructs == nil {
		return "", fmt.Errorf("readonly intern pool or flat structs store is nil")
	}

	fieldPathSetID, offset, count, ok := pool.flatStructs.GetValueSpan(structID)
	if !ok {
		return "", fmt.Errorf("struct ID %d not found", structID)
	}
	if count == 0 {
		return "{}\n", nil
	}

	schema := s.getSchema(fieldPathSetID, pool)
	if len(schema.Ops) == 0 {
		return "{}\n", nil
	}

	s.serializeFlatStructWithIndent(&s.buf, pool, schema, offset, count, 0)
	return s.buf.String(), nil
}

// getSchema retrieves the compiled schema from cache or compiles it on first access.
func (s *DirectYAMLSerializer) getSchema(fieldPathSetID uint32, resolver StringAndFieldSetResolver) *FieldPathSchema {
	if cached, ok := s.schemaCache.Load(fieldPathSetID); ok {
		return cached.(*FieldPathSchema)
	}

	ids := resolver.ResolveFieldSetFromID(fieldPathSetID)
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = resolver.ResolveStringFromID(id)
	}

	schema := compileFieldPathSchema(keys)
	s.schemaCache.Store(fieldPathSetID, schema)
	return schema
}

// compileFieldPathSchema parses the flattened keys and constructs the static prefix and indentation headers.
func compileFieldPathSchema(keys []string) *FieldPathSchema {
	ops := make([]FieldOp, len(keys))
	var prevParts []string

	for i, keyStr := range keys {
		parts := strings.Split(keyStr, fieldPathSeparator)

		// Calculate common prefix with the previous field path to avoid re-opening existing parent maps.
		common := 0
		for common < len(prevParts)-1 && common < len(parts)-1 && prevParts[common] == parts[common] {
			common++
		}

		// Emit opening lines for newly entered parent maps with safe key escaping.
		var prefixBuilder strings.Builder
		for lvl := common; lvl < len(parts)-1; lvl++ {
			prefixBuilder.WriteString(strings.Repeat("  ", lvl))
			prefixBuilder.WriteString(FormatKeyName(parts[lvl]))
			prefixBuilder.WriteString(":\n")
		}

		leafLvl := len(parts) - 1
		ops[i] = FieldOp{
			PrefixYAML:  prefixBuilder.String(),
			Indent:      strings.Repeat("  ", leafLvl),
			KeyName:     FormatKeyName(parts[leafLvl]),
			IndentLevel: leafLvl,
		}

		prevParts = parts
	}

	return &FieldPathSchema{Ops: ops}
}

// FormatKeyName formats a mapping key safely, applying quotes if the key starts with @ or contains control characters.
func FormatKeyName(key string) string {
	if isSafePlainScalar(key) {
		return key
	}
	return strconv.Quote(key)
}

// serializeStructWithIndent writes the formatted struct fields into the target buffer.
func (s *DirectYAMLSerializer) serializeStructWithIndent(
	targetBuf *bytes.Buffer,
	structObj *pb.InternedStruct,
	pool *InternPool,
	schema *FieldPathSchema,
	baseIndentLvl int,
) {
	baseIndent := strings.Repeat("  ", baseIndentLvl)

	for i, op := range schema.Ops {
		if i >= len(structObj.Values) {
			break
		}

		if op.PrefixYAML != "" {
			if baseIndentLvl > 0 {
				lines := strings.Split(strings.TrimSuffix(op.PrefixYAML, "\n"), "\n")
				for _, line := range lines {
					targetBuf.WriteString(baseIndent)
					targetBuf.WriteString(line)
					targetBuf.WriteString("\n")
				}
			} else {
				targetBuf.WriteString(op.PrefixYAML)
			}
		}

		targetBuf.WriteString(baseIndent)
		targetBuf.WriteString(op.Indent)
		targetBuf.WriteString(op.KeyName)
		targetBuf.WriteString(": ")

		s.emitValue(targetBuf, structObj.Values[i], pool, baseIndentLvl+op.IndentLevel)
		targetBuf.WriteString("\n")
	}
}

// emitValue writes a single InternedValue to the buffer with appropriate YAML formatting and quoting.
func (s *DirectYAMLSerializer) emitValue(targetBuf *bytes.Buffer, v *pb.InternedValue, pool *InternPool, indentLvl int) {
	if v == nil || v.Kind == nil {
		targetBuf.WriteString("null")
		return
	}

	switch kind := v.Kind.(type) {
	case *pb.InternedValue_NullValue:
		targetBuf.WriteString("null")
	case *pb.InternedValue_BoolValue:
		if kind.BoolValue {
			targetBuf.WriteString("true")
		} else {
			targetBuf.WriteString("false")
		}
	case *pb.InternedValue_StringValue:
		str := pool.resolveStringFromID(kind.StringValue)
		s.emitString(targetBuf, str)
	case *pb.InternedValue_Int64Value:
		targetBuf.WriteString(strconv.FormatInt(kind.Int64Value, 10))
	case *pb.InternedValue_DoubleValue:
		targetBuf.WriteString(strconv.FormatFloat(kind.DoubleValue, 'f', -1, 64))
	case *pb.InternedValue_TimestampValue:
		if kind.TimestampValue == nil {
			targetBuf.WriteString("null")
		} else {
			targetBuf.WriteString(kind.TimestampValue.AsTime().Format(time.RFC3339))
		}
	case *pb.InternedValue_ListValue:
		list := kind.ListValue.GetValues()
		if len(list) == 0 {
			targetBuf.WriteString("[]")
			return
		}
		for _, item := range list {
			s.emitListItem(targetBuf, item, pool, indentLvl+1)
		}
	case *pb.InternedValue_StructId:
		nested := pool.resolveStructFromID(kind.StructId)
		if nested == nil || nested.FieldPathSetId == nil {
			targetBuf.WriteString("{}")
			return
		}
		nestedSchema := s.getSchema(*nested.FieldPathSetId, pool)
		if len(nestedSchema.Ops) == 0 {
			targetBuf.WriteString("{}")
			return
		}
		targetBuf.WriteString("\n")
		s.serializeStructWithIndent(targetBuf, nested, pool, nestedSchema, indentLvl+1)
	case *pb.InternedValue_StructValue:
		nested := kind.StructValue
		if nested == nil || nested.FieldPathSetId == nil {
			targetBuf.WriteString("{}")
			return
		}
		nestedSchema := s.getSchema(*nested.FieldPathSetId, pool)
		if len(nestedSchema.Ops) == 0 {
			targetBuf.WriteString("{}")
			return
		}
		targetBuf.WriteString("\n")
		s.serializeStructWithIndent(targetBuf, nested, pool, nestedSchema, indentLvl+1)
	default:
		targetBuf.WriteString("null")
	}
}

// emitListItem writes a single sequence item with proper standard YAML indentation and `- ` prefix.
func (s *DirectYAMLSerializer) emitListItem(targetBuf *bytes.Buffer, item *pb.InternedValue, pool *InternPool, itemIndentLvl int) {
	targetBuf.WriteString("\n")
	if item == nil {
		targetBuf.WriteString(strings.Repeat("  ", itemIndentLvl-1))
		targetBuf.WriteString("- null")
		return
	}

	var nested *pb.InternedStruct
	if structID, ok := item.Kind.(*pb.InternedValue_StructId); ok {
		nested = pool.resolveStructFromID(structID.StructId)
	} else if structVal, ok := item.Kind.(*pb.InternedValue_StructValue); ok {
		nested = structVal.StructValue
	}

	if nested != nil && nested.FieldPathSetId != nil {
		nestedSchema := s.getSchema(*nested.FieldPathSetId, pool)
		if len(nestedSchema.Ops) > 0 {
			var subBuf bytes.Buffer
			s.serializeStructWithIndent(&subBuf, nested, pool, nestedSchema, 0)
			subYAML := strings.TrimRight(subBuf.String(), "\n")
			lines := strings.Split(subYAML, "\n")

			itemPrefix := strings.Repeat("  ", itemIndentLvl-1) + "- "
			subsequentPrefix := strings.Repeat("  ", itemIndentLvl)

			if len(lines) > 0 {
				targetBuf.WriteString(itemPrefix)
				targetBuf.WriteString(lines[0])
				for _, line := range lines[1:] {
					targetBuf.WriteString("\n")
					targetBuf.WriteString(subsequentPrefix)
					targetBuf.WriteString(line)
				}
			}
			return
		}
	}

	targetBuf.WriteString(strings.Repeat("  ", itemIndentLvl-1))
	targetBuf.WriteString("- ")
	s.emitValue(targetBuf, item, pool, itemIndentLvl)
}

// isSafePlainScalar checks if a string is guaranteed to be parsed as a string (never boolean, number, or control token).
func isSafePlainScalar(str string) bool {
	if len(str) == 0 {
		return false
	}

	// Must not have leading or trailing whitespace.
	if str[0] == ' ' || str[len(str)-1] == ' ' {
		return false
	}

	// Must start with an ASCII letter to guarantee it cannot be a number, timestamp, control token, or dot literal.
	first := str[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')) {
		return false
	}

	// Check against reserved YAML boolean and null keywords.
	if len(str) <= 5 {
		lower := strings.ToLower(str)
		switch lower {
		case "true", "false", "yes", "no", "on", "off", "null", "nan", "inf", "y", "n":
			return false
		}
	}

	// Must only contain safe identifier characters without any YAML syntax characters.
	for i := 0; i < len(str); i++ {
		c := str[i]
		if (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == ' ' || c == '-' || c == '_' || c == '.' || c == '/' {
			continue
		}
		return false
	}

	return true
}

// emitQuotedString writes a double-quoted YAML string preserving valid UTF-8 runes.
func emitQuotedString(targetBuf *bytes.Buffer, str string) {
	targetBuf.WriteByte('"')
	for _, r := range str {
		switch r {
		case '"':
			targetBuf.WriteString(`\"`)
		case '\\':
			targetBuf.WriteString(`\\`)
		case '\n':
			targetBuf.WriteString(`\n`)
		case '\r':
			targetBuf.WriteString(`\r`)
		case '\t':
			targetBuf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(targetBuf, `\x%02x`, r)
			} else {
				targetBuf.WriteRune(r)
			}
		}
	}
	targetBuf.WriteByte('"')
}

// emitString writes a string value to the buffer, using plain scalar for safe identifiers and double-quotes otherwise.
func (s *DirectYAMLSerializer) emitString(targetBuf *bytes.Buffer, str string) {
	if isSafePlainScalar(str) {
		targetBuf.WriteString(str)
	} else {
		emitQuotedString(targetBuf, str)
	}
}

// serializeFlatStructWithIndent writes formatted fields from FlatStructStore into target buffer.
func (s *DirectYAMLSerializer) serializeFlatStructWithIndent(
	targetBuf *bytes.Buffer,
	pool *ReadonlyInternPool,
	schema *FieldPathSchema,
	startOffset uint32,
	valCount uint32,
	baseIndentLvl int,
) {
	baseIndent := strings.Repeat("  ", baseIndentLvl)

	for i, op := range schema.Ops {
		if uint32(i) >= valCount {
			break
		}

		if op.PrefixYAML != "" {
			if baseIndentLvl > 0 {
				lines := strings.Split(strings.TrimSuffix(op.PrefixYAML, "\n"), "\n")
				for _, line := range lines {
					targetBuf.WriteString(baseIndent)
					targetBuf.WriteString(line)
					targetBuf.WriteString("\n")
				}
			} else {
				targetBuf.WriteString(op.PrefixYAML)
			}
		}

		targetBuf.WriteString(baseIndent)
		targetBuf.WriteString(op.Indent)
		targetBuf.WriteString(op.KeyName)
		targetBuf.WriteString(": ")

		slot := startOffset + uint32(i)
		s.emitFlatValue(targetBuf, slot, pool, baseIndentLvl+op.IndentLevel)
		targetBuf.WriteString("\n")
	}
}

// emitFlatValue writes a packed value from FlatStructStore with proper YAML formatting and quoting.
func (s *DirectYAMLSerializer) emitFlatValue(targetBuf *bytes.Buffer, slot uint32, pool *ReadonlyInternPool, indentLvl int) {
	kind, data, aux := pool.flatStructs.GetRawValueAt(slot)
	switch kind {
	case FlatValueKindNull:
		targetBuf.WriteString("null")
	case FlatValueKindBool:
		if data != 0 {
			targetBuf.WriteString("true")
		} else {
			targetBuf.WriteString("false")
		}
	case FlatValueKindInt64:
		targetBuf.WriteString(strconv.FormatInt(int64(data), 10))
	case FlatValueKindDouble:
		targetBuf.WriteString(strconv.FormatFloat(math.Float64frombits(data), 'f', -1, 64))
	case FlatValueKindStringID:
		str := pool.ResolveStringFromID(uint32(data))
		s.emitString(targetBuf, str)
	case FlatValueKindStructID:
		nestedID := uint32(data)
		fieldPathSetID, offset, count, ok := pool.flatStructs.GetValueSpan(nestedID)
		if !ok || count == 0 {
			targetBuf.WriteString("{}")
			return
		}
		nestedSchema := s.getSchema(fieldPathSetID, pool)
		if len(nestedSchema.Ops) == 0 {
			targetBuf.WriteString("{}")
			return
		}
		targetBuf.WriteString("\n")
		s.serializeFlatStructWithIndent(targetBuf, pool, nestedSchema, offset, count, indentLvl+1)
	case FlatValueKindTimestamp:
		t := time.Unix(int64(data), int64(aux)).UTC()
		targetBuf.WriteString(t.Format(time.RFC3339))
	case FlatValueKindList:
		listOffset := uint32(data)
		listCount := aux
		if listCount == 0 {
			targetBuf.WriteString("[]")
			return
		}
		for j := uint32(0); j < listCount; j++ {
			s.emitFlatListItem(targetBuf, listOffset+j, pool, indentLvl+1)
		}
	case FlatValueKindStructValue:
		fieldSetID := aux
		nestedCount := uint32(data >> 32)
		nestedOffset := uint32(data & 0xFFFFFFFF)
		if nestedCount == 0 {
			targetBuf.WriteString("{}")
			return
		}
		nestedSchema := s.getSchema(fieldSetID, pool)
		if len(nestedSchema.Ops) == 0 {
			targetBuf.WriteString("{}")
			return
		}
		targetBuf.WriteString("\n")
		s.serializeFlatStructWithIndent(targetBuf, pool, nestedSchema, nestedOffset, nestedCount, indentLvl+1)
	default:
		targetBuf.WriteString("null")
	}
}

// emitFlatListItem writes a single packed sequence item with proper YAML `- ` prefix.
func (s *DirectYAMLSerializer) emitFlatListItem(targetBuf *bytes.Buffer, slot uint32, pool *ReadonlyInternPool, itemIndentLvl int) {
	targetBuf.WriteString("\n")

	kind, data, aux := pool.flatStructs.GetRawValueAt(slot)

	var nestedFieldPathSetID uint32
	var nestedOffset uint32
	var nestedCount uint32
	var isStruct bool

	switch kind {
	case FlatValueKindStructID:
		nestedID := uint32(data)
		fsID, off, cnt, ok := pool.flatStructs.GetValueSpan(nestedID)
		if ok && cnt > 0 {
			nestedFieldPathSetID = fsID
			nestedOffset = off
			nestedCount = cnt
			isStruct = true
		}
	case FlatValueKindStructValue:
		fsID := aux
		cnt := uint32(data >> 32)
		off := uint32(data & 0xFFFFFFFF)
		if cnt > 0 {
			nestedFieldPathSetID = fsID
			nestedOffset = off
			nestedCount = cnt
			isStruct = true
		}
	}

	if isStruct {
		nestedSchema := s.getSchema(nestedFieldPathSetID, pool)
		if len(nestedSchema.Ops) > 0 {
			var subBuf bytes.Buffer
			s.serializeFlatStructWithIndent(&subBuf, pool, nestedSchema, nestedOffset, nestedCount, 0)
			subYAML := strings.TrimRight(subBuf.String(), "\n")
			lines := strings.Split(subYAML, "\n")

			itemPrefix := strings.Repeat("  ", itemIndentLvl-1) + "- "
			subsequentPrefix := strings.Repeat("  ", itemIndentLvl)

			if len(lines) > 0 {
				targetBuf.WriteString(itemPrefix)
				targetBuf.WriteString(lines[0])
				for _, line := range lines[1:] {
					targetBuf.WriteString("\n")
					targetBuf.WriteString(subsequentPrefix)
					targetBuf.WriteString(line)
				}
			}
			return
		}
	}

	targetBuf.WriteString(strings.Repeat("  ", itemIndentLvl-1))
	targetBuf.WriteString("- ")

	switch kind {
	case FlatValueKindNull:
		targetBuf.WriteString("null")
	case FlatValueKindBool:
		if data != 0 {
			targetBuf.WriteString("true")
		} else {
			targetBuf.WriteString("false")
		}
	case FlatValueKindInt64:
		targetBuf.WriteString(strconv.FormatInt(int64(data), 10))
	case FlatValueKindDouble:
		targetBuf.WriteString(strconv.FormatFloat(math.Float64frombits(data), 'f', -1, 64))
	case FlatValueKindStringID:
		str := pool.ResolveStringFromID(uint32(data))
		s.emitString(targetBuf, str)
	case FlatValueKindStructID, FlatValueKindStructValue:
		targetBuf.WriteString("{}")
	case FlatValueKindTimestamp:
		t := time.Unix(int64(data), int64(aux)).UTC()
		targetBuf.WriteString(t.Format(time.RFC3339))
	case FlatValueKindList:
		listOffset := uint32(data)
		listCount := aux
		if listCount == 0 {
			targetBuf.WriteString("[]")
			return
		}
		for j := uint32(0); j < listCount; j++ {
			s.emitFlatListItem(targetBuf, listOffset+j, pool, itemIndentLvl+1)
		}
	default:
		targetBuf.WriteString("null")
	}
}
