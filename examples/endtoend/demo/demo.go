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

// Package demo contains
package demo

import (
	"context"
	"fmt"
	"io"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"open-match.dev/open-match/examples/endtoend/dashboard"
	"open-match.dev/open-match/internal/pb"
)

type Demo struct {
	nextPlayerId chan int64
	updates      chan<- func(*dashboard.Dashboard)
	clients      *Clients
}

func New(updates chan func(*dashboard.Dashboard), clients *Clients) *Demo {
	d := &Demo{
		updates:      updates,
		nextPlayerId: make(chan int64),
		clients:      clients,
	}

	go d.Start()

	return d
}

func (d *Demo) Start() {
	go func() {
		for range time.Tick(time.Second) {
			d.updates <- func(dash *dashboard.Dashboard) {
				dash.Uptime++
			}
		}
	}()

	go func() {
		for id := int64(0); true; id++ {
			d.nextPlayerId <- id
		}
	}()

	go d.NewAi()
	go d.NewAi()
	go d.NewAi()
	go d.NewDirector()
}

type Id int64

func (d *Demo) NewAi() {
	id := <-d.nextPlayerId

	d.updates <- func(dash *dashboard.Dashboard) {
		dash.Players[id] = &dashboard.Player{
			Name:   fmt.Sprintf("Robo-bot_%d", id),
			Status: "Main Menu",
		}
	}

	status := func(s string) {
		d.updates <- func(dash *dashboard.Dashboard) {
			dash.Players[id].Status = s
		}
	}
	handleError := func(err error) bool {
		if err != nil {
			d.updates <- func(dash *dashboard.Dashboard) {
				dash.Players[id].Error = err.Error()
				dash.Players[id].Status = "Crashed"
			}
			return true
		}
		return false
	}

	time.Sleep(time.Second * 5)
	status("Creating ticket in open match")

	var ticketId string
	{
		req := &pb.CreateTicketRequest{
			Ticket: &pb.Ticket{
				Properties: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"id": {
							Kind: &structpb.Value_StringValue{
								StringValue: fmt.Sprintf("%d", id),
							},
						},
						"mode.demo": {
							Kind: &structpb.Value_NumberValue{
								NumberValue: 1,
							},
						},
					},
				},
			},
		}
		resp, err := d.clients.FE.CreateTicket(context.Background(), req)
		if handleError(err) {
			return
		}

		ticketId = resp.Ticket.Id
	}

	status(fmt.Sprintf("Waiting for match, open match ticket id=%s", ticketId))
}

func (d *Demo) NewDirector() {
	d.updates <- func(dash *dashboard.Dashboard) {
		dash.Director = &dashboard.Director{
			Status: "Starting",
		}
	}

	status := func(s string) {
		d.updates <- func(dash *dashboard.Dashboard) {
			dash.Director.Status = s
		}
	}

	handleError := func(err error) bool {
		if err != nil {
			d.updates <- func(dash *dashboard.Dashboard) {
				dash.Director.Error = err.Error()
				dash.Director.Status = "Crashed"
			}
			return true
		}
		return false
	}

	for {
		status("Sleeping")
		time.Sleep(time.Second * 5)
		status("Making Match: Sending Request")

		{
			req := &pb.FetchMatchesRequest{
				Config: &pb.FunctionConfig{
					Name: "Basic",
					Type: &pb.FunctionConfig_Grpc{
						Grpc: &pb.GrpcFunctionConfig{
							Host: "om-function",
							Port: 50502,
						},
					},
				},
				Profile: []*pb.MatchProfile{
					{
						Name: "Ya Basic",
						Pool: []*pb.Pool{
							{
								Name: "Everyone",
								Filter: []*pb.Filter{
									{
										Attribute: "mode.demo",
										Min:       -100,
										Max:       100,
									},
								},
							},
						},
						Roster: []*pb.Roster{
							{
								Name: "Player 1",
							},
							{
								Name: "Player 2",
							},
						},
					},
				},
			}

			stream, err := d.clients.BE.FetchMatches(context.Background(), req)
			if handleError(err) {
				return
			}
			status("Making Match: Streaming Response")
			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if handleError(err) {
					return
				}
				d.updates <- func(dash *dashboard.Dashboard) {
					dash.Director.RecentMatch = resp.String()
				}
			}
		}
	}
}

type Director struct {
}

type DGS struct {
}

func (d *DGS) allocate() string {
	return ""
}
