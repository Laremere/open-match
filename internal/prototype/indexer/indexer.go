package indexer

import (
	// 	"crypto/sha512"
	// 	"encoding/base64"
	// 	"encoding/binary"
	// 	"hash"
	// 	"math"
	// 	"sort"

	"sync"

	"open-match.dev/open-match/internal/future/pb"
)

type indexImpl interface {
	update(update)
	list(*pb.Filter) []*pb.Ticket
}

type indexNode struct {
	ctx           context.context
	impl          indexImpl
	implLock      sync.RWMutex
	parentFilters []*pb.Filter
	children      map[ihash]*indexNode
	// Strict ordering: When acquiring both childrenLock and implLock, you must acquire implLock first.
	// This prevents two go routines stuck on each other: locking one, and waiting to acquire the other.
	childrenLock sync.RWMutex
	updates      <-chan update
}

type update struct {
	add    map[string]*pb.Ticket
	remove map[string]*pb.Ticket
}

func (u *update) add(ts []*pb.Ticket) {
	for _, t := range ts {
		u.add[t.Id] = t
	}
}

func startIndexNode(ctx context.context, impl indeximpl, parentFilters []*pb.Filter, parent *indexNode) {
	updates := make(chan update)

	n := indexNode{
		impl:          impl,
		parentFilters: parentFilters,
		children:      make(map[ihash]*indexNode),
		updates:       updates,
	}

	thisId := ihash("TODO")
	var startingTickets []*Ticket

	// Allow being added to the parent, but don't allow requests until ready.
	n.implLock.Lock()
	defer n.implLock.Unlock()

	if parent != nil {
		// This prevents tickets being added to the parent's index before the child
		// has a chance to be added to the parent's children.
		// The parent will update it's impl, then send updates to children.
		parent.implLock.RLock()
		defer parent.implLock.RUnlock()

		parent.childrenLock.Lock()
		defer parent.childrenLock.Unlock()

		// Multiple queries might try to add the same index.  Abort if it was added
		// between function call and now.
		if _, ok := parent.children[thisId]; ok {
			return
		}

		parent.children[thisId] = &n

		startingTickets = parent.impl.list(parentFilters[0])
	}

	update := update{}
	update.add(startingTickets)

	go runIndexNode()
}

func runIndexNode(n *indexNode, updates chan update) {
}

type ihash string
