package views

import (
	"fmt"
	"reflect"
	"testing"
)

// nestedSplits reproduces #3071: vsplit, hsplit, vsplit. Transposing the
// layout exercises the corresponding horizontal divider behavior.
func nestedSplits(transpose, nestedFirst bool) *Node {
	root := NewRoot(0, 0, 80, 80)
	first := root.ID()
	split := func(n *Node, vertical bool) uint64 {
		if vertical != transpose {
			return n.VSplit(true)
		}
		return n.HSplit(true)
	}
	nested := split(root, true)
	if nestedFirst {
		nested = first
	}
	last := split(root.GetNode(nested), false)
	split(root.GetNode(last), true)
	return root
}

func checkSplitLayout(t *testing.T, n *Node) {
	t.Helper()
	if n.W < 1 || n.H < 1 {
		t.Fatalf("collapsed node: %v\n%s", n.View, n)
	}
	x, y := n.X, n.Y
	for _, c := range n.children {
		if c.parent != n || c.X != x || c.Y != y {
			t.Fatalf("incorrect parent or position: child %v, expected (%d,%d)\n%s", c.View, x, y, n)
		}
		if n.Kind == STHoriz {
			if c.H != n.H {
				t.Fatalf("child does not fill parent height: %v in %v", c.View, n.View)
			}
			x += c.W
		} else {
			if c.W != n.W {
				t.Fatalf("child does not fill parent width: %v in %v", c.View, n.View)
			}
			y += c.H
		}
		checkSplitLayout(t, c)
	}
	if !n.IsLeaf() && ((n.Kind == STHoriz && x != n.X+n.W) || (n.Kind == STVert && y != n.Y+n.H)) {
		t.Fatalf("children do not tile their parent\n%s", n)
	}
}

type splitState struct {
	view         View
	propW, propH float64
	id           uint64
}

func splitSnapshot(n *Node) []splitState {
	result := []splitState{{n.View, n.propW, n.propH, n.id}}
	for _, c := range n.children {
		result = append(result, splitSnapshot(c)...)
	}
	return result
}

func TestNestedResizeSplitMinimum(t *testing.T) {
	for _, transpose := range []bool{false, true} {
		for _, nestedFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("transpose=%v/nestedFirst=%v", transpose, nestedFirst), func(t *testing.T) {
				root := nestedSplits(transpose, nestedFirst)
				// The nested subtree needs two cells along the resize axis.
				min, max := 1, 78
				if nestedFirst {
					min, max = 2, 79
				}
				for size := -1; size <= 81; size++ {
					before := splitSnapshot(root)
					want := size >= min && size <= max
					if got := root.children[0].ResizeSplit(size); got != want {
						t.Fatalf("ResizeSplit(%d) = %v, want %v", size, got, want)
					}
					if !want && !reflect.DeepEqual(before, splitSnapshot(root)) {
						t.Fatalf("rejected resize %d changed the tree", size)
					}
					checkSplitLayout(t, root)
				}
				for size := max; size >= min; size-- {
					if !root.children[0].ResizeSplit(size) {
						t.Fatalf("valid resize %d rejected", size)
					}
					checkSplitLayout(t, root)
				}
			})
		}
	}
}

func TestNestedResizePreservesProportions(t *testing.T) {
	for _, transpose := range []bool{false, true} {
		t.Run(fmt.Sprint(transpose), func(t *testing.T) {
			root := nestedSplits(transpose, false)
			before := splitSnapshot(root)
			for cycle := 0; cycle < 3; cycle++ {
				for size := 41; size <= 65; size++ {
					root.children[0].ResizeSplit(size)
				}
				for size := 64; size >= 40; size-- {
					root.children[0].ResizeSplit(size)
				}
				if !reflect.DeepEqual(before, splitSnapshot(root)) {
					t.Fatalf("dragging changed descendant proportions\n%s", root)
				}
			}
		})
	}
}

