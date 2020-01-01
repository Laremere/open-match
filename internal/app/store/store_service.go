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

	watermark     uint64
	pendingUpdate *update
}

func newStoreService() *storeService {
	return &storeService{
		tickets: map[string]*ticketState{},
		frozen:  false,

		watermark: 2,
		pendingUpdate: &update{
			ready: make(chan struct{}),
			firehose: &ipb.FirehoseResponse{
				Watermark: 3,
			},
		},
	}
}

type ticketState struct {
	ticket     *pb.Ticket
	pending    bool
	assignment *pb.Assignment
}

type update struct {
	// Only read other values once ready is closed.
	ready    chan struct{}
	next     *update
	firehose *ipb.FirehoseResponse
}

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.store",
	})
)

func (s *storeService) releaseUpdate() {
	// TODO: Save to disk.
	s.watermark = s.pendingUpdate.firehose.Watermark
	s.pendingUpdate.next = &update{
		ready: make(chan struct{}),
		firehose: &ipb.FirehoseResponse{
			Watermark: s.watermark + 1,
		},
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

	s.pendingUpdate.firehose.Update = &ipb.FirehoseResponse_NewTicket{req.Ticket}
	s.releaseUpdate()

	return &ipb.CreateTicketResponse{}, nil
}

func (s *storeService) Firehose(req *ipb.FirehoseRequest, stream ipb.Store_FirehoseServer) error {
	return nil
}

func (s *storeService) startFirehose() ([]*ipb.FirehoseResponse, *update) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// Possible improvement: This could be cached for a short time.
	return nil, nil
}

func (s *storeService) Freeze(ctx context.Context, req *ipb.FreezeRequest) (*ipb.FreezeResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if req.Freeze == s.frozen {
		return &ipb.FreezeResponse{}, nil
	}

	// TODO:Send update, save to disk.
	return nil, nil
}

func (s *storeService) GetTicket(ctx context.Context, req *ipb.GetTicketRequest) (*ipb.GetTicketResponse, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ts, ok := s.tickets[req.Id]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "ticket with id not found")
	}

	// TODO: Add watermark, would be useful when waiting on ticket assignment.
	return &ipb.GetTicketResponse{
		Ticket:     ts.ticket,
		Assignment: ts.assignment,
	}, nil
}

func (s *storeService) AssignTickets(ctx context.Context, req *ipb.AssignTicketsRequest) (*ipb.AssignTicketsResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, id := range req.Ids {
		ts, ok := s.tickets[id]
		if !ok {
			return nil, status.Errorf(codes.NotFound, "ticket with id not found")
		}
		ts.assignment = req.Assignment

		s.pendingUpdate.firehose.Update = &ipb.FirehoseResponse_AssignedId{id}
		s.releaseUpdate()
	}

	return nil, nil
}

func (s *storeService) GetCurrentWatermark(ctx context.Context, req *ipb.GetCurrentWatermarkRequest) (*ipb.GetCurrentWatermarkResponse, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return &ipb.GetCurrentWatermarkResponse{
		Watermark: s.watermark,
	}, nil
}

func (s *storeService) MarkPending(ctx context.Context, req *ipb.MarkPendingRequest) (*ipb.MarkPendingResponse, error) {
	// TODO: Add timeout mechanism, can be in memory only (reseting timeout over crashes is fine)

	s.lock.Lock()
	defer s.lock.Unlock()

	for _, id := range req.Ids {
		ts, ok := s.tickets[id]
		if !ok {
			// TODO: return list of failed pendings?
			continue
		}

		ts.pending = true

		s.pendingUpdate.firehose.Update = &ipb.FirehoseResponse_PendingId{id}
	}

	return &ipb.MarkPendingResponse{}, nil
}

func (s *storeService) DeleteTicket(ctx context.Context, req *ipb.DeleteTicketRequest) (*ipb.DeleteTicketResponse, error) {

	return nil, nil
}

// CreateTickets = modify and send updates
// UpdateState = modify and send updates
// Freeze = modify and send updates, pause some updates to query services?  Maybe just update has a bool of frozen or not?

// Firehose = get current whole state, subscribe to updates

// Get Ticket = Read current state.
