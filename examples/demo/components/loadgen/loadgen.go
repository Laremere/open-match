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
	"time"

	"open-match.dev/open-match/examples/demo/components"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

type statsFrame struct {
	Outstanding  int
	Completed    int
	Errored      int
	ErrorExample string
}

// Two types of stats: totals for area, and status at current timestamp.

type stats struct {
}

func Run(ds *components.DemoShared) {
	s := struct {
		Outstanding    []int
		Completed      []int
		Errored        []int
		Processing     []string
		RecentErrors   []string
		TotalCompleted int
		TotalErrors    int
	}{
		Outstanding: make([]int, 10),
		Completed:   make([]int, 10),
	}

	currentOutstanding := 0
	cycleCompleted := 0
	cycleErrors := 0

	errorChan := make(chan string)

	t := time.Ticker()
	lastTime := time.Now()

	for {
		select {
		case <-ds.Ctx.Done():
			for currentOutstanding > 0 {
				select {
				case <-errorChan:
				case <-done:
				}
				currentOutstanding--
			}
			return

		case currentTime := <-t.C:

			ds.Update(s)
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

	s := status{}

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
		req := &pb.CreateTicketRequest{
			Ticket: &pb.Ticket{
				Properties: structs.Struct{
					"mode.demo": structs.Number(1),
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
