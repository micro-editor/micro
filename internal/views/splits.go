package views

import (
	"fmt"
	"strings"
)

type SplitType uint8

const (
	STVert  = 0
	STHoriz = 1
	STUndef = 2
)

var idcounter uint64

// NewID returns a new unique id
func NewID() uint64 {
	idcounter++
	return idcounter
}

// A View is a size and location of a split
type View struct {
	X, Y int
	W, H int
}

// A Node describes a split in the tree
// If a node is a leaf node then it corresponds to a buffer that is being
// displayed otherwise it has a number of children of the opposite type
// (vertical splits have horizontal children and vice versa)
type Node struct {
	View

	Kind SplitType

	parent   *Node
	children []*Node

	// Nodes can be marked as non resizable if they shouldn't be rescaled
	// when the terminal window is resized or when a new split is added
	// Only the splits on the edges of the screen can be marked as non resizable
	canResize bool
	// A node may also be marked with proportional scaling. This means that when
	// the window is resized the split maintains its proportions
	propScale bool

	// Defines the proportion of the screen this node should take up if propScale is
	// on
	propW, propH float64
	// The id is unique for each leaf node and provides a way to keep track of a split
	// The id cannot be 0
	id uint64
}

// NewNode returns a new node with the given specifications
func NewNode(Kind SplitType, x, y, w, h int, parent *Node, id uint64) *Node {
	n := new(Node)
	n.Kind = Kind
	n.canResize = true
	n.propScale = true
	n.X, n.Y, n.W, n.H = x, y, w, h
	n.children = make([]*Node, 0)
	n.parent = parent
	n.id = id
	if parent != nil {
		n.propW, n.propH = float64(w)/float64(parent.W), float64(h)/float64(parent.H)
	} else {
		n.propW, n.propH = 1, 1
	}

	return n
}

// NewRoot returns an empty Node with a size and location
// The type of the node will be determined by the first action on the node
// In other words, a lone split is neither horizontal nor vertical, it only
// becomes one or the other after a vsplit or hsplit is made
func NewRoot(x, y, w, h int) *Node {
	n1 := NewNode(STUndef, x, y, w, h, nil, NewID())

	return n1
}

// IsLeaf returns if this node is a leaf node
func (n *Node) IsLeaf() bool {
	return len(n.children) == 0
}

// ID returns this node's id or 0 if it is not viewable
func (n *Node) ID() uint64 {
	if n.IsLeaf() {
		return n.id
	}
	return 0
}

// CanResize returns if this node can be resized
func (n *Node) CanResize() bool {
	return n.canResize
}

// PropScale returns if this node is proportionally scaled
func (n *Node) PropScale() bool {
	return n.propScale
}

// SetResize sets the resize flag
func (n *Node) SetResize(b bool) {
	n.canResize = b
}

// SetPropScale sets the propScale flag
func (n *Node) SetPropScale(b bool) {
	n.propScale = b
}

// Parent returns this node's parent
func (n *Node) Parent() *Node {
	return n.parent
}

// Children returns this node's children
func (n *Node) Children() []*Node {
	return n.children
}

// GetNode returns the node with the given id in the tree of children
// that this node has access to or nil if the node with that id cannot be found
func (n *Node) GetNode(id uint64) *Node {
	if n.id == id && n.IsLeaf() {
		return n
	}
	for _, c := range n.children {
		if c.id == id && c.IsLeaf() {
			return c
		}
		gc := c.GetNode(id)
		if gc != nil {
			return gc
		}
	}
	return nil
}

// vResizeSplit resizes a node vertically on the given side
func (n *Node) vResizeSplit(i int, size int, bottom bool) bool {
	if i < 0 || i >= len(n.children) {
		return false
	}
	var c1, c2 *Node
	if bottom {
		if i == len(n.children)-1 {
			return false
		}
		c1, c2 = n.children[i], n.children[i+1]
	} else {
		if i == 0 {
			return false
		}
		c1, c2 = n.children[i-1], n.children[i]
		size = c1.H + c2.H - size
	}
	if size <= 0 || size >= n.H {
		return false
	}
	toth := c1.H + c2.H
	if size >= toth {
		return false
	}
	c2.Y = c1.Y + size
	c1.Resize(c1.W, size)
	c2.Resize(c2.W, toth-size)
	n.markSizes()
	n.alignSizes(n.W, n.H)
	return true
}

// hResizeSplit resizes a node horizontally on the given side
func (n *Node) hResizeSplit(i int, size int, right bool) bool {
	if i < 0 || i >= len(n.children) {
		return false
	}
	var c1, c2 *Node
	if right {
		if i == len(n.children)-1 {
			return false
		}
		c1, c2 = n.children[i], n.children[i+1]
	} else {
		if i == 0 {
			return false
		}
		c1, c2 = n.children[i-1], n.children[i]
		size = c1.W + c2.W - size
	}
	if size <= 0 || size >= n.W {
		return false
	}
	totw := c1.W + c2.W
	if size >= totw {
		return false
	}
	c2.X = c1.X + size
	c1.Resize(size, c1.H)
	c2.Resize(totw-size, c2.H)
	n.markSizes()
	n.alignSizes(n.W, n.H)
	return true
}

