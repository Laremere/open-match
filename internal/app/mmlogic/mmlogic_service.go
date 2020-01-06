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
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/app/store/storeclient"
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
	cfg               config.View
	store             ipb.StoreClient
	watermarkRequests chan chan error
}

func newMmlogicService(cfg config.View) *mmlogicService {
	s := &mmlogicService{
		cfg:               cfg,
		store:             storeclient.FromCfg(cfg),
		watermarkRequests: make(chan chan error),
	}

	firehoseWatermarks := make(chan uint64, 1)

	go runFirehose(firehoseWatermarks, s.store)

	go runWatermarker(firehoseWatermarks, watermarkRequests, s.store)

	return s
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

	watermark := make(chan error, 1)
	select {
	case s.watermarkRequests <- watermark:
	case <-ctx.Done():
		return ctx.Err()
	}
	err := <-watermark
	if err != nil {
		return err // TODO: Error wrapping?
	}

	tickets := s.ts.query(pool)

	for start := 0; start < len(tickets); start += pSize {
		end := start + pSize
		if end > len(tickets) {
			end = len(tickets)
		}

		err := responseServer.Send(&pb.QueryTicketsResponse{
			Tickets: tickets[start:end],
		})
		if err != nil {
			return err
		}
	}

	// callback := func(tickets []*pb.Ticket) error {
	// 	err := responseServer.Send(&pb.QueryTicketsResponse{Tickets: tickets})
	// 	if err != nil {
	// 		logger.WithError(err).Error("Failed to send Redis response to grpc server")
	// 		return status.Errorf(codes.Aborted, err.Error())
	// 	}
	// 	return nil
	// }

	// return doQueryTickets(ctx, pool, pSize, callback, s.store)
}

// func doQueryTickets(ctx context.Context, pool *pb.Pool, pageSize int, sender func(tickets []*pb.Ticket) error, store ipb.StoreClient) error {
// 	// Send requests to the storage service
// 	// err := store.FilterTickets(ctx, pool, pageSize, sender)
// 	// if err != nil {
// 	// 	logger.WithError(err).Error("Failed to retrieve result from storage service.")
// 	// 	return err
// 	// }

// 	return nil
// }

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
	err                       error
}

func newTicketStash(store ipb.StoreClient) *ticketStash {
	return &ticketStash{} // Values set by reset on first update.
}

func (ts *ticketStash) query(pool *pb.Pool) ([]*pb.Ticket, error) {
	ts.lock.RLock()
	defer ts.lock.RUnlock()

	if ts.err != nil {
		return nil, ts.err
	}

	var results []*pb.Ticket

	for _, ticket := range ts.listed {
		if filter.InPool(ticket, pool) {
			results = append(results, ticket)
		}
	}

	// TODO: offer the ability to query pending and assigned tickets, for error
	// recovery.

	return results, nil
}

func (ts *ticketStash) update(reset bool, firehoses []*ipb.FirehoseResponse, err error) {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	ts.err = err

	if reset {
		ts.listed = make(map[string]*pb.Ticket)
		ts.pending = make(map[string]*pb.Ticket)
		ts.assigned = make(map[string]*pb.Ticket)
	}

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

		case nil:
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func runWatermarker(firehoseWatermarks chan uint64, reqs chan chan error, store ipb.StoreClient) {
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
				ctx := context.WithTimeout(context.Background(), time.Second)
				resp, err := s.store.GetCurrentWatermark(ctx, &ipb.GetCurrentWatermarkRequest{})
				if err != nil {
					getError <- err
				} else {
					getResp <- resp.Watermark
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

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func runFirehose(ts *ticketStash, firehoseWatermarks chan uint64, store ipb.StoreClient) {
	for {
		err := firehoseIteration(firehoseWatermarks, store)
		ts.update(true, nil, err)
		// TODO: log error
		time.Sleep(time.Second)
	}
}

func firehoseIteration(ts *ticketStash, firehoseWatermarks chan uint64, store ipb.StoreClient) error {
	c, err := store.Firehose(ctx.Background(), &ipb.FirehoseRequest{})
	if err != nil {
		return err
	}

	{
		initialUpdates := []*ipb.FirehoseResponse{}
		var last *ipb.FirehoseResponse

		for last == nil || last.Watermark == 0 {
			last, err = c.Recv()
			if err != nil {
				return err
			}
			initialUpdates = append(initialUpdates, last)
		}

		ts.update(true, initialUpdates, err)
		select {
		case firehoseWatermarks <- last.Watermark:
		case <-firehoseWatermarks:
			firehoseWatermarks <- last.Watermark
		}
	}

	updates := make(chan *ipb.FirehoseResponse, 10000)
	updateDone := make(chan struct{})

	go func() {
		// TODO: SEND UPDATES to ts
	}()

	for {
		resp, err := c.Recv()
		if err != nil {
			// TODO: LOg
			break
		}
		updates <- resp
	}

	close(updates)
	<-updateDone
}
