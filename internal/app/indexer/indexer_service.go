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
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/filter"
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
	s.nextWatermark <- 1

	return s
}

func (s *indexerService) GetWatermark() int64 {
	n := <-s.nextWatermark
	s.nextWatermark <- n + 1
	return n
}

// GetPoolTickets gets the list of Tickets that match every Filter in the
// specified Pool.
func (s *indexerService) QueryTickets(req *pb.QueryTicketsRequest, responseServer pb.MmLogic_QueryTicketsServer) error {
	// panic("nope")
	logger.Info("Query request recieved")
	if req.GetPool() == nil {
		return status.Error(codes.InvalidArgument, ".pool is required")
	}

	ctx := responseServer.Context()

	watermark := s.GetWatermark()
	s.fetcher.updateTo(watermark)

	fmt.Println("IN QUERY, ASSIGNED WATERMARK IS ", watermark)

	u := s.fetcher.GetUpdates()
	tickets := make(map[string]*pb.Ticket)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-u.ready:
		}

		fmt.Println("IN QUERY, RECIEVED WATERMARK IS ", u.watermark)
		if u.err != nil && u.watermark >= watermark {
			// If the query is currently not caught up, and also returned an error,
			// the indexer is in an error state and we should return that.
			return u.err
		}

		for _, t := range filter.Filter(u.added, req.Pool.Filters) {
			tickets[t.Id] = t
		}
		for _, t := range u.removed {
			delete(tickets, t.Id)
		}

		if u.watermark >= watermark {
			break
		}
		u = u.next
	}

	pSize := getPageSize(s.cfg)
	asSlice := make([]*pb.Ticket, 0, len(tickets))
	for _, t := range tickets {
		asSlice = append(asSlice, t)
	}

	for i := 0; i < len(tickets); i += pSize {
		end := i + pSize
		if end > len(tickets) {
			end = len(tickets)
		}
		page := asSlice[i:end]
		err := responseServer.Send(&pb.QueryTicketsResponse{Tickets: page})
		if err != nil {
			logger.WithError(err).Error("Failed to send indexer response to grpc server")
			return status.Errorf(codes.Aborted, err.Error())
		}
	}
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
	store           statestore.Service
	reqs            chan int64
	requestPool     chan chan *updates
	watermark       chan int64
	watermarkPooled chan int64
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

func newFetcher(cfg config.View) *fetcher {
	f := &fetcher{
		store:           statestore.New(cfg),
		reqs:            make(chan int64, 50),
		requestPool:     make(chan chan *updates),
		watermark:       make(chan int64),
		watermarkPooled: make(chan int64),
	}

	firstUpdate := &updates{
		ready: make(chan struct{}),
	}

	go f.run(firstUpdate)
	go f.pool(f.requestPool, firstUpdate)

	go func() {
		for {
			w := <-f.watermark

		loop:
			for {
				select {
				case w2 := <-f.watermark:
					if w2 > w {
						w = w2
					}
				case f.watermarkPooled <- w:
					break loop
				}
			}
		}
	}()

	return f
}

func (f *fetcher) HealthCheck(ctx context.Context) error {
	// TODO: Check current update value for errors.
	return f.store.HealthCheck(ctx)
}

func (f *fetcher) GetUpdates() *updates {
	c := make(chan *updates)
	f.requestPool <- c
	return <-c
}

// TODO: This should be the source of the watermarks.  It can, using a mutex,
// see if there is already the next request queued.  If it hasn't been yet, it
// can retrieve that next watermark, and use sync.Cond to kick the go routine
// back into running.
func (f *fetcher) updateTo(watermark int64) {
	f.watermark <- watermark
}

func (f *fetcher) run(u *updates) {
	current := make(map[string]*pb.Ticket)

	for watermark := range f.watermarkPooled {
		fmt.Println("IN RUN, RUNNING REQUEST FOR WATERMARK ", watermark)
		f.populateUpdate(u, current)
		u.watermark = watermark
		u.next = &updates{
			ready: make(chan struct{}),
		}
		fmt.Println("IN RUN, SENDING UPDATE FOR WATERMARK ", watermark)
		close(u.ready)
		u = u.next
	}
}

func (f *fetcher) populateUpdate(u *updates, current map[string]*pb.Ticket) {
	ctx := context.Background()

	indexed, ignored, err := f.store.GetCurrentTickets(ctx)
	if err != nil {
		u.err = err // TODO: Wrap
		return
	}
	fmt.Println("IN POPULATE UPDATE, INDEXED = ", indexed)
	fmt.Println("IN POPULATE UPDATE, IGNORED = ", ignored)

	ignoredSet := make(map[string]struct{})
	for _, id := range ignored {
		ignoredSet[id] = struct{}{}
	}

	// Maybe worth it to keep around references to parsed tickets in ignoredSet,
	// but it seems like a rare use case for the complexity.
	var toFetch []string

	indexedSet := make(map[string]struct{})
	for _, id := range indexed {
		if _, ok := ignoredSet[id]; !ok {
			indexedSet[id] = struct{}{}
			if _, ok2 := current[id]; !ok2 {
				toFetch = append(toFetch, id)
			}
		}
	}

	for id, t := range current {
		if _, ok := indexedSet[id]; !ok {
			delete(current, id)
			u.removed = append(u.removed, t)
		}
	}

	newTickets, err := f.store.GetTickets(ctx, toFetch)
	if err != nil {
		u.err = err // TODO: Wrap
		return
	}

	for _, t := range newTickets {
		current[t.Id] = t
		u.added = append(u.added, t)
	}
}

// TODO: maybe just return a list and updates combo, because then this could be
// used for the root pool queries as well?
func (f *fetcher) pool(reqs chan chan *updates, u *updates) {
	tickets := make(map[string]*pb.Ticket)

	closed := make(chan struct{})
	close(closed)

	watermark := int64(-1)
	var err error

	for {
		select {
		case <-u.ready:
			fmt.Println("IN POOL, READY TO READ WATERMARK", u.watermark)
			for _, t := range u.added {
				tickets[t.Id] = t
			}
			for _, t := range u.removed {
				delete(tickets, t.Id)
			}
			watermark = u.watermark
			err = u.err
			u = u.next
		case req := <-reqs:
			fmt.Println("IN POOL, SENDING WATERMARK", watermark)
			resp := &updates{
				ready:     closed,
				next:      u,
				watermark: watermark,
				err:       err,
			}
			// TODO: This slice can be cached.
			resp.added = make([]*pb.Ticket, 0, len(tickets))
			for _, t := range tickets {
				resp.added = append(resp.added, t)
			}
			req <- resp
		}
	}
}