// ResizeSplit resizes a node to a given size in the specified direction
func (n *Node) ResizeSplit(size int, right_or_bottom bool) bool {
	if len(n.parent.children) <= 1 {
		// cannot resize a lone node
		return false
	}
	ind := 0
	for i, c := range n.parent.children {
		if c.id == n.id {
			ind = i
		}
	}
	if n.parent.Kind == STVert {
		return n.parent.vResizeSplit(ind, size, right_or_bottom)
	} else if n.parent.Kind == STHoriz {
		return n.parent.hResizeSplit(ind, size, right_or_bottom)
	}
	return false
}

// Resize sets this node's size and resizes all children accordingly
func (n *Node) Resize(w, h int) {
	n.W, n.H = w, h

	if n.IsLeaf() {
		return
	}

	x, y := n.X, n.Y
	totw, toth := 0, 0
	for _, c := range n.children {
		cW := int(float64(w) * c.propW)
		cH := int(float64(h) * c.propH)

		c.X, c.Y = x, y
		c.Resize(cW, cH)
		if n.Kind == STHoriz {
			x += cW
			totw += cW
		} else {
			y += cH
			toth += cH
		}
	}

	n.alignSizes(totw, toth)
}

func (n *Node) alignSizes(totw, toth int) {
	// Make sure that there are no off-by-one problems with the rounding
	// of the sizes by making the final split fill the screen
	if n.Kind == STVert && toth != n.H {
		last := n.children[len(n.children)-1]
		last.Resize(last.W, last.H+n.H-toth)
	} else if n.Kind == STHoriz && totw != n.W {
		last := n.children[len(n.children)-1]
		last.Resize(last.W+n.W-totw, last.H)
	}
}

// Resets all proportions for children
func (n *Node) markSizes() {
	for _, c := range n.children {
		c.propW = float64(c.W) / float64(n.W)
		c.propH = float64(c.H) / float64(n.H)
		c.markSizes()
	}
}

func (n *Node) markResize() {
	n.markSizes()
	n.Resize(n.W, n.H)
}

// vsplits a vertical split and returns the id of the new split
func (n *Node) vVSplit(right bool) uint64 {
	ind := 0
	for i, c := range n.parent.children {
		if c.id == n.id {
			ind = i
		}
	}
	return n.parent.hVSplit(ind, right)
}

// hsplits a horizontal split
func (n *Node) hHSplit(bottom bool) uint64 {
	ind := 0
	for i, c := range n.parent.children {
		if c.id == n.id {
			ind = i
		}
	}
	return n.parent.vHSplit(ind, bottom)
}

// Returns the size of the non-resizable area and the number of resizable
// splits
func (n *Node) getResizeInfo(h bool) (int, int) {
	numr := 0
	numnr := 0
	nonr := 0
	for _, c := range n.children {
		if !c.CanResize() {
			if h {
				nonr += c.H
			} else {
				nonr += c.W
			}
			numnr++
		} else {
			numr++
		}
	}

	// if there are no resizable splits make them all resizable
	if numr == 0 {
		numr = numnr
	}

	return nonr, numr
}

func (n *Node) applyNewSize(size int, h bool) {
	a := n.X
	if h {
		a = n.Y
	}
	for _, c := range n.children {
		if h {
			c.Y = a
		} else {
			c.X = a
		}
		if c.CanResize() {
			if h {
				c.Resize(c.W, size)
			} else {
				c.Resize(size, c.H)
			}
		}
		if h {
			a += c.H
		} else {
			a += c.H
		}
	}
	n.markResize()
}

// hsplits a vertical split
func (n *Node) vHSplit(i int, right bool) uint64 {
	if n.IsLeaf() {
		newid := NewID()
		hn1 := NewNode(STHoriz, n.X, n.Y, n.W, n.H/2, n, n.id)
		hn2 := NewNode(STHoriz, n.X, n.Y+hn1.H, n.W, n.H/2, n, newid)
		if !right {
			hn1.id, hn2.id = hn2.id, hn1.id
		}

		n.children = append(n.children, hn1, hn2)
		n.markResize()
		return newid
	} else {
		nonrh, numr := n.getResizeInfo(true)

		// size of resizable area
		height := (n.H - nonrh) / (numr + 1)

		newid := NewID()
		hn := NewNode(STHoriz, n.X, 0, n.W, height, n, newid)

		// insert the node into the correct slot
		n.children = append(n.children, nil)
		inspos := i
		if right {
			inspos++
		}
		copy(n.children[inspos+1:], n.children[inspos:])
		n.children[inspos] = hn

		n.applyNewSize(height, true)
		return newid
	}
}

