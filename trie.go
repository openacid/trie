// Package trie is a pointer based Trie implementation.
//
// Since 0.1.0
package trie

import (
	"github.com/openacid/errors"
	"github.com/openacid/low/tree"
	"github.com/openacid/low/typehelper"
)

// Node is a Trie node.
//
// Since 0.1.0
type Node struct {

	// Children nodes the Branches pointing to.
	//
	// Since 0.1.0
	Children map[int]*Node

	// Branches is a array of outgoing branch labels.
	//
	// Since 0.1.0
	Branches []int

	// Step is the number or "squash"-ed nodes along a branch.
	//
	// Since 0.1.0
	Step uint16

	// Value is user data bound to a leaf node.
	//
	// Since 0.1.0
	Value interface{}

	// squash indicates whether to remove nodes with only one child.
	squash bool

	// InnerNodeCnt records the number of outgoing branches to inner nodes.
	InnerNodeCnt int
}

const leafBranch = -1

// NewTrie creates a trie from a serial of ascendingly ordered keys and corresponding values.
//
// `values` must be a slice, or it panic.
// if `squash` is `true`, indicate this trie to squash preceding branches every time after Append a new
// key.
//
// Since 0.1.0
func NewTrie(keys [][]byte, values interface{}, squash bool) (root *Node, err error) {

	root = &Node{Children: make(map[int]*Node), Step: 1, squash: squash, InnerNodeCnt: 1}

	if keys == nil {
		return
	}

	valSlice := typehelper.ToSlice(values)

	if len(keys) != len(valSlice) {
		err = ErrKVLenNotMatch
		return
	}

	for i := 0; i < len(keys); i++ {
		key := keys[i]
		_, err = root.Append(key, valSlice[i])
		if err != nil {
			err = errors.Wrapf(err, "trie failed to add kvs")
			return
		}
	}

	if squash {
		root.Squash()
	}

	return
}

// String outputs multiline trie structure.
//
// Since 0.1.0
func (r *Node) String() string {
	s := &trieStringly{tnode: r}
	return tree.String(s)
}

// Squash compresses a Trie by removing single-branch nodes.
//
// Since 0.1.0
func (r *Node) Squash() int {

	var cnt int

	for _, n := range r.Children {
		cnt += n.Squash()
	}

	if len(r.Branches) == 1 && r.Branches[0] != leafBranch {
		child := r.Children[r.Branches[0]]
		r.Branches = child.Branches
		r.Children = child.Children
		r.Step = child.Step + 1
		cnt++
	}

	return cnt
}

// removeSameLeaf removes leaf that has the same value as preceding leaf.
//
//   a ------->e =1
//   `>b------>f =2
//     `>c->d->g =2 // "g" and "d" is removed, c has other child and is kept.
//        `--->h =3
func (r *Node) removeSameLeaf() {

	var prevValue interface{} = nil

	// wrapped as a generalized tree
	s := &trieStringly{tnode: r}

	tree.DepthFirst(s,
		func(t tree.Tree, parent, branch, node interface{}) {

			n := node.(*Node)
			needRemove := false

			v, isLeaf := t.LeafVal(node)
			if isLeaf {
				if v == prevValue {
					// same value no need to store
					needRemove = true
				} else {
					prevValue = v
				}
			} else {
				if len(n.Branches) == 0 {
					needRemove = true
				}
			}

			if needRemove && parent != nil && branch != nil {
				p := parent.(*Node)
				b := branch.(int)

				delete(p.Children, b)

				for i, bb := range p.Branches {
					if bb == b {
						p.Branches = append(p.Branches[:i], p.Branches[i+1:]...)
					}
				}
				if !isLeaf {
					r.InnerNodeCnt--
				}

			}
		})
}

