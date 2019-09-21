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

package clients

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"open-match.dev/open-match/examples/demo/components"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

func Run(ds *components.DemoShared) {
	number := 0
	ticker := time.NewTicker(time.Second)
	ticketDone := make(chan error)

	s := struct {
		TicketsSearching     int
		TotalTicketsAssigned int
		TotalTicketsErrored  int
		LastError            string
	}{}

loop:
	for {
		select {
		case <-ticker.C:
			for i := 0; i < 5; i++ {
				name := fmt.Sprintf("fakeplayer_%d", number)
				number++
				go runScenario(ds.Ctx, ds.Cfg, name, ticketDone)
				s.TicketsSearching++
			}
			ds.Update(s)
		case <-ds.Ctx.Done():
			break loop
		case err := <-ticketDone:
			s.TicketsSearching--
			if err == nil {
				s.TotalTicketsAssigned++
			} else {
				s.TotalTicketsErrored++
				s.LastError = err.Error()
			}
		}
	}
	ticker.Stop()
}

func isContextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func runScenario(ctx context.Context, cfg config.View, name string, ticketDone chan error) {
	defer func() {
		var err error
		r := recover()
		if r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}
		}
		select {
		case ticketDone <- err:
		case <-ctx.Done():
		}
	}()

	//////////////////////////////////////////////////////////////////////////////
	conn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fe := pb.NewFrontendClient(conn)

	//////////////////////////////////////////////////////////////////////////////
	var ticketId string
	{
		p := structs.Struct{
			"name":        structs.String(name),
			"mmr":         structs.Number(randRank()),
			"searchStart": structs.Number(float64(time.Now().Unix())),
		}

		if rand.Float64() < 0.7 {
			p["mode_deathmatch"] = structs.Bool(true)
		} else {
			p["mode_ctf"] = structs.Bool(true)
		}

		for _, region := range randRegions() {
			p[region] = structs.Bool(true)
		}

		req := &pb.CreateTicketRequest{
			Ticket: &pb.Ticket{
				Properties: p.S(),
			},
		}

		resp, err := fe.CreateTicket(ctx, req)
		if err != nil {
			panic(err)
		}
		ticketId = resp.Ticket.Id
	}

	//////////////////////////////////////////////////////////////////////////////
	var assignment *pb.Assignment
	{
		req := &pb.GetAssignmentsRequest{
			TicketId: ticketId,
		}

		stream, err := fe.GetAssignments(ctx, req)
		for assignment.GetConnection() == "" {
			resp, err := stream.Recv()
			if err != nil {
				// For now we don't expect to get EOF, so that's still an error worthy of panic.
				panic(err)
			}

			assignment = resp.Assignment
		}

		err = stream.CloseSend()
		if err != nil {
			panic(err)
		}
	}

	//////////////////////////////////////////////////////////////////////////////
	// TODO: delete ticket?
}

func randRank() float64 {
	v := rand.NormFloat64()*15 + 50
	if v > 100 {
		return 100
	}
	if v < 0 {
		return 0
	}
	return v
}

var regions = []string{
	"region_us",
	"region_europe",
	"region_asia",
	// "region_us_west",
	// "region_us_east",
	// "region_korea",
	// "region_japan",
	// "region_china",
	// "region_australia",
	// "region_middle_east",
	// "region_europe_west",
	// "region_europe_east",
	// "region_brazil",
}

func randRegions() []string {
	// Better: possibilities of choosing mulitple regions, eg us-west and us-east
	// but not unrealistic options like us-west and middle-east
	return []string{regions[rand.Intn(len(regions))]}
}