// vsplits a horizontal split
func (n *Node) hVSplit(i int, right bool) uint64 {
	if n.IsLeaf() {
		newid := NewID()
		vn1 := NewNode(STVert, n.X, n.Y, n.W/2, n.H, n, n.id)
		vn2 := NewNode(STVert, n.X+vn1.W, n.Y, n.W/2, n.H, n, newid)
		if !right {
			vn1.id, vn2.id = vn2.id, vn1.id
		}

		n.children = append(n.children, vn1, vn2)
		n.markResize()
		return newid
	} else {
		nonrw, numr := n.getResizeInfo(false)

		width := (n.W - nonrw) / (numr + 1)

		newid := NewID()
		vn := NewNode(STVert, 0, n.Y, width, n.H, n, newid)

		// Inser the node into the correct slot
		n.children = append(n.children, nil)
		inspos := i
		if right {
			inspos++
		}
		copy(n.children[inspos+1:], n.children[inspos:])
		n.children[inspos] = vn

		n.applyNewSize(width, false)
		return newid
	}
}

// HSplit creates a horizontal split and returns the id of the new split
// bottom specifies if the new split should be created on the top or bottom
// of the current split
func (n *Node) HSplit(bottom bool) uint64 {
	if !n.IsLeaf() {
		return 0
	}
	if n.parent == nil {
		n.Kind = STVert
	}
	if n.Kind == STVert {
		return n.vHSplit(0, bottom)
	}
	return n.hHSplit(bottom)
}

// VSplit creates a vertical split and returns the id of the new split
// right specifies if the new split should be created on the right or left
// of the current split
func (n *Node) VSplit(right bool) uint64 {
	if !n.IsLeaf() {
		return 0
	}
	if n.parent == nil {
		n.Kind = STHoriz
	}
	if n.Kind == STHoriz {
		return n.hVSplit(0, right)
	}
	return n.vVSplit(right)
}

// unsplits the child of a split
func (n *Node) unsplit(i int) {
	copy(n.children[i:], n.children[i+1:])
	n.children[len(n.children)-1] = nil
	n.children = n.children[:len(n.children)-1]

	h := n.Kind == STVert
	nonrs, numr := n.getResizeInfo(h)
	if numr == 0 {
		// This means that this was the last child
		// The parent will get cleaned up in the next iteration and
		// will resolve all sizing issues with its parent
		return
	}
	size := (n.W - nonrs) / numr
	if h {
		size = (n.H - nonrs) / numr
	}
	n.applyNewSize(size, h)
}

// Unsplit deletes this split and resizes everything
// else accordingly
func (n *Node) Unsplit() bool {
	if !n.IsLeaf() || n.parent == nil {
		return false
	}
	ind := 0
	for i, c := range n.parent.children {
		if c.id == n.id {
			ind = i
		}
	}
	n.parent.unsplit(ind)
	if n.parent.IsLeaf() {
		return n.parent.Unsplit()
	}

	n.parent.flatten()
	return true
}

// flattens the tree by removing unnecessary intermediate parents that have only one child
// and handles the side effect of it
func (n *Node) flatten() {
	if len(n.children) != 1 {
		return
	}

	// Special case for root node
	if n.parent == nil {
		*n = *n.children[0]
		n.parent = nil
		for _, c := range n.children {
			c.parent = n
		}
		if len(n.children) == 0 {
			n.Kind = STUndef
		}
		return
	}

	ind := 0
	for i, c := range n.parent.children {
		if c.id == n.id {
			ind = i
		}
	}

	// Replace current node with its child node to remove chained parent
	successor := n.children[0]
	n.parent.children[ind] = successor
	successor.parent = n.parent

	// Maintain the tree in a consistent state: any child node's kind (horiz vs vert)
	// should be the opposite of its parent's kind.
	if successor.IsLeaf() {
		successor.Kind = n.Kind
	} else {
		// If the successor node has children, that means it is a chained parent as well.
		// Therefore it can be replaced by its own children.
		origsize := len(n.parent.children)

		// Let's say we have 5 children and want to replace [2] with its children [a] [b] [c]
		// [0] [1] [2] [3] [4] --> [0] [1] [a] [b] [c] [3] [4]
		// insertcount will be `3 - 1 = 2` in this case
		insertcount := len(successor.children) - 1

		n.parent.children = append(n.parent.children, make([]*Node, insertcount)...)
		copy(n.parent.children[ind+insertcount+1:], n.parent.children[ind+1:origsize])
		copy(n.parent.children[ind:], successor.children)

		for i := 0; i < len(successor.children); i++ {
			n.parent.children[ind+i].parent = n.parent
		}
	}

	// Update propW and propH since the parent of the children has been updated,
	// so the children have new siblings
	n.parent.markSizes()
}

// String returns the string form of the node and all children (used for debugging)
func (n *Node) String() string {
	var strf func(n *Node, ident int) string
	strf = func(n *Node, ident int) string {
		marker := ""
		if n.Kind == STHoriz {
			marker = "-"
		} else if n.Kind == STVert {
			marker = "|"
		}

		var parentId uint64 = 0
		if n.parent != nil {
			parentId = n.parent.id
		}

		str := fmt.Sprint(strings.Repeat("\t", ident), marker, n.View, n.id, parentId)
		if n.IsLeaf() {
			str += "🍁"
		}
		str += "\n"
		for _, c := range n.children {
			str += strf(c, ident+1)
		}
		return str
	}
	return strf(n, 0)
}
