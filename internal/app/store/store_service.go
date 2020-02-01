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
	"open-match.dev/open-match/internal/telemetry"

	// "open-match.dev/open-match/internal/rpc"
	// "open-match.dev/open-match/internal/statestore"
	// "open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

var (
	mTickets = telemetry.Gauge("store_tickets", "tickets")
)

type storeService struct {
	lock sync.RWMutex

	tickets map[string]*ticketState
	//	frozen  bool

	watermark     uint64
	pendingUpdate *update
}

func newStoreService() *storeService {
	return &storeService{
		tickets: map[string]*ticketState{},
		//		frozen:  false,

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

	telemetry.SetGauge(ctx, mTickets, int64(len(s.tickets)))

	return &ipb.CreateTicketResponse{}, nil
}

func (s *storeService) Firehose(req *ipb.FirehoseRequest, stream ipb.Store_FirehoseServer) error {
	initialUpdates, pending := s.startFirehose()

	for _, u := range initialUpdates {
		err := stream.Send(u)
		if err != nil {
			return err
		}
	}
	initialUpdates = nil // Free memory

	for {
		<-pending.ready

		err := stream.Send(pending.firehose)
		if err != nil {
			return err
		}

		pending = pending.next
	}

	return nil
}

func (s *storeService) startFirehose() ([]*ipb.FirehoseResponse, *update) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// TODO: this needs to be saved at the start of a freeze for any new firehose during freeze?
	// or otherwise wait on firehose until after freeze is finished.
	initialUpdates := []*ipb.FirehoseResponse{}
	// TODO: Freeze?
	for id, ts := range s.tickets {
		initialUpdates = append(initialUpdates, &ipb.FirehoseResponse{
			Update: &ipb.FirehoseResponse_NewTicket{ts.ticket},
		})

		if ts.assignment != nil {
			initialUpdates = append(initialUpdates, &ipb.FirehoseResponse{
				Update: &ipb.FirehoseResponse_AssignedId{id},
			})
		} else if ts.pending {
			initialUpdates = append(initialUpdates, &ipb.FirehoseResponse{
				Update: &ipb.FirehoseResponse_PendingId{id},
			})
		}
	}

	initialUpdates = append(initialUpdates, &ipb.FirehoseResponse{
		Watermark: s.watermark,
	})

	// Possible improvement: This could be cached for a short time.
	return initialUpdates, s.pendingUpdate
}

// func (s *storeService) Freeze(ctx context.Context, req *ipb.FreezeRequest) (*ipb.FreezeResponse, error) {
// 	s.lock.Lock()
// 	defer s.lock.Unlock()

// 	if req.Freeze == s.frozen {
// 		return &ipb.FreezeResponse{}, nil
// 	}

// 	// TODO:Send update, save to disk.
// 	return nil, nil
// }

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
	if req.Assignment == nil {
		return nil, status.Errorf(codes.InvalidArgument, "assignment required")
	}
	if len(req.Ids) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "ids required")
	}

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

	return &ipb.AssignTicketsResponse{}, nil
}

func (s *storeService) ReleaseTickets(ctx context.Context, req *ipb.ReleaseTicketsRequest) (*ipb.ReleaseTicketsResponse, error) {
	if len(req.Ids) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "ids required")
	}

	s.lock()
	defer s.lock.Unlock()

	for _, id := range req.Ids {
		ts, ok := s.tickets[id]
		if !ok || !ts.pending || ts.assignment != nil {
			continue
		}

		ts.pending = false

		s.pendingUpdate.firehose.Update = &ipb.FirehoseResponse_ReleasedId{id}
		s.releaseUpdate()
	}

	return &ipb.ReleaseTicketsRequest{}, nil
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

	if len(req.Ids) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "ids required")
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	for _, id := range req.Ids {
		ts, ok := s.tickets[id]
		if !ok || ts.pending || ts.assignment != nil {
			// TODO: return list of failed pendings?
			continue
		}

		ts.pending = true

		s.pendingUpdate.firehose.Update = &ipb.FirehoseResponse_PendingId{id}
		s.releaseUpdate()
	}

	return &ipb.MarkPendingResponse{}, nil
}

func (s *storeService) DeleteTicket(ctx context.Context, req *ipb.DeleteTicketRequest) (*ipb.DeleteTicketResponse, error) {
	if req.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id required")
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	_, ok := s.tickets[req.Id]
	if ok {
		delete(s.tickets, req.Id)
		s.pendingUpdate.firehose.Update = &ipb.FirehoseResponse_DeletedId{req.Id}
		s.releaseUpdate()
		telemetry.SetGauge(ctx, mTickets, int64(len(s.tickets)))
	}

	return &ipb.DeleteTicketResponse{}, nil
}
