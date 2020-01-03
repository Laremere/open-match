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

package mmlogic

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/filter"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.mmlogic",
	})
)

// The MMLogic API provides utility functions for common MMF functionality such
// as retreiving Tickets from state storage.
type mmlogicService struct {
	cfg   config.View
	store ipb.StoreClient
}

// QueryTickets gets a list of Tickets that match all Filters of the input Pool.
//   - If the Pool contains no Filters, QueryTickets will return all Tickets in the state storage.
// QueryTickets pages the Tickets by `storage.pool.size` and stream back response.
//   - storage.pool.size is default to 1000 if not set, and has a mininum of 10 and maximum of 10000
func (s *mmlogicService) QueryTickets(req *pb.QueryTicketsRequest, responseServer pb.MmLogic_QueryTicketsServer) error {
	pool := req.GetPool()
	if pool == nil {
		return status.Error(codes.InvalidArgument, ".pool is required")
	}

	ctx := responseServer.Context()
	pSize := getPageSize(s.cfg)

	callback := func(tickets []*pb.Ticket) error {
		err := responseServer.Send(&pb.QueryTicketsResponse{Tickets: tickets})
		if err != nil {
			logger.WithError(err).Error("Failed to send Redis response to grpc server")
			return status.Errorf(codes.Aborted, err.Error())
		}
		return nil
	}

	return doQueryTickets(ctx, pool, pSize, callback, s.store)
}

func doQueryTickets(ctx context.Context, pool *pb.Pool, pageSize int, sender func(tickets []*pb.Ticket) error, store ipb.StoreClient) error {
	// Send requests to the storage service
	// err := store.FilterTickets(ctx, pool, pageSize, sender)
	// if err != nil {
	// 	logger.WithError(err).Error("Failed to retrieve result from storage service.")
	// 	return err
	// }

	return nil
}

func getPageSize(cfg config.View) int {
	const (
		name = "storage.page.size"
		// Minimum number of tickets to be returned in a streamed response for QueryTickets. This value
		// will be used if page size is configured lower than the minimum value.
		minPageSize int = 10
		// Default number of tickets to be returned in a streamed response for QueryTickets.  This value
		// will be used if page size is not configured.
		defaultPageSize int = 1000
		// Maximum number of tickets to be returned in a streamed response for QueryTickets. This value
		// will be used if page size is configured higher than the maximum value.
		maxPageSize int = 10000
	)

	if !cfg.IsSet(name) {
		return defaultPageSize
	}

	pSize := cfg.GetInt("storage.page.size")
	if pSize < minPageSize {
		logger.Infof("page size %v is lower than the minimum limit of %v", pSize, maxPageSize)
		pSize = minPageSize
	}

	if pSize > maxPageSize {
		logger.Infof("page size %v is higher than the maximum limit of %v", pSize, maxPageSize)
		return maxPageSize
	}

	return pSize
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

type ticketStash struct {
	lock                      sync.RWMutex
	listed, pending, assigned map[string]*pb.Ticket
}

func newTicketStash(store ipb.StoreClient) *ticketStash {
	return &ticketStash{
		listed:   make(map[string]*pb.Ticket),
		pending:  make(map[string]*pb.Ticket),
		assigned: make(map[string]*pb.Ticket),
	}
}

func (ts *ticketStash) query(watermark uint64, pool *pb.Pool) []*pb.Ticket {
	ts.lock.RLock()
	defer ts.lock.RUnlock()

	var results []*pb.Ticket

	for _, ticket := range ts.listed {
		if filter.InPool(ticket, pool) {
			results = append(results, ticket)
		}
	}

	// TODO: offer the ability to query pending and assigned tickets, for error
	// recovery.

	return results
}

func (ts *ticketStash) update(firehoses []*ipb.FirehoseResponse) {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	for _, f := range firehoses {
		switch f := f.Update.(type) {
		case *ipb.FirehoseResponse_NewTicket:
			ts.listed[f.NewTicket.Id] = f.NewTicket
		case *ipb.FirehoseResponse_RelistedId:
			ts.listed[f.RelistedId] = ts.pending[f.RelistedId]
			delete(ts.pending, f.RelistedId)
		case *ipb.FirehoseResponse_PendingId:
			ts.pending[f.PendingId] = ts.listed[f.PendingId]
			delete(ts.listed, f.PendingId)
		case *ipb.FirehoseResponse_AssignedId:
			t, ok := ts.pending[f.AssignedId]
			if ok {
				delete(ts.pending, f.AssignedId)
			} else {
				t = ts.listed[f.AssignedId]
				delete(ts.listed, f.AssignedId)
			}
			ts.assigned[f.AssignedId] = t

		case *ipb.FirehoseResponse_DeletedId:
			delete(ts.listed, f.DeletedId)
			delete(ts.pending, f.DeletedId)
			delete(ts.assigned, f.DeletedId)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (s *mmlogicService) getWatermark() (uint64, error) {
	resp, err := s.store.GetCurrentWatermark(context.Background(), &ipb.GetCurrentWatermarkRequest{})
	if err != nil {
		return 0, err
	}
	return resp.Watermark, nil
}

func RunWatermarker(firehoseWatermarks chan uint64, reqs chan chan error, getCurrentWatermark func() (uint64, error)) {
	type waiting struct {
		watermark uint64
		req       chan error
		next      *waiting
	}

	getError := make(chan error)
	getResp := make(chan uint64)

	var waitingToGet []chan error
	var outgoing []chan struct{}
	var next *waiting
	var last *waiting
	var firehoseWatermark uint64

	for {
		for next != nil && next.watermark <= firehoseWatermark {
			next.req <- nil
			next = next.next
		}
		if len(outgoing) == 0 && len(waitingToGet) > 0 {
			outgoing = waitingToGet
			waitingToGet = nil
			go func() {
				w, err := getCurrentWatermark()
				if err != nil {
					getError <- err
				} else {
					getResp <- w
				}
			}()
		}

		select {
		case firehoseWatermark = <-firehoseWatermarks:
		case req := <-reqs:
			waitingToGet = append(waitingToGet, req)
		case waitingFor := <-getResp:
			for _, req := range outgoing {
				w := &waiting{
					watermark: waitingFor,
					req:       req,
					next:      nil,
				}
				if next == nil {
					next = w
				} else {
					last.next = w
				}
				last = w
			}
			outgoing = nil
		case err := <-getError:
			for _, req := range outgoing {
				req <- err
			}
			outgoing = nil
		}
	}
}
