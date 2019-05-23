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

// Package dashboard contains
package dashboard

type Dashboard struct {
	Uptime   int
	Players  map[int64]*Player
	Director *Director
}

func New() *Dashboard {
	return &Dashboard{
		Players: make(map[int64]*Player),
	}
}

type Player struct {
	Name   string
	Status string
	Error  string
}

type Director struct {
	Status      string
	Error       string
	RecentMatch string
}

type Server struct {
	Scores map[string]int
	Status string
}
