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
	ipb "open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/pkg/pb"
)

type service struct {
	tickets          map[string]*pb.Ticket
	assignments      map[string]*pb.Assignment
	indexed          map[string]struct{}
	historicalFrames []*historicalFrame
}

// Hm, it's hard with this model to know where a ticket is within a historical frame.
type historicalFrame struct {
	lastCall time.Time
	tickets  []string

	// How man slots in tickets are the empty string
	emptySlots int
}

func newStoreService(cfg config.View) *service {
	return &service{}
}

func (s *service) IndexUpdates(stream ipb.Store_IndexUpdatesServer) error {
	return nil
}

func (s *service) DeindexTicket(ctx context.Context, r *ipb.DeindexTicketRequest) (*ipb.DeindexTicketResponse, error) {
	return nil, nil
}

func (s *service) CreateTicket(ctx context.Context, r *ipb.CreateTicketRequest) (*ipb.CreateTicketResponse, error) {
	return nil, nil
}

func (s *service) AssignTickets(ctx context.Context, r *ipb.AssignTicketsRequest) (*ipb.AssignTicketsResponse, error) {
	return nil, nil
}

gfunc (s *service) AssignmentSubscribe(stream ipb.Store_AssignmentSubscribeServer) error {
	return nil
}
