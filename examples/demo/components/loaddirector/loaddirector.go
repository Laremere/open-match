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

package loaddirector

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"open-match.dev/open-match/examples/demo/components"
	"open-match.dev/open-match/examples/demo/updater"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

func Run(ds *components.DemoShared) {
	u := updater.NewNested(ds.Ctx, ds.Update)

	matches := make(chan *pb.Match, 50)

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("assigner_%d", i)
		go func() {
			for !isContextDone(ds.Ctx) {
				runAssigner(ds.Ctx, ds.Cfg, u.ForField(name), matches)
			}
		}()
	}

	for !isContextDone(ds.Ctx) {
		runMatcher(ds.Ctx, ds.Cfg, u.ForField("matcher"), matches)
	}
}

func isContextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

type status struct {
	Status string
}

func runMatcher(ctx context.Context, cfg config.View, update updater.SetFunc, matches chan *pb.Match) {
	defer func() {
		r := recover()
		if r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}

			update(status{Status: fmt.Sprintf("Encountered error: %s", err.Error())})
			time.Sleep(time.Second * 10)
		}
	}()

	s := status{}

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Running"
	update(s)

	conn, err := rpc.GRPCClientFromConfig(cfg, "api.backend")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	be := pb.NewBackendClient(conn)

	//////////////////////////////////////////////////////////////////////////////
	{
		req := &pb.FetchMatchesRequest{
			Config: &pb.FunctionConfig{
				Host: cfg.GetString("api.functions.hostname"),
				Port: int32(cfg.GetInt("api.functions.grpcport")),
				Type: pb.FunctionConfig_GRPC,
			},
			Profiles: []*pb.MatchProfile{
				{
					Name: "1v1",
					Pools: []*pb.Pool{
						{
							Name: "Everyone",
							Filters: []*pb.Filter{
								{
									Attribute: "mode.demo",
									Min:       -100,
									Max:       100,
								},
							},
						},
					},
				},
			},
		}

		resp, err := be.FetchMatches(ctx, req)
		if err != nil {
			panic(err)
		}

		for _, match := range resp.Matches {
			matches <- match
		}
	}
}

func runAssigner(ctx context.Context, cfg config.View, update updater.SetFunc, matches chan *pb.Match) {
	defer func() {
		r := recover()
		if r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}

			update(status{Status: fmt.Sprintf("Encountered error: %s", err.Error())})
			time.Sleep(time.Second * 10)
		}
	}()

	s := status{}

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Running"
	update(s)

	conn, err := rpc.GRPCClientFromConfig(cfg, "api.backend")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	be := pb.NewBackendClient(conn)

	//////////////////////////////////////////////////////////////////////////////

	for match := range matches {
		ids := []string{}

		for _, t := range match.Tickets {
			ids = append(ids, t.Id)
		}

		req := &pb.AssignTicketsRequest{
			TicketIds: ids,
			Assignment: &pb.Assignment{
				Connection: fmt.Sprintf("%d.%d.%d.%d:2222", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)),
			},
		}

		resp, err := be.AssignTickets(ctx, req)
		if err != nil {
			panic(err)
		}

		_ = resp
	}
}
