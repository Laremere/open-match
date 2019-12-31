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
	// "encoding/json"
	// "fmt"
	// "io"
	// "net/http"
	// "strings"
	"sync"
	"time"

	// "github.com/golang/protobuf/jsonpb"
	"github.com/sirupsen/logrus"
	// "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/ipb"
	// "open-match.dev/open-match/internal/rpc"
	// "open-match.dev/open-match/internal/statestore"
	// "open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

type storeService struct {
	lock sync.RWMutex

	tickets map[string]*ticketState
	frozen  bool

	pendingUpdate *update
}

func newStoreService() *storeService {
	return &storeService{
		tickets: map[string]*ticketState{},
		frozen:  false,

		pendingUpdate: &update{
			ready: make(chan struct{}),
		},
	}
}

type ticketState struct {
	ticket         *pb.Ticket
	pendingTimeout *time.Time // TODO: When restoring from file, just set pending
	assignment     *pb.Assignment
}

type update struct {
	// Only read other values once ready is closed.
	ready chan struct{}
	next  *update
	u     FirehoseResponse_Update
}

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.store",
	})
)

func (s *storeService) releaseUpdate() {
	s.pendingUpdate.next = &update{
		ready: make(chan struct{}),
	}
	close(s.pendingUpdate.ready)
	s.pendingUpdate = s.pendingUpdate.next
}

func (s *storeService) CreateTicket(ctx context.Context, req *ipb.CreateTicketRequest) (*ipb.CreateTicketResponse, error) {
	if req.Ticket == nil {
		return nil, status.Errorf(codes.InvalidArgument, "ticket required")
	}
	if req.Ticket.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ticket.id required")
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.tickets[req.Ticket.Id]; ok {
		return nil, status.Errorf(codes.AlreadyExists, "ticket with id already exists")
	}

	s.tickets[req.Ticket.Id] = &ticketState{
		ticket: req.Ticket,
	}

	// s.pendingUpdate.newTicket = req.Ticket
	s.releaseUpdate()

	// TODO: save to disk.
	return &ipb.CreateTicketResponse{}, nil
}

func (s *storeService) Firehose(stream ipb.Store_FirehoseServer) error {
	return nil
}

func (s *storeService) startFirehose() *update {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// Possible improvement: This could be cached for a short time.
	return nil
}

func (s *storeService) Freeze(ctx context.Context, req *ipb.FreezeRequest) (*ipb.FreezeResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// TODO:Send update, save to disk.
	return nil, nil
}

func (s *storeService) GetTicket(ctx context.Context, req *ipb.GetTicketRequest) (*ipb.GetTicketResponse, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ts, ok := s.Tickets[req.Id]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "ticket with id not found")
	}

	return &ipb.GetTicketResponse{
		Ticket:     ts.ticket,
		Assignment: ts.assignment,
	}, nil
}

func (s *storeService) UpdateState(ctx context.Context, req *ipb.UpdateStateRequest) (*ipb.UpdateStateResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// TODO:Send update, save to disk.
	return nil, nil
}

// CreateTickets = modify and send updates
// UpdateState = modify and send updates
// Freeze = modify and send updates, pause some updates to query services?  Maybe just update has a bool of frozen or not?

// Firehose = get current whole state, subscribe to updates

// Get Ticket = Read current state.
