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
	cfg           config.View
	store         ipb.StoreClient
	queryRequests chan *queryRequest
	stashUpdates  chan stashUpdate
}

func newMmlogicService(cfg config.View) *mmlogicService {
	s := &mmlogicService{
		cfg:           cfg,
		store:         storeclient.FromCfg(cfg),
		queryRequests: make(chan *queryRequest),
		stashUpdates:  make(chan stashUpdate, 1000),
	}

	go s.runQueryLoop()
	go s.runFirehoseLoop()

	return s
}

type queryRequest struct {
	ctx  context.Context
	resp chan *queryResponse
}

type queryResponse struct {
	err error
	wg  *sync.WaitGroup
	ts  *ticketStash
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

	qr := &queryRequest{
		ctx:  ctx,
		resp: make(chan *queryResponse),
	}
	s.queryRequests <- qr

	var qresp *queryResponse

	select {
	case <-ctx.Done():
		return ctx.Err()
	case qresp <- wr.resp:
	}

	if qresp.err != nil {
		return qresp.err
	}

	tickets, err := qresp.ts.query(pool)
	qresp.wg.Done()

	if err != nil {
		return err
	}

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
	listed, pending, assigned map[string]*pb.Ticket
	err                       error
	watermark                 uint64
}

func newTicketStash(store ipb.StoreClient) *ticketStash {
	return &ticketStash{} // Values set by reset on first update.
}

func (ts *ticketStash) query(pool *pb.Pool) ([]*pb.Ticket, error) {
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

type stashUpdate func(*ticketStash)

func firehoseUpdate(firehose *ipb.FirehoseResponse) stashUpate {
	return func(ts *ticketStash) {
		ts.watermark = f.Watermark

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

		default:
			panic("Unknown Update oneof value.")
		}
	}
}

func resetAllUpdate(ts *ticketStash) {
	ts.listed = make(map[string]*pb.Ticket)
	ts.pending = make(map[string]*pb.Ticket)
	ts.assigned = make(map[string]*pb.Ticket)
	ts.err = nil
	ts.watermark = 0
}

func setErrorUpdate(err error) stashUpdate {
	return func(ts *ticketStash) {
		resetAllUpdate(ts)
		ts.err = err
	}
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (s *mmlogicService) runQueryLoop() {
	ts := &ticketStash{}

outerLoop:
	for {
		// Wait for first query, processing updates while doing so.
		var reqs []*queryRequests
		for len(reqs) == 0 {
			select {
			case u := <-stashUpdates:
				u(ts)
			case first := <-s.queryRequests:
				reqs = append(reqs, first)
			}
		}

		// Collect all waiting querries.
	collectAllWaiting:
		for {
			select {
			case req := s.queryRequests:
				reqs = append(reqs, req)
			default:
				break collectAllWaiting
			}
		}

		// Get watermark and process updates until watermark is reached
		{
			getErr := make(chan error)
			getResp := make(chan uint64)

			go func() {
				ctx := context.WithTimeout(context.Background(), time.Second)
				resp, err := s.store.GetCurrentWatermark(ctx, &ipb.GetCurrentWatermarkRequest{})
				if err == nil {
					getResp <- resp.Watermark
				} else {
					getErr <- err
				}
			}()

			desiredWatermark := math.MaxUint64
			for desiredWatermark > ts.watermark {
				select {
				case desiredWatermark = <-getResp:
				case err := <-getErr:
					resp := &queryResp{
						err: err,
					}

					for _, req := range reqs {
						select {
						case req.resp <- resp:
						case <-req.ctx.Done():
						}
					}

					continue outerLoop
				case u := <-stashUpdates:
					u(ts)
				}
			}

		}

		wg := &sync.WaitGroup{}

		// Send ticket stash to query calls.
		for _, req := range reqs {
			resp := &queryResp{
				wg: wg,
				ts: ts,
			}

			for _, req := range reqs {
				select {
				case req.resp <- resp:
					wg.Add(1)
				case <-req.ctx.Done():
				}
			}
		}

		// wait for query calls to finish using ticket stash.
		wg.Wait()
	}
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (s *mmlogicService) runFirehoseLoop() {
	for {
		err := firehoseIteration()
		s.stashUpdates <- setErrorUpdate(err)
		// TODO: log error
		time.Sleep(time.Second)
	}
}

func (s *mmlogicService) firehoseIteration(ts *ticketStash, firehoseWatermarks chan uint64, store ipb.StoreClient) error {
	c, err := store.Firehose(ctx.Background(), &ipb.FirehoseRequest{})
	if err != nil {
		return err
	}

	s.stashUpdates <- resetAllUpdate

	for {
		firehose, err := c.Recv()
		if err != nil {
			return fmt.Errorf("Error from firehose recieve: %w", err)
		}

		s.stashUpdates <- firehoseUpdate(firehose)
	}
}
