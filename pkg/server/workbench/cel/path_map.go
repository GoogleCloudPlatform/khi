package cel

import (
	"reflect"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

// pathMap is the CEL value bound to the `path` variable of timeline queries.
// A timeline only has path levels for itself and its ancestors, so a query like `path["pod"] == "x"`
// would fail with "no such key" on namespace or kind timelines and abort the whole filter.
// pathMap resolves a missing string key to an empty string instead, while `has(path.x)` and
// `"x" in path` still report whether the level actually exists.
//
// pathMap intentionally does not implement traits.Mapper: cel-go qualifies a Mapper with Find
// and reports a missing key as an error, but it qualifies an Indexer with Get and uses
// traits.FieldTester for presence tests.
type pathMap struct {
	levels traits.Mapper
}

var _ ref.Val = pathMap{}
var _ traits.Indexer = pathMap{}
var _ traits.FieldTester = pathMap{}
var _ traits.Container = pathMap{}
var _ traits.Sizer = pathMap{}
var _ traits.Iterable = pathMap{}

// newPathMap wraps the path levels computed by TimelineData.ComputePath as a CEL value.
func newPathMap(levels map[string]string) pathMap {
	return pathMap{levels: types.NewStringStringMap(types.DefaultTypeAdapter, levels)}
}

// Get returns the timeline name of the given path level, or an empty string when the level is missing.
func (p pathMap) Get(key ref.Val) ref.Val {
	if _, ok := key.(types.String); !ok {
		return types.MaybeNoSuchOverloadErr(key)
	}
	if val, found := p.levels.Find(key); found {
		return val
	}
	return types.String("")
}

// IsSet reports whether the timeline has the given path level.
func (p pathMap) IsSet(key ref.Val) ref.Val {
	return p.levels.Contains(key)
}

// Contains reports whether the timeline has the given path level.
func (p pathMap) Contains(key ref.Val) ref.Val {
	return p.levels.Contains(key)
}

// Size returns the number of path levels.
func (p pathMap) Size() ref.Val {
	return p.levels.Size()
}

// Iterator returns an iterator over the path level keys.
func (p pathMap) Iterator() traits.Iterator {
	return p.levels.Iterator()
}

// ConvertToNative converts the path levels to the given Go type.
func (p pathMap) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return p.levels.ConvertToNative(typeDesc)
}

// ConvertToType converts the path levels to the given CEL type.
func (p pathMap) ConvertToType(typeVal ref.Type) ref.Val {
	return p.levels.ConvertToType(typeVal)
}

// Equal compares the path levels with another CEL value.
func (p pathMap) Equal(other ref.Val) ref.Val {
	return p.levels.Equal(other)
}

// Type returns the CEL map type.
func (p pathMap) Type() ref.Type {
	return types.MapType
}

// Value returns the underlying map[string]string.
func (p pathMap) Value() any {
	return p.levels.Value()
}