// Search for `key` in a Trie.
//
// It returns 3 values of:
// The left sibling value.
// The matching value.
// The right sibling value.
//
// Any of them could be nil.
//
// Since 0.1.0
func (r *Node) Search(key []byte) (ltValue, eqValue, gtValue interface{}) {

	var eqNode = r
	var ltNode *Node
	var gtNode *Node

	lenKey := len(key)

	for i := -1; ; {
		i += int(eqNode.Step)

		if lenKey < i {
			gtNode = eqNode
			eqNode = nil
			break
		}

		var br int
		if lenKey == i {
			br = leafBranch
		} else {
			br = int(key[i])
		}

		li, ri := neighborBranches(eqNode.Branches, br)
		if li >= 0 {
			ltNode = eqNode.Children[eqNode.Branches[li]]
		}
		if ri >= 0 {
			gtNode = eqNode.Children[eqNode.Branches[ri]]
		}

		eqNode = eqNode.Children[br]

		if eqNode == nil {
			break
		}

		if br == leafBranch {
			break
		}
	}

	if ltNode != nil {
		ltValue = ltNode.rightMost().Value
	}
	if gtNode != nil {
		gtValue = gtNode.leftMost().Value
	}
	if eqNode != nil {
		eqValue = eqNode.Value
	}

	return
}

func neighborBranches(branches []int, br int) (ltIndex, rtIndex int) {

	if len(branches) == 0 {
		return
	}

	var i int
	var b int

	for i, b = range branches {
		if b >= br {
			break
		}
	}

	if b == br {
		rtIndex = i + 1
		ltIndex = i - 1

		if rtIndex == len(branches) {
			rtIndex = -1
		}
		return
	}

	if b > br {
		rtIndex = i
		ltIndex = i - 1
		return
	}

	rtIndex = -1
	ltIndex = i

	return
}

func (r *Node) leftMost() *Node {

	node := r
	for {
		if len(node.Branches) == 0 {
			return node
		}

		firstBr := node.Branches[0]
		node = node.Children[firstBr]
	}
}

func (r *Node) rightMost() *Node {

	node := r
	for {
		if len(node.Branches) == 0 {
			return node
		}

		lastBr := node.Branches[len(node.Branches)-1]
		node = node.Children[lastBr]
	}
}

// Append adds a key-value pair into Trie.
//
// The key to add must be greater than any existent key in the Trie.
//
// It returns the leaf node representing the added key.
//
// Since 0.1.0
func (r *Node) Append(key []byte, value interface{}) (leaf *Node, err error) {

	var node = r
	var j int

	for j = 0; j < len(key); j++ {
		br := int(key[j])
		if node.Children[br] == nil {
			l := len(node.Branches)
			if l > 0 && node.Branches[l-1] > br {
				err = errors.Wrapf(ErrKeyOutOfOrder, "append %q", key)
				return
			}
			break
		}
		node = node.Children[br]
	}

	if j == len(key) {
		leaf = node.Children[leafBranch]
		if leaf != nil {
			err = ErrDuplicateKeys
			return
		}

		if len(node.Branches) != 0 {
			// means this key is a prefix of an existed key, so key's adding order is not ascending.
			err = errors.Wrapf(ErrKeyOutOfOrder, "append %q is a prefix", key)
			return
		}
	}

	commonNode := node

	var ltNode *Node
	numBr := len(commonNode.Branches)
	if numBr > 0 {
		ltNode = commonNode.Children[commonNode.Branches[numBr-1]]
	}

	for _, b := range key[j:] {
		br := int(b)
		n := &Node{Children: make(map[int]*Node), Step: 1, squash: node.squash}

		node.Children[br] = n
		node.Branches = append(node.Branches, br)
		node = n

		r.InnerNodeCnt++
	}

	leaf = &Node{Value: value}

	node.Children[leafBranch] = leaf
	node.Branches = append(node.Branches, leafBranch)

	if commonNode.squash {
		if ltNode != nil {
			r.InnerNodeCnt -= ltNode.Squash()
		}
	}

	return
}
