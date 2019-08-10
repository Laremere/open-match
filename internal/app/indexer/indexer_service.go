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

package indexer

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/pkg/pb"

	"open-match.dev/open-match/internal/statestore"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.mmlogic",
	})
)

// The MMLogic API provides utility functions for common MMF functionality such
// as retreiving Tickets from state storage.
type indexerService struct {
	cfg           config.View
	fetcher       *fetcher
	nextWatermark chan int64
}

func newIndexerService(cfg config.View) *indexerService {
	s := &indexerService{
		cfg:           cfg,
		fetcher:       newFetcher(cfg),
		nextWatermark: make(chan int64, 1),
	}
	s.nextWatermark <- 0

	return s
}

func (s *indexerService) nextWatermark() int64 {
	n := <-nextWatermark
	nextWatermark <- n + 1
	return n
}

// GetPoolTickets gets the list of Tickets that match every Filter in the
// specified Pool.
func (s *indexerService) QueryTickets(req *pb.QueryTicketsRequest, responseServer pb.MmLogic_QueryTicketsServer) error {
	if req.GetPool() == nil {
		return status.Error(codes.InvalidArgument, ".pool is required")
	}

	ctx := responseServer.Context()
	pSize := getPageSize(s.cfg)

	_ = ctx
	_ = pSize

	// query index
	// filter out ignored
	// run other filters

	// poolFilters := req.GetPool().GetFilters()
	// callback := func(tickets []*pb.Ticket) error {
	// 	err := responseServer.Send(&pb.QueryTicketsResponse{Tickets: tickets})
	// 	if err != nil {
	// 		logger.WithError(err).Error("Failed to send Redis response to grpc server")
	// 		return status.Errorf(codes.Aborted, err.Error())
	// 	}
	// 	return nil
	// }

	// return doQueryTickets(ctx, poolFilters, pSize, callback, s.store)
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

type fetcher struct {
	store statestore.Service
	reqs  chan int64
}

// Updates works like a linked list of additions and removals.  This allows
// different consumers to branch.  A branch is performed by querying the relvant
// tickets from the branch source, then returning those tickets and the current
// waiting updates.  From this point the consumers can process the linked list
// of updates at any independant rate.  As the branch source and new consumer
// are created at the same lockstep of updates, and both will proceed to update
// themselves from that point, they are garunteed to contain all additions and
// removals up to that point, and won't miss any future changes either.
type updates struct {
	ready chan struct{}

	added     []*pb.Ticket
	removed   []*pb.Ticket
	err       error
	watermark int64
	next      *updates
}

func newFetcherAndPool(cfg config.View) (*fetcher, func() *updates) {
	f = &fetcher{
		store: statestore.New(cfg),
		reqs:  make(chan int64, 50),
	}

	firstUpdate := &updates{
		ready: make(chan struct{}),
	}

	go f.run(firstUpdate)

	requestPool := make(chan chan *updates)

	go pool(requestPool, firstUpdate)

	branchFromPool := func() *updates {
		c := make(chan *updates)
		requestPool <- c
		return <-c
	}

	return f, branchFromPool
}

func (f *fetcher) HealthCheck() error {
	return f.store.HealthCheck()
}

// TODO: This should be the source of the watermarks.  It can, using a mutex,
// see if there is already the next request queued.  If it hasn't been yet, it
// can retrieve that next watermark, and use sync.Cond to kick the go routine
// back into running.
func (f *fetcher) updateTo(watermark int64) {
	f.reqs <- watermark
}

func (f *fetcher) run(u *updates) {
	watermark := int64(-1)
	for req := range f.reqs {
		if req > watermark {
			watermark = req
		} else {
			continue
		}

	loop:
		for {
			select {
			case req2 := <-f.reqs:
				if req2 > watermark {
					watermark = req2
				}
			default:
				break loop
			}

			populateUpdate(u)
			u.watermark = watermark
			u.next = &updates{
				ready: make(chan struct{}),
			}
			close(u.ready)
			u = u.next
		}
	}
}

func (f *fetcher) populateUpdate(u *updates) *updates {
}

// TODO: maybe just return a list and updates combo, because then this could be
// used for the root pool queries as well?
func pool(reqs chan chan *updates, u *updates) {
	tickets := make(map[string]*pb.Ticket)

	closed := make(chan struct{})
	close(closed)

	watermark := -1
	var err error

	for {
		select {
		case <-u.ready:
			for _, t := range u.added {
				tickets[t.Id] = t
			}
			for _, t := range u.removed {
				delete(tickets, t.Id)
			}
			watermark = u.watermark
			err = u.err
			u = u.next()
		case req := <-reqs:
			resp := &updates{
				ready:     closed,
				next:      u,
				watermark: watermark,
				err:       err,
			}
			resp.added = make([]*pb.Ticket, 0, len(tickets))
			for _, t := range tickets {
				resp.added = append(resp.added, k)
			}
			req <- resp
		}
	}
}
