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

package director

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"open-match.dev/open-match/examples/demo/components"
	"open-match.dev/open-match/examples/demo/updater"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

var regions = []string{
	"region_us_west",
	"region_us_east",
	"region_korea",
	"region_japan",
	"region_china",
	"region_australia",
	"region_middle_east",
	"region_europe_west",
	"region_europe_east",
	"region_brazil",
}

var gamemodes = []string{
	"mode_deathmatch",
	"mode_ctf",
}

type rankRange struct {
	min, max float64
}

var rankRanges = []rankRange{
	{0, 35},
	{30, 45},
	{40, 60},
	{55, 70},
	{65, 100},
}

func Run(ds *components.DemoShared) {
	u := updater.NewNested(ds.Ctx, ds.Update)

	for _, region := range regions {
		for _, gamemode := range gamemodes {
			for _, rankRange := range rankRanges {
				name := fmt.Sprintf("%s-%s-%v-to-%v", region, gamemode, rankRange.min, rankRange.max)
				update := u.ForField(name)
				go continousRun(ds, update, region, gamemode, rankRange.min, rankRange.max)
			}
		}
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
	Status                 string
	TotalMatchesMade       int
	MatchesLatestIteration int
}

func continousRun(ds *components.DemoShared, update updater.SetFunc, region, gamemode string, rankMin float64, rankMax float64) {
	s := &status{}
	for !isContextDone(ds.Ctx) {
		run(ds, s, update, region, gamemode, rankMin, rankMax)
	}
}

func run(ds *components.DemoShared, s *status, update updater.SetFunc, region, gamemode string, rankMin float64, rankMax float64) {
	defer func() {
		r := recover()
		if r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}

			ds.Update(status{Status: fmt.Sprintf("Encountered error: %s", err.Error())})
			time.Sleep(time.Second * 10)
		}
	}()

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Connecting to backend"
	update(s)

	conn, err := rpc.GRPCClientFromConfig(ds.Cfg, "api.backend")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	be := pb.NewBackendClient(conn)

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Match Match: Sending Request"
	update(s)

	var matches []*pb.Match
	{
		req := &pb.FetchMatchesRequest{
			Config: &pb.FunctionConfig{
				Host: ds.Cfg.GetString("api.functions.hostname"),
				Port: int32(ds.Cfg.GetInt("api.functions.grpcport")),
				Type: pb.FunctionConfig_GRPC,
			},
			Profiles: []*pb.MatchProfile{
				{
					Name: "1v1",
					Pools: []*pb.Pool{
						{
							Name: "RegionGamemodeRank",
							FloatRangeFilters: []*pb.FloatRangeFilter{
								{
									Attribute: "mmr",
									Min:       rankMin,
									Max:       rankMax,
								},
							},
							BoolEqualsFilters: []*pb.BoolEqualsFilter{
								{
									Attribute: gamemode,
									Value:     true,
								},
								{
									Attribute: region,
									Value:     true,
								},
							},
						},
					},
				},
			},
		}

		stream, err := be.FetchMatches(ds.Ctx, req)
		if err != nil {
			panic(err)
		}

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				panic(err)
			}
			matches = append(matches, resp.GetMatch())
		}
	}

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Matches Found"
	s.MatchesLatestIteration = len(matches)
	s.TotalMatchesMade += s.MatchesLatestIteration
	update(s)

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Assigning Players"
	update(s)

	for _, match := range matches {
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

		resp, err := be.AssignTickets(ds.Ctx, req)
		if err != nil {
			panic(err)
		}

		_ = resp
	}

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Iteration Complete"
	update(s)
}