func TestResizeReservesDescendantSpace(t *testing.T) {
	for _, transpose := range []bool{false, true} {
		t.Run(fmt.Sprint(transpose), func(t *testing.T) {
			root := nestedSplits(transpose, false)
			nested := root.children[1].children[1]
			// A narrow first child must retain a cell even when its proportional
			// share rounds down to zero after resizing an ancestor.
			if !nested.children[0].ResizeSplit(1) {
				t.Fatal("initial resize failed")
			}
			if !root.children[0].ResizeSplit(78) {
				t.Fatal("minimum-size subtree was rejected")
			}
			checkSplitLayout(t, root)
			if transpose {
				if nested.children[0].H != 1 || nested.children[1].H != 1 {
					t.Fatal("minimum heights not respected")
				}
			} else if nested.children[0].W != 1 || nested.children[1].W != 1 {
				t.Fatal("minimum widths not respected")
			}
		})
	}
}

func TestResizeWholeTreeMinimumAndRecovery(t *testing.T) {
	for _, transpose := range []bool{false, true} {
		t.Run(fmt.Sprint(transpose), func(t *testing.T) {
			root := nestedSplits(transpose, false)
			before := splitSnapshot(root)
			for _, size := range []int{79, 25, 3, 1, 0, -1} {
				root.Resize(size, size)
				checkSplitLayout(t, root)
			}
			// Smaller terminals cannot fit all panes: keep the minimum layout
			// intact, and recover its proportions when space becomes available.
			wantW, wantH := 3, 2
			if transpose {
				wantW, wantH = 2, 3
			}
			if root.W != wantW || root.H != wantH {
				t.Fatalf("minimum root size: got %dx%d, want %dx%d", root.W, root.H, wantW, wantH)
			}
			root.Resize(80, 80)
			if !reflect.DeepEqual(before, splitSnapshot(root)) {
				t.Fatal("window resize lost original proportions")
			}
		})
	}
}

func TestResizeThreeDescendants(t *testing.T) {
	root := nestedSplits(false, false)
	nested := root.children[1].children[1]
	nested.children[0].VSplit(false)
	if !root.children[0].ResizeSplit(77) {
		t.Fatal("three-cell subtree rejected")
	}
	checkSplitLayout(t, root)
	before := splitSnapshot(root)
	if root.children[0].ResizeSplit(78) {
		t.Fatal("allowed three children in two cells")
	}
	if !reflect.DeepEqual(before, splitSnapshot(root)) {
		t.Fatal("rejected resize changed layout")
	}
	// Closing a pane must still flatten the tree and leave it resizable.
	id := nested.children[0].ID()
	if !root.GetNode(id).Unsplit() {
		t.Fatal("closing pane failed")
	}
	checkSplitLayout(t, root)
	root.Resize(81, 79)
	checkSplitLayout(t, root)
}

func TestSplitAtMinimumRelayoutsAncestors(t *testing.T) {
	for _, transpose := range []bool{false, true} {
		t.Run(fmt.Sprint(transpose), func(t *testing.T) {
			root := NewRoot(0, 0, 80, 80)
			var right, bottom uint64
			if transpose {
				right = root.HSplit(true)
				bottom = root.GetNode(right).VSplit(true)
			} else {
				right = root.VSplit(true)
				bottom = root.GetNode(right).HSplit(true)
			}
			if !root.children[0].ResizeSplit(79) {
				t.Fatal("initial resize failed")
			}
			var last uint64
			if transpose {
				last = root.GetNode(bottom).HSplit(true)
			} else {
				last = root.GetNode(bottom).VSplit(true)
			}
			if root.GetNode(last) == nil {
				t.Fatal("new pane missing")
			}
			checkSplitLayout(t, root)
			if root.W != 80 || root.H != 80 {
				t.Fatal("layout grew despite available space in the root")
			}
			if !root.children[0].ResizeSplit(40) {
				t.Fatal("restoring outer size failed")
			}
			first, second := root.GetNode(bottom), root.GetNode(last)
			if transpose {
				if first.H != 20 || second.H != 20 {
					t.Fatal("new horizontal splits lost equal proportions")
				}
			} else if first.W != 20 || second.W != 20 {
				t.Fatal("new vertical splits lost equal proportions")
			}
		})
	}
}
