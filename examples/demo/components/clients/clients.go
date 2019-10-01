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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"open-match.dev/open-match/examples/demo/components"
	"open-match.dev/open-match/examples/demo/updater"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

func Run(ds *components.DemoShared) {
	u := updater.NewNested(ds.Ctx, ds.Update)

	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("fakeplayer_%d", i)
		go func() {
			for !isContextDone(ds.Ctx) {
				runScenario(ds.Ctx, ds.Cfg, name, u.ForField(name))
			}
		}()
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
	Status     string
	Assignment *pb.Assignment
}

func runScenario(ctx context.Context, cfg config.View, name string, update updater.SetFunc) {
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
	s.Status = "Main Menu"
	update(s)

	time.Sleep(time.Duration(rand.Int63()) % (time.Second * 15))

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Connecting to Open Match frontend"
	update(s)

	conn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fe := pb.NewFrontendClient(conn)

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Creating Open Match Ticket"
	update(s)

	var ticketId string
	// {
	// 	req := &pb.CreateTicketRequest{
	// 		Ticket: &pb.Ticket{
	// 			Properties: structs.Struct{
	// 				"name":      structs.String(name),
	// 				"mode.demo": structs.Number(1),
	// 			}.S(),
	// 		},
	// 	}

	// 	resp, err := fe.CreateTicket(ctx, req)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	ticketId = resp.Ticket.Id
	// }

	{
		req := map[string]interface{}{
			"ticket": map[string]interface{}{
				"properties": map[string]interface{}{
					"name":      name,
					"mode.demo": 1,
				},
				"details": map[string]interface{}{
					"@type": "google.protobuf.Struct",
					"value": map[string]interface{}{
						"Struct detail": "struct detail value",
						"bool":          false,
						"null":          nil,
						"number":        1.618,
					},
				},
			},
		}

		b, err := json.Marshal(req)
		if err != nil {
			panic(err)
		}

		url := fmt.Sprintf("http://%s:%d/v1/frontend/tickets", cfg.GetString("api.frontend.hostname"), cfg.GetInt("api.frontend.httpport"))
		log.Println(url)
		httpResp, err := http.Post(url, "text/json", bytes.NewReader(b))
		if err != nil {
			panic(err)
		}

		d := json.NewDecoder(httpResp.Body)

		resp := map[string]interface{}{}
		err = d.Decode(&resp)
		if err != nil {
			panic(err)
		}
		log.Println("Request", req, "Response", resp)
		ticketId = resp["ticket"].(map[string]interface{})["id"].(string)
	}

	//////////////////////////////////////////////////////////////////////////////
	s.Status = fmt.Sprintf("Waiting match with ticket Id %s", ticketId)
	update(s)

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
	s.Status = "Sleeping (pretend this is playing a match...)"
	s.Assignment = assignment
	update(s)

	time.Sleep(time.Second * 10)
}
