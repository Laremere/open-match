package indexer

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"hash"
	"math"
	"sort"

	"open-match.dev/open-match/internal/future/pb"
)

type index interface {
	add([]*pb.Ticket)
	remove(map[string]struct{})
	list(*pb.Query) []*pb.Ticket
}

type Indexer struct {
	tickets map[string]*pb.Ticket
	indexes map[string]index
}

func (i *Indexer) Add(t *pb.Ticket) {

}

func (i *Indexer) Remove(t *pb.Ticket) {

}

func (i *Indexer) List(q *pb.Query) []*pb.Ticket {
	return nil
}

////////////////////////////////////////////////////////////

// Only calling list from multiple routines is thread safe.
type poolIndex struct {
	filters []*pb.Filter
	tickets map[string]*pb.Ticket
}

func (p *poolIndex) add(ts []*pb.Ticket) {
	for _, t := range ts {
		if withinFilter(t, p.filters) {
			p.tickets[t.Id] = t
		}
	}
}

func (p *poolIndex) remove(ts map[string]struct{}) {
	for id := range ts {
		delete(p.tickets, id)
	}
}

func (p *poolIndex) list(q *pb.Query) []*pb.Ticket {
	result := make([]*pb.Ticket, 0, len(p.tickets))

	for _, t := range p.tickets {
		// Filtering done outside this call to reduce time using contested memory.
		// if withinFilter(t, q.Filters) {
		result = append(result, t)
		// }
	}

	return result
}

////////////////////////////////////////////////////////////

// This is a rather poor implementation, as I don't think this
// method off adding and removing tickets is very efficient.
// There may be other method (eg, red black trees) which have
// faster add and removals.  This just is a simple way to get
// the prototype working.
type rangeIndex struct {
	tickets []*pb.Ticket
	darg    string
}

func (r *rangeIndex) add(ts []*pb.Ticket) {
	r.tickets = append(r.tickets, ts...)
	sort.Sort(r)

	// todo remove duplicates, or prevent them from being added at a
	// system level.
}

// TODO: Consider combining add and remove into an update method?
func (r *rangeIndex) remove(toRemove map[string]struct{}) {
	j := 0
	for _, t := range r.tickets {
		if _, ok := toRemove[t.Id]; !ok {
			r.tickets[j] = t
			j++
		}
	}
	r.tickets = r.tickets[:j]
}

func (r *rangeIndex) list(q *pb.Query) []*pb.Ticket {
	// Can't return direct slice, in case this is still being serialized
	// out of an rpc connection while values are added or removed.  If add
	// and remove were called in a single update method, it would make more
	// sense to instead build a new array after each update, so that a slice
	// can build returned directly.

	lower := math.Inf(-1)
	upper := math.Inf(1)

	for _, f := range q.Filters {
		switch f.Value.(type) {
		case *pb.Filter_RangeFilter:
			rf := f.GetRangeFilter()
			lower = math.Max(lower, rf.Min)
			upper = math.Min(upper, rf.Max)
		}
	}

	// Consider caching the values for fewer memory indirections, then use
	// the sort.search designed for float64s.
	min := sort.Search(len(r.tickets), func(i int) bool { return r.tickets[i].Dargs[r.darg] <= lower })
	max := sort.Search(len(r.tickets), func(i int) bool { return r.tickets[i].Dargs[r.darg] <= upper })

	result := make([]*pb.Ticket, max-min)
	copy(r.tickets[min:max], result)
	return result
}

func (r *rangeIndex) Len() int {
	return len(r.tickets)
}

func (r *rangeIndex) Less(i, j int) bool {
	return r.tickets[i].Dargs[r.darg] <= r.tickets[j].Dargs[r.darg]
}

func (r *rangeIndex) Swap(i, j int) {
	r.tickets[i], r.tickets[j] = r.tickets[j], r.tickets[i]
}

////////////////////////////////////////////////////////////////////////////

func withinFilter(t *pb.Ticket, filters []*pb.Filter) bool {
	// TODO: before calling into this point, the system should verify
	// that all filters have value set, and return to the rpc caller if it
	// hasn't been, because they're either using it wrong, or they're using
	// a filter which isn't available on this version of om.
	for _, f := range filters {
		switch f.Value.(type) {
		case *pb.Filter_RangeFilter:
			rf := f.GetRangeFilter()

			v, ok := t.Dargs[rf.Darg]
			if !ok || v < rf.Min || v >= rf.Max {
				return false
			}
		default:
			panic("Unknown filter reached")
		}
	}

	return true
}

////////////////////////////////////////////////////////////////////////////

func indexHash(i *pb.Index) string {
	// Perhaps put more thought into hash choice.. seems like sha512 is the latest most
	// secure version, and 224 is the shortest version, as collisions are very unlikely.

	// On the other hand, but not using a crpyto hash would be fine?  64 bits seems just barely
	// too few to never expect a collision on any instance of open matfch.
	h := sha512.New512_224()
	recursiveIndexHash(h, i)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func recursiveIndexHash(h hash.Hash, i *pb.Index) {
	switch i.Value.(type) {
	case *pb.Index_PoolIndex:
		binary.Write(h, binary.LittleEndian, "p")
		pi := i.GetPoolIndex()
		recursiveIndexHash(h, pi.Index)
		for _, f := range pi.Filters {
			switch f.Value.(type) {
			case *pb.Filter_RangeFilter:
				rf := f.GetRangeFilter()
				binary.Write(h, binary.LittleEndian, "rf")
				binary.Write(h, binary.LittleEndian, rf.Darg)
				binary.Write(h, binary.LittleEndian, rf.Min)
				binary.Write(h, binary.LittleEndian, rf.Max)

			default:
				panic("Unknown filter reached")
			}
		}

	case *pb.Index_RangeIndex:
		binary.Write(h, binary.LittleEndian, "r")
		ri := i.GetRangeIndex()
		binary.Write(h, binary.LittleEndian, ri.Darg)

	default:
		panic("Unknown index reached")
	}
}
