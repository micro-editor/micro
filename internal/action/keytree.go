package action

import (
	"bytes"

	"github.com/gdamore/tcell/v3"
)

type PaneKeyAction func(Pane) bool
type PaneMouseAction func(Pane, *tcell.EventMouse) bool
type PaneKeyAnyAction func(Pane, []KeyEvent) bool

// A KeyTreeNode stores a single node in the KeyTree (trie). The
// children are stored as a map, and any node may store a list of
// actions (the list will be nil if no actions correspond to a certain
// node)
type KeyTreeNode struct {
	children map[Event]*KeyTreeNode

	// Only one of these actions may be active in the current
	// mode, and only one will be returned. If multiple actions
	// are active, it is undefined which one will be the one
	// returned.
	actions []TreeAction
}

func NewKeyTreeNode() *KeyTreeNode {
	n := new(KeyTreeNode)
	n.children = make(map[Event]*KeyTreeNode)
	n.actions = []TreeAction{}
	return n
}

// A TreeAction stores an action, and a set of mode constraints for
// the action to be active.
type TreeAction struct {
	// only one of these can be non-nil
	action PaneKeyAction
	any    PaneKeyAnyAction
	mouse  PaneMouseAction

	modes []ModeConstraint
}

// A KeyTree is a data structure for storing keybindings. It maps
// key events to actions, and maintains a set of currently enabled
// modes, which affects the action that is returned for a key event.
// The tree acts like a Trie for Events to handle sequence events.
type KeyTree struct {
	root  *KeyTreeNode
	modes map[string]bool

	cursor KeyTreeCursor
}

// A KeyTreeCursor keeps track of the current location within the
// tree, and stores any information from previous events that may
// be needed to execute the action (values of wildcard events or
// mouse events)
type KeyTreeCursor struct {
	node *KeyTreeNode

	recordedEvents []Event
	wildcards      []KeyEvent
	mouseInfo      *tcell.EventMouse
}

// MakeClosure uses the information stored in a key tree cursor to construct
// a PaneKeyAction from a TreeAction (which may have a PaneKeyAction, PaneMouseAction,
// or AnyAction)
func (k *KeyTreeCursor) MakeClosure(a TreeAction) PaneKeyAction {
	if a.action != nil {
		return a.action
	} else if a.any != nil {
		return func(p Pane) bool {
			return a.any(p, k.wildcards)
		}
	} else if a.mouse != nil {
		return func(p Pane) bool {
			return a.mouse(p, k.mouseInfo)
		}
	}

	return nil
}

// NewKeyTree allocates and returns an empty key tree
func NewKeyTree() *KeyTree {
	root := NewKeyTreeNode()
	tree := new(KeyTree)

	tree.root = root
	tree.modes = make(map[string]bool)
	tree.cursor = KeyTreeCursor{
		node:      root,
		wildcards: []KeyEvent{},
		mouseInfo: nil,
	}

	return tree
}

// A ModeConstraint specifies that an action can only be executed
// while a certain mode is enabled or disabled.
type ModeConstraint struct {
	mode     string
	disabled bool
}

// RegisterKeyBinding registers a PaneKeyAction with an Event.
func (k *KeyTree) RegisterKeyBinding(e Event, a PaneKeyAction) {
	k.registerBinding(e, TreeAction{
		action: a,
		any:    nil,
		mouse:  nil,
		modes:  nil,
	})
}

// RegisterKeyAnyBinding registers a PaneKeyAnyAction with an Event.
// The event should contain an "any" event.
func (k *KeyTree) RegisterKeyAnyBinding(e Event, a PaneKeyAnyAction) {
	k.registerBinding(e, TreeAction{
		action: nil,
		any:    a,
		mouse:  nil,
		modes:  nil,
	})
}

// RegisterMouseBinding registers a PaneMouseAction with an Event.
// The event should contain a mouse event.
func (k *KeyTree) RegisterMouseBinding(e Event, a PaneMouseAction) {
	k.registerBinding(e, TreeAction{
		action: nil,
		any:    nil,
		mouse:  a,
		modes:  nil,
	})
}

func (k *KeyTree) registerBinding(e Event, a TreeAction) {
	switch ev := e.(type) {
	case KeyEvent, MouseEvent, RawEvent:
		newNode, ok := k.root.children[e]
		if !ok {
			newNode = NewKeyTreeNode()
			k.root.children[e] = newNode
		}
		// newNode.actions = append(newNode.actions, a)
		newNode.actions = []TreeAction{a}
	case KeySequenceEvent:
		n := k.root
		for _, key := range ev.keys {
			newNode, ok := n.children[key]
			if !ok {
				newNode = NewKeyTreeNode()
				n.children[key] = newNode
			}

			n = newNode
		}
		// n.actions = append(n.actions, a)
		n.actions = []TreeAction{a}
	}
}

// NextEvent returns the action for the current sequence where e is the next
// event. Even if the action was registered as a PaneKeyAnyAction or PaneMouseAction,
// it will be returned as a PaneKeyAction closure where the appropriate arguments
// have been provided.
// If no action is associated with the given Event, or mode constraints are not
// met for that action, nil is returned.
// A boolean is returned to indicate if there is a conflict with this action. A
// conflict occurs when there is an active action for this event but there are
// bindings associated with further sequences starting with this event. The
// calling function can decide what to do about the conflict (e.g. use a
// timeout).
func (k *KeyTree) NextEvent(e Event, mouse *tcell.EventMouse) (PaneKeyAction, bool) {
	n := k.cursor.node
	c, ok := n.children[e]

	if !ok {
		return nil, false
	}

	more := len(c.children) > 0

	k.cursor.node = c

	k.cursor.recordedEvents = append(k.cursor.recordedEvents, e)

	switch ev := e.(type) {
	case KeyEvent:
		if ev.any {
			k.cursor.wildcards = append(k.cursor.wildcards, ev)
		}
	case MouseEvent:
		k.cursor.mouseInfo = mouse
	}

	if len(c.actions) > 0 {
		// check if actions are active
		for _, a := range c.actions {
			active := true
			for _, mc := range a.modes {
				// if any mode constraint is not met, the action is not active
				hasMode := k.modes[mc.mode]
				if hasMode != mc.disabled {
					active = false
				}
			}

			if active {
				// the first active action to be found is returned
				return k.cursor.MakeClosure(a), more
			}
		}
	}

	return nil, more
}

// ResetEvents sets the current sequence back to the initial value.
func (k *KeyTree) ResetEvents() {
	k.cursor.node = k.root
	k.cursor.wildcards = []KeyEvent{}
	k.cursor.recordedEvents = []Event{}
	k.cursor.mouseInfo = nil
}

// RecordedEventsStr returns the list of recorded events as a string
func (k *KeyTree) RecordedEventsStr() string {
	buf := &bytes.Buffer{}
	for _, e := range k.cursor.recordedEvents {
		buf.WriteString(e.Name())
	}
	return buf.String()
}

// DeleteBinding removes any currently active actions associated with the
// given event.
func (k *KeyTree) DeleteBinding(e Event) {

}

// DeleteAllBindings removes all actions associated with the given event,
// regardless of whether they are active or not.
func (k *KeyTree) DeleteAllBindings(e Event) {

}

// SetMode enables or disabled a given mode
func (k *KeyTree) SetMode(mode string, en bool) {
	k.modes[mode] = en
}

// HasMode returns if the given mode is currently active
func (k *KeyTree) HasMode(mode string) bool {
	return k.modes[mode]
}
