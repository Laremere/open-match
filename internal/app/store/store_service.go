// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store

import (
	"context"
	"time"

	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/pkg/pb"
)

type service struct {
	active   map[string]*t
	pending  map[string]*t
	archived map[string]*t

	currentFrame *historicalFrame
	frames       []*historicalFrame
}

type t struct {
	ticket     *pb.Ticket
	assignment *pb.Assignment
	canceled   bool
}

// Hm, it's hard with this model to know where a ticket is within a historical frame.
type historicalFrame struct {
	lastCall time.Time
	tickets  []string
}

func newStoreService(cfg config.View) *service {
	return &service{
		active:       make(map[string]*t),
		pending:      make(map[string]*t),
		archived:     make(map[string]*t),
		currentFrame: &historicalFrame{},
	}
}

// Do we need a get ticket???? I mean, probably.

func (s *service) IndexUpdates(stream ipb.Store_IndexUpdatesServer) error {
	go func() {
		for {
			r, err := stream.Recv()
			if err != nil {

			}
			_ = r.Watermark
		}
	}()

	return nil
}

func (s *service) DeindexTicket(ctx context.Context, r *ipb.DeindexTicketRequest) (*ipb.DeindexTicketResponse, error) {
	return nil, nil
}

func (s *service) CreateTicket(ctx context.Context, r *ipb.CreateTicketRequest) (*ipb.CreateTicketResponse, error) {

	return &ipb.CreateTicketResponse{}, nil
}

func (s *service) AssignTickets(ctx context.Context, r *ipb.AssignTicketsRequest) (*ipb.AssignTicketsResponse, error) {
	return nil, nil
}

func (s *service) AssignmentSubscribe(stream ipb.Store_AssignmentSubscribeServer) error {
	return nil
}

type watermark struct{}

type updates struct {
	// Only once ready is closed is it ok to read
	// any of the rest of the fields.
	ready chan struct{}

	added     []string
	removed   []string
	watermark interface{}
	next      *updates
}

type latest struct {
	values  map[string]struct{}
	updates *updates
}

type updater struct {
	pending  *updates
	adding   map[string]struct{}
	removing map[string]struct{}
}

func newUpdater() *updater {
	u := &updater{
		pending: &updates{
			ready: make(chan struct{}),
		},
		adding:   make(map[string]struct{}),
		removing: make(map[string]struct{}),
	}

	requests := make(chan chan *updates)

	go pool(u.pending, requests)

	return u
}

func (u *updater) add(v string) {
	if _, ok := u.removing[v]; ok {
		delete(u.removing, v)
	} else {
		u.adding[v] = struct{}{}
	}
	if len(u.adding) > 100 {
		u.flush()
	}
}

func (u *updater) remove(v string) {
	if _, ok := u.adding[v]; ok {
		delete(u.adding, v)
	} else {
		u.removing[v] = struct{}{}
	}
	if len(u.adding) > 100 {
		u.flush()
	}
}

func (u *updater) watermark(w interface{}) {
	u.pending.watermark = w
	u.flush()
}

func (u *updater) flush() {
	for v := range u.adding {
		u.pending.added = append(u.pending.added, v)
		delete(u.adding, v)
	}
	for v := range u.removing {
		u.pending.removed = append(u.pending.removed, v)
		delete(u.removing, v)
	}

	u.pending.next = &updates{}
	close(u.pending.ready)
	u.pending = u.pending.next
}

func pool(u *updates, requests chan chan *updates) {
	current := make(map[string]struct{})

	closed := make(chan struct{})
	close(closed)

	for {
		select {
		case <-u.ready:
			for _, v := range u.added {
				current[v] = struct{}{}
			}
			for _, v := range u.removed {
				delete(current, v)
			}
			u = u.next
		case req := <-requests:
			resp := updates{
				ready: closed,
				next:  u,
			}
			for k := range current {
				resp.added = append(resp.added, k)
			}
			resp.next = u
			req <- &resp
		}
	}
}

// NOTE to improve:  It seems that both pool want to take, and then give to callers
// only next and nextReady.  Well, actually, it could just give the first update with all
// only added... that could work too.

type store struct {
	add       chan string
	remove    chan string
	watermark chan interface{}
	updater   *updater
}

func (s *store) add(v string) {
	s.add <- v
}

func (s *store) remove(v string) {
	s.remove <- v
}

func (s *store) watermark(w interface{}) {
	s.watermark <- v
}

func (s *store) subscribe() *updater {
	return nil
}

func NewStore(ctx content.Context) {

}

func (s *store) run() {
	current := make(map[string]struct{})
}
