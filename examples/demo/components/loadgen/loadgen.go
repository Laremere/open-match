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

package loadgen

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

type statsFrame struct {
	Completed    int
	Errored      int
	ErrorExample string
}

func Run(ds *components.DemoShared) {
	frames := make([]*statsFrame, 100)
	for i := range frames {
		frames[i] = &statsFrame{}
	}

	errors := make(chan string)
	done := make(chan struct{})
	finished := make(chan struct{})

	const threadCount = 500
	// const threadCount = 1000
	// const threadCount = 10000
	for i := 0; i < threadCount; i++ {
		go func() {
			for !isContextDone(ds.Ctx) {
				runScenario(ds.Ctx, ds.Cfg, errors, done)
			}
			finished <- struct{}{}
		}()
	}

	t := time.NewTicker(time.Second)
	for {
		select {
		case <-ds.Ctx.Done():
			for i := 0; i < threadCount; {
				select {
				case <-finished:
					i++
				case <-errors:
				case <-done:
				}
			}
			return

		case <-t.C:
			ds.Update(struct {
				LastSecond    *statsFrame
				Last10Seconds *statsFrame
				LastMinute    *statsFrame
				// Last100Seconds *statsFrame
			}{
				frames[0],
				frames[9],
				frames[59],
				// frames[99],
			})
			copy(frames[1:], frames[:len(frames)-1])
			frames[0] = &statsFrame{}

		case e := <-errors:
			for _, f := range frames {
				f.Errored++
				if f.ErrorExample == "" {
					f.ErrorExample = e
				}
			}
		case <-done:
			for _, f := range frames {
				f.Completed++
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

func runScenario(ctx context.Context, cfg config.View, errorChan chan string, done chan struct{}) {
	defer func() {
		r := recover()
		if r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}

			errorChan <- err.Error()
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
		l := randLatencies()

		req := &pb.CreateTicketRequest{
			Ticket: &pb.Ticket{
				Properties: structs.Struct{
					"mode.demo":           structs.Number(1),
					"mmr.rating":          randRank(),
					"region.europe-east1": l["europe-east1"],
					"region.europe-west1": l["europe-west1"],
					"region.europe-west2": l["europe-west2"],
					"region.europe-west3": l["europe-west3"],
					"region.europe-west4": l["europe-west4"],
				}.S(),
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

	done <- struct{}{}
}

func randRank() float64 {
	return clamp(-4, rand.NormFloat64(), 4)
}

func clamp(low, v, high float64) float64 {
	if v > high {
		return high
	}
	if v < low {
		return low
	}
	return v
}

// More ideally, this should better reflect real world scenarios.  eg, collect some real
// latency data, or simulate distances from a set of real world locations with rough
// populations maps to generate starting points.

var regions = []string{
	"europe-east1",
	"europe-west1",
	"europe-west2",
	"europe-west3",
	"europe-west4",
}

func randLatencies() *map[string]float64 {
	rawLatency := make(map[string]float64)

	// Latencies are random value between 0ms and 300ms
	for _, r := range regions {
		rawLatency[r] = rand.Float64() * 300
	}

	// Find the lowest latency possible.
	best := rawLatency[regions[0]]
	for _, v := range rawLatency {
		if v < best {
			best = v
		}
	}

	for region, v := range rawLatency {
		dist := v - best

	}
}
