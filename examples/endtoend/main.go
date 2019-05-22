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

// Package main TODO
package main

//  "open-match.dev/open-match/internal/pb"
import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "endtoend",
	})
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	logging.ConfigureLogging(cfg)

	logger.Info("Starting Server")

	d := newDashboard()

	pm := NewPlayerManager(d.update)

	go pm.NewAi()

	go func() {
		for range time.Tick(time.Second) {
			d.update <- func(ds *ds) {
				ds.Uptime++
			}
		}
	}()

	fileServe := http.FileServer(http.Dir("/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fileServe))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dashboard/" {
			http.NotFound(w, r)
			return
		}
		fileServe.ServeHTTP(w, r)
	})

	http.Handle("/dashboard/connect", websocket.Handler(d.newListener))

	// TODO: Other services read their port from the common config map (oddly including
	// the mmf, which isn't part of the core open-match), how should this be choosing the ports
	// it exposes?
	err = http.ListenAndServe(":3389", nil)
	logger.WithFields(logrus.Fields{
		"err": err.Error(),
	}).Fatal("Http ListenAndServe failed.")
}

type dashboard struct {
	subscribe chan *chanContext
	ds        *ds
	update    chan func(*ds)
}

func newDashboard() *dashboard {
	d := dashboard{
		subscribe: make(chan *chanContext),
		ds:        newDs(),
		update:    make(chan func(*ds)),
	}

	updates := make(chan []byte)

	go multiplexByteChan(collapseByteChan(updates), d.subscribe)

	go func() {

		for f := range d.update {
			f(d.ds)

		applyAllWaitingUpdates:
			for {
				select {
				case f := <-d.update:
					f(d.ds)
				default:
					break applyAllWaitingUpdates
				}
			}

			b, err := json.MarshalIndent(d.ds, "", "  ")
			if err == nil {
				updates <- b
			} else {
				logger.WithFields(logrus.Fields{
					"err": err.Error(),
				}).Error("Error encoding dashboard state.")
			}
		}
	}()

	return &d
}

func (d *dashboard) newListener(ws *websocket.Conn) {
	ctx, cancel := context.WithCancel(ws.Request().Context())

	go func() {
		updates := make(chan []byte)

		d.subscribe <- &chanContext{
			c:   updates,
			ctx: ctx,
		}
		collapsed := collapseByteChan(updates)

		for b := range collapsed {
			_, err := ws.Write(b)
			if err != nil {
				cancel()
				for range collapsed {
				}
				return
			}
		}
	}()

	<-ctx.Done()
}

// Relays from the input channel to the returned channel.  If multiple inputs
// are sent before the output channel accepts the input, only the most recent
// input is sent.  Closing the input channel closes the output channel, dropping
// any unsent inputs.
func collapseByteChan(in <-chan []byte) <-chan []byte {
	out := make(chan []byte)
	go func() {
		for {
			recent, ok := <-in
			if !ok {
				close(out)
				return
			}

		tryToSend:
			for {
				select {
				case recent, ok = <-in:
					if !ok {
						close(out)
						return
					}
				case out <- recent:
					break tryToSend
				}
			}
		}
	}()
	return out
}

type chanContext struct {
	c   chan<- []byte
	ctx context.Context
}

// Sends inputs to all subscribers.  When the subsciber's context is canceled,
// the subscribers channel will be closed.  If in is closed, the subscribers are
// closed.  A new subscriber gets the most recent value.
func multiplexByteChan(in <-chan []byte, subscribe chan *chanContext) {
	s := make(map[*chanContext]struct{})

	var latest []byte

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {

		case <-ticker.C:
			for cc := range s {
				select {
				case <-cc.ctx.Done():
					delete(s, cc)
					close(cc.c)
				default:
				}
			}

		case v, ok := <-in:
			latest = v
			if !ok {
				for cc := range s {
					close(cc.c)
				}
			}

			for cc := range s {
				select {
				case <-cc.ctx.Done():
					delete(s, cc)
					close(cc.c)
				case cc.c <- v:
				}
			}

		case cc := <-subscribe:
			s[cc] = struct{}{}
			select {
			case <-cc.ctx.Done():
				delete(s, cc)
				close(cc.c)
			case cc.c <- latest:
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

type ds struct {
	Uptime  int
	Players map[Id]*Player
}

func newDs() *ds {
	return &ds{
		Players: make(map[Id]*Player),
	}
}

type Player struct {
	Name   string
	Status string
	Error  string
}

////////////////////////////////////////////////////////////////////////////////

type Id int64

type PlayerManager struct {
	nextId chan Id
	update chan<- func(*ds)
}

func NewPlayerManager(update chan<- func(*ds)) *PlayerManager {
	pm := &PlayerManager{
		nextId: make(chan Id),
		update: update,
	}

	go func() {
		for id := Id(0); true; id++ {
			pm.nextId <- id
		}
	}()

	return pm
}

func (pm *PlayerManager) NewAi() {
	id := <-pm.nextId

	pm.update <- func(ds *ds) {
		ds.Players[id] = &Player{
			Name:   fmt.Sprintf("Robo-bot_%d", id),
			Status: "Main Menu",
		}
	}
}
