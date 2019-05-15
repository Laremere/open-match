package indexer

import (
	// 	"crypto/sha512"
	// 	"encoding/base64"
	// 	"encoding/binary"
	// 	"hash"
	// 	"math"
	// 	"sort"

	"context"
	"sync"

	"open-match.dev/open-match/internal/future/pb"
)

type indexImpl interface {
	update(update)
	list(*pb.Filter) []*pb.Ticket
}

type indexNode struct {
	ctx           context.Context
	cancel        context.CancelFunc
	impl          indexImpl
	implLock      sync.RWMutex
	parentFilters []*pb.Filter
	children      map[ihash]*indexNode
	// Strict ordering: When acquiring both childrenLock and implLock, you must acquire implLock first.
	// This prevents two go routines stuck on each other: locking one, and waiting to acquire the other.
	childrenLock sync.RWMutex
	updates      chan update
}

type update struct {
	// Invariant: Ticket is not in both add and remove at the same time.
	add    map[string]*pb.Ticket
	remove map[string]*pb.Ticket
}

func (u *update) addTickets(ts []*pb.Ticket) {
	for _, t := range ts {
		u.add[t.Id] = t
	}
}

func (u *update) filterUpdates(fs []*pb.Filter) {
	for id, t := range u.add {
		if !filter(fs, t) {
			delete(u.add, id)
		}
	}
	for id, t := range u.remove {
		if !filter(fs, t) {
			delete(u.remove, id)
		}
	}
}

func (u *update) merge(u2 update) {
	// This method assumes that for a given ticket id, remove will only be called
	// after a corresponding add, and add will only be called for a new ticket id,
	// or after a remove.  Basically, an add can't follow an add, a remove can't
	// follow a remove.
	for k, v := range u2.add {
		if _, ok := u.remove[k]; ok {
			delete(u.remove, k)
		} else {
			u.add[k] = v
		}
	}

	for k, v := range u2.remove {
		if _, ok := u.add[k]; ok {
			delete(u.add, k)
		} else {
			u.remove[k] = v
		}
	}
}

func (u *update) hasUpdates() bool {
	return len(u.add) > 0 || len(u.remove) > 0
}

func (u *update) clear() {
	u.add = make(map[string]*pb.Ticket)
	u.remove = make(map[string]*pb.Ticket)
}

func startIndexNode(ctx context.Context, impl indexImpl, parentFilters []*pb.Filter) *indexNode {
	updates := make(chan update)

	ctx, cancel := context.WithCancel(ctx)

	n := indexNode{
		ctx:           ctx,
		cancel:        cancel,
		impl:          impl,
		parentFilters: parentFilters,
		children:      make(map[ihash]*indexNode),
		updates:       updates,
	}

	return &n
}

func (n *indexNode) Start() {
	go handleUpdates(n, collapseUpdates(n.updates))

}

func (n *indexNode) Stop() {
	n.cancel()
	close(n.updates)
}

func handleUpdates(n *indexNode, updates chan update) {
	for u := range updates {
		u.filterUpdates(n.parentFilters)

		n.implLock.Lock()
		n.childrenLock.RLock()

		n.impl.update(u)
		n.implLock.Unlock()

		for _, c := range n.children {
			c.updates <- u
		}
		n.childrenLock.RUnlock()
	}
}

func collapseUpdates(updates chan update) chan update {
	collapsed := make(chan update)

	go func() {
		noUpdates := make(chan update)

		var u update

		for {
			sendUpdates := noUpdates
			if u.hasUpdates() {
				sendUpdates = collapsed
			}

			select {
			case u2, ok := <-updates:
				if !ok {
					close(collapsed)
					return
				}
				u.merge(u2)
			case sendUpdates <- u:
				u.clear()
			}
		}
	}()
	return collapsed
}

func attatchToParent(child, parent *indexNode) {
	// Allow being added to the parent, but don't allow requests until ready.
	child.implLock.Lock()
	defer child.implLock.Unlock()

	// This prevents tickets being added to the parent's index before the child
	// has a chance to be added to the parent's children.
	// The parent will update it's impl, then send updates to children.
	parent.implLock.RLock()
	defer parent.implLock.RUnlock()

	parent.childrenLock.Lock()
	defer parent.childrenLock.Unlock()

	thisId := ihash("TODO")
	// Multiple queries might try to add the same index.  Abort if it was added
	// between function call and now.
	if _, ok := parent.children[thisId]; ok {
		child.cancel()
		return
	}
	child.Start()

	parent.children[thisId] = child

	startingTickets := parent.impl.list(child.parentFilters[0])

	// Actually parent locks should be released at this point....

	update := update{}
	update.addTickets(startingTickets)
	child.updates <- update
}

type ihash string

func filter(fs []*pb.Filter, t *pb.Ticket) bool {
	for _, f := range fs {
		switch f.GetValue().(type) {
		case *pb.Filter_StringFilter:
			sf := f.GetStringFilter()
			v, ok := t.Sargs[sf.Sarg]
			if !ok {
				return false
			}
			_ = v
			// TODO: string comparison methods
		case *pb.Filter_DoubleFilter:
			df := f.GetDoubleFilter()
			v, ok := t.Dargs[df.Darg]
			if !ok {
				return false
			}
			if v < df.Min || v > df.Max {
				return false
			}
		case *pb.Filter_PoolIndex:
			// Nothing to do.
		default:
			panic("Unknown filter type encountered.")
		}
	}
	return true
}
