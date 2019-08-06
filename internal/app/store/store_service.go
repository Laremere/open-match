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

	curentFrame *historicalFrame
	frames      []*historicalFrame
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

// type ticketUpdates struct {
// 	added      map[string]*pb.Ticket
// 	removed    map[string]struct{}
// 	watermarks map[*token]int64 // Does this lead to a memory leak for all tokens??
// }

// func (u *ticketUpdates) mergeIn(o *ticketUpdates) {
// 	for token, otherVal := range o.watermarks {
// 		if val, ok := u.watermarks[token]; ok {
// 			if otherVal < val {
// 				panic("Merge in should always merge in a higher watermark.")
// 			}
// 		}
// 	}

// 	for id, t := range o.added {
// 		if _, ok := u.removed[id]; ok {
// 			delete(u.removed, id)
// 		} else {
// 			u.added[id] = t
// 		}
// 	}

// 	for id := range o.removed {
// 		if _, ok := u.added[id]; ok {
// 			delete(u.removed, id)
// 		} else {
// 			u.removed[id] = struct{}{}
// 		}
// 	}
// }

// type token struct{}

// type updateChain struct {
// 	updates   *ticketUpdates
// 	nextReady chan struct{}
// }

type token struct{}

type updates struct {
	added      map[string]struct{}
	removed    map[string]struct{}
	watermarks map[*token]int64
	nextReady  chan struct{}
	next       *updates
}

type latest struct {
	values  map[string]struct{}
	updates *updates
}

type updater struct {
}

func newUpdater() *updater {

}

func (u *updater) add(v string)                {}
func (u *updater) remove(v string)             {}
func (u *updater) watermark(t *token, v int64) {}
func (u *updater) flush()                      {}

func pool(u *updates, latest chan chan latest) {
	current := make(map[string]struct{})

	if len(u.added) > 0 || len(u.removed) > 0 || len(u.watermarks) > 0 {
		panic("Pool called with unexpected circumstances.")
	}

	for {
		<-u.nextReady
		u = u.next
		for k := range u.added {
			current[k] = struct{}{}
		}
		for k := range u.removed {
			delete(current, k)
		}
	}
}
