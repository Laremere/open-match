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
	"math/rand"
	"sync"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"open-match.dev/open-match/examples/endtoend/dashboard"
	"open-match.dev/open-match/internal/pb"
)

type Demo struct {
	nextPlayerId chan int64
	updates      chan<- func(*dashboard.Dashboard)
	clients      *Clients
	Fakegones    *Fakegones
}

func New(updates chan func(*dashboard.Dashboard), clients *Clients) *Demo {
	d := &Demo{
		updates:      updates,
		nextPlayerId: make(chan int64),
		clients:      clients,
		Fakegones:    NewFakegones(updates),
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
	name := fmt.Sprintf("Robo-bot_%d", id)

	d.updates <- func(dash *dashboard.Dashboard) {
		dash.Players[id] = &dashboard.Player{
			Name:   name,
			Status: "Startup",
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

	for {

		status("Main Menu")
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
		addr := ""
		for addr == "" {
			time.Sleep(time.Second)
			sideLock.Lock()
			addr = sideChannel[ticketId]
			delete(sideChannel, ticketId)
			sideLock.Unlock()
		}

		status(fmt.Sprintf("Playing game on %s", addr))
		d.Fakegones.Connect(addr, name)
	}
}

var sideChannel = make(map[string]string)
var sideLock sync.Mutex

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
						Name: "1v1",
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
				addr := d.Fakegones.Allocate()
				sideLock.Lock()
				for _, ticket := range resp.Match.Ticket {
					sideChannel[ticket.Id] = addr
				}
				sideLock.Unlock()
			}
		}
	}
}

type Fakegones struct {
	newServer         chan chan string
	serverClose       chan string
	connectionRequest chan *connectionRequest
}

type connectionRequest struct {
	addr       string
	playerName string
	done       chan struct{}
}

func NewFakegones(updates chan func(*dashboard.Dashboard)) *Fakegones {
	f := &Fakegones{
		newServer:         make(chan chan string),
		serverClose:       make(chan string),
		connectionRequest: make(chan *connectionRequest),
	}

	go f.start(updates)
	return f
}

func (f *Fakegones) start(updates chan func(*dashboard.Dashboard)) {
	serverNum := 0
	servers := make(map[string]chan *connectionRequest)

	for {
		select {
		case r := <-f.newServer:
			addr := fmt.Sprintf("257.0.0.%d:2667", serverNum)
			r <- addr
			c := make(chan *connectionRequest)
			go f.startServer(addr, updates, c)
			servers[addr] = c
			serverNum++
		case r := <-f.connectionRequest:
			s, ok := servers[r.addr]
			if ok {
				s <- r
			} else {
				r.done <- struct{}{}
			}
		case addr := <-f.serverClose:
			delete(servers, addr)
		}
	}
}

func (f *Fakegones) startServer(addr string, updates chan<- func(*dashboard.Dashboard), incoming chan *connectionRequest) {
	updates <- func(dash *dashboard.Dashboard) {
		dash.Fakegones.Servers[addr] = &dashboard.Server{Status: "Waiting for first player"}
	}

	status := func(s string) {
		updates <- func(dash *dashboard.Dashboard) {
			dash.Fakegones.Servers[addr].Status = s
		}
	}

	addPlayer := func(s string) {
		updates <- func(dash *dashboard.Dashboard) {
			dash.Fakegones.Servers[addr].Players = append(dash.Fakegones.Servers[addr].Players, s)
		}
	}

	p1 := <-incoming
	status("Waiting for second player")
	addPlayer(p1.playerName)

	p2 := <-incoming
	status("Game startings")
	addPlayer(p2.playerName)

	flavorTexts := []string{
		"%s got a sick mid-air 360 no-scope headshot!",
		"%s switched to Hanzo 🤦",
		"All your base belong to %s.",
		"%s played Exodia",
		"%s used splash, it wasn't effective.",
		"%s got a tactical nuke.",
		"%s leveled up.",
		"%s crafted Diamond armour.",
	}

	names := []string{
		p1.playerName,
		p2.playerName,
	}

	for i := 0; i < 4; i++ {
		time.Sleep(time.Second * 4)
		status("In Game: " + fmt.Sprintf(flavorTexts[rand.Int()%len(flavorTexts)], names[rand.Int()%2]))
	}
	time.Sleep(time.Second * 4)
	status(fmt.Sprintf("%s won!", p1.playerName))
	time.Sleep(time.Second * 4)

	p1.done <- struct{}{}
	p2.done <- struct{}{}
	f.serverClose <- addr
	updates <- func(dash *dashboard.Dashboard) {
		delete(dash.Fakegones.Servers, addr)
	}
}

func (f *Fakegones) Allocate() string {
	r := make(chan string)
	f.newServer <- r
	return <-r
}

func (f *Fakegones) Connect(addr string, playerName string) {
	r := &connectionRequest{
		addr:       addr,
		playerName: playerName,
		done:       make(chan struct{}),
	}

	f.connectionRequest <- r
	<-r.done
}
