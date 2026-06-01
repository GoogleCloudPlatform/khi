package structured

// fieldFilterNode wraps an existing Node and filters its children by ignoring specific keys.
type fieldFilterNode struct {
	original Node
	ignored  map[string]struct{}
}

var _ Node = (*fieldFilterNode)(nil)

// NewFieldFilterNode returns a filtered Node that wraps the original Node.
// It filters out any map children whose keys are present in the ignored keys list.
func NewFieldFilterNode(original Node, keys []string) Node {
	ignored := make(map[string]struct{})
	for _, key := range keys {
		ignored[key] = struct{}{}
	}
	return &fieldFilterNode{
		original: original,
		ignored:  ignored,
	}
}

// Type returns the NodeType of the wrapped original node.
func (n *fieldFilterNode) Type() NodeType {
	return n.original.Type()
}

// NodeScalarValue returns the scalar value of the wrapped original node, or an error if it is not a scalar node.
func (n *fieldFilterNode) NodeScalarValue() (any, error) {
	return n.original.NodeScalarValue()
}

// Children returns a filtered iterator of the original node's children.
// Children whose keys are present in the ignored list are skipped.
func (n *fieldFilterNode) Children() NodeChildrenIterator {
	return func(callback func(key NodeChildrenKey, value Node) bool) {
		index := 0
		n.original.Children()(func(key NodeChildrenKey, value Node) bool {
			if _, ok := n.ignored[key.Key]; ok {
				return true
			}
			next := NodeChildrenKey{
				Index: index,
				Key:   key.Key,
			}
			index += 1
			return callback(next, value)
		})
	}
}

// Len returns the number of children after filtering out ignored keys.
func (n *fieldFilterNode) Len() int {
	count := 0
	n.Children()(func(key NodeChildrenKey, value Node) bool {
		count += 1
		return true
	})
	return count
}
