package arweave

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	NODE_TYPE_ROOT   = 0
	NODE_TYPE_BRANCH = 1
	NODE_TYPE_LEAF   = 2
)

type (
	node struct {
		id       [32]byte
		nodeType uint64   // 0 root, 1 branch, 2 leaf
		data     [32]byte // hash of data
		note     big.Int
		left     [32]byte
		right    [32]byte
		max      big.Int
	}

	element struct {
		data [32]byte // hash of data
		note big.Int
	}
)

func GenerateTree(elements []*element) ([32]byte, []*node) {
	return generateAllRows(generateLeaves(elements))
}

func generateLeaves(elements []*element) []*node {
	nodes := make([]*node, len(elements))
	for i, e := range elements {
		h0 := sha256.Sum256(e.data[:])
		h1 := sha256.Sum256(noteToBinary(&e.note))
		v0 := append(h0[:], h1[:]...)
		h := sha256.Sum256(v0)
		// TODO: Should check note is ordered?
		nodes[i] = &node{
			id:       h,
			nodeType: NODE_TYPE_LEAF, // leaf
			note:     e.note,
			max:      e.note,
			data:     e.data,
		}
	}
	return nodes
}

func noteToBinary(note *big.Int) []byte {
	return common.LeftPadBytes(note.Bytes(), 32)
}

func generateNode(left *node, right *node) *node {
	if right == nil {
		return left
	}

	h0 := sha256.Sum256(left.id[:])
	h1 := sha256.Sum256(right.id[:])
	h2 := sha256.Sum256(noteToBinary(&left.max))
	v0 := append(h0[:], h1[:]...)
	v1 := append(v0, h2[:]...)

	return &node{
		nodeType: NODE_TYPE_BRANCH,
		id:       sha256.Sum256(v1),
		left:     left.id,
		right:    right.id,
		note:     left.max,
		max:      right.max,
	}
}

func generateRow(nodes []*node) []*node {
	newNodes := make([]*node, 0)
	for i := 0; i < len(nodes); i += 2 {
		left := nodes[i]
		var right *node = nil
		if i+1 < len(nodes) {
			right = nodes[i+1]
		}
		newNodes = append(newNodes, generateNode(left, right))
	}
	return newNodes
}

func generateAllRows(row []*node) ([32]byte, []*node) {
	tree := row
	var hash [32]byte

	if len(row) == 0 {
		return hash, tree
	}

	cRow := row

	for {
		if len(cRow) <= 1 {
			break
		}

		newRow := generateRow(cRow)

		tree = append(newRow, tree...)

		cRow = newRow
	}

	return tree[0].id, tree
}

// generate the proof of the nearest leaf with note < dest
func GeneratePath(root [32]byte, dest *big.Int, tree []*node) []byte {
	parts := generatePathParts(root, dest, tree)
	path := make([]byte, 0)
	for _, part := range parts {
		path = append(path, part...)
	}
	return path
}

func pathToParts(path []byte) [][]byte {
	if len(path)%32 != 0 {
		// TODO: convert to error
		panic("inccorect path")
	}

	parts := make([][]byte, len(path)/32)

	for i := 0; i < len(path); i += 32 {
		parts[i/32] = path[i : i+32]
	}
	return parts
}

func getNode(id [32]byte, tree []*node) *node {
	for _, n := range tree {
		if id == n.id {
			return n
		}
	}
	return nil
}

func generatePathParts(id [32]byte, dest *big.Int, tree []*node) [][]byte {
	n := getNode(id, tree)
	r := make([][]byte, 0)
	if n.nodeType == NODE_TYPE_LEAF {
		return append(r, n.data[:], noteToBinary(&n.note))
	} else if n.nodeType == NODE_TYPE_BRANCH {
		var nextNodeId [32]byte
		if dest.Cmp(&n.note) < 0 {
			nextNodeId = n.left
		} else {
			nextNodeId = n.right
		}
		r = append(r, n.left[:], n.right[:], noteToBinary(&n.note))
		return append(r, generatePathParts(nextNodeId, dest, tree)...)
	} else {
		panic("nodeType not supported")
	}
}

func ValidatePath(root [32]byte, dest *big.Int, rightBound *big.Int, path []byte) ([32]byte, *big.Int, *big.Int, error) {
	if dest.Cmp(big.NewInt(0)) < 0 {
		dest = big.NewInt(0)
	}

	if dest.Cmp(rightBound) >= 0 {
		dest = big.NewInt(0).Sub(rightBound, big.NewInt(1))
	}

	return validatePath(root, dest, big.NewInt(0), rightBound, path)
}

func bigMax(l *big.Int, r *big.Int) *big.Int {
	if l.Cmp(r) < 0 {
		return r
	} else {
		return l
	}
}

func bigMin(l *big.Int, r *big.Int) *big.Int {
	if l.Cmp(r) < 0 {
		return l
	} else {
		return r
	}
}

func validatePath(id [32]byte, dest *big.Int, leftBound *big.Int, rightBound *big.Int, path []byte) ([32]byte, *big.Int, *big.Int, error) {
	parts := pathToParts(path)

	for len(parts) > 0 {
		if len(parts) > 2 {
			left := parts[0]
			right := parts[1]
			note := parts[2]
			h0 := sha256.Sum256(left)
			h1 := sha256.Sum256(right)
			h2 := sha256.Sum256(note)
			v0 := append(h0[:], h1[:]...)
			v1 := append(v0, h2[:]...)
			h := sha256.Sum256(v1)

			if h != id {
				return [32]byte{}, nil, nil, fmt.Errorf("hash mismatch")
			}

			noteValue := new(big.Int).SetBytes(note)
			if dest.Cmp(noteValue) < 0 {
				// leftBound unchanged
				rightBound = bigMin(noteValue, rightBound)
				copy(id[:], left)
			} else {
				// rightBound unchanged
				leftBound = bigMax(noteValue, leftBound)
				copy(id[:], right)
			}

			parts = parts[3:]
		} else if len(parts) == 2 {
			// must be 2
			h0 := sha256.Sum256(parts[0]) // data hash
			h1 := sha256.Sum256(parts[1]) // note
			v0 := append(h0[:], h1[:]...)
			h := sha256.Sum256(v0)

			if h != id {
				return [32]byte{}, nil, nil, fmt.Errorf("hash mismatch")
			}

			noteValue := new(big.Int).SetBytes(parts[1])

			var bs [32]byte
			copy(bs[:], parts[0])
			return bs, leftBound, bigMax(bigMin(rightBound, noteValue), big.NewInt(0).Add(leftBound, big.NewInt(1))), nil
		} else {
			return [32]byte{}, nil, nil, fmt.Errorf("insufficient proof")
		}
	}

	// never reach here
	return [32]byte{}, nil, nil, nil
}
