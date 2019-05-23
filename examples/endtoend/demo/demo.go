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
	"fmt"
	"time"

	"open-match.dev/open-match/examples/endtoend/dashboard"
)

type Demo struct {
	nextPlayerId chan int64
	updates      chan<- func(*dashboard.Dashboard)
}

func New(updates chan func(*dashboard.Dashboard)) *Demo {
	d := &Demo{
		updates:      updates,
		nextPlayerId: make(chan int64),
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
}

func (d *Demo) NewDirector() {
	d.updates <- func(dash *dashboard.Dashboard) {
		dash.Director = &dashboard.Director{
			Status: "Starting",
		}
	}

	for {
		d.updates <- func(dash *dashboard.Dashboard) {
			dash.Director.Status = "Sleeping"
		}

		time.Sleep(time.Second * 5)

		d.updates <- func(dash *dashboard.Dashboard) {
			dash.Director.Status = "Making Match"
		}

		time.Sleep(time.Second * 5)
	}
}

type Director struct {
}

type DGS struct {
}
