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

// Package mmf provides a sample match function that uses the GRPC harness to set up 1v1 matches.
// This sample is a reference to demonstrate the usage of the GRPC harness and should only be used as
// a starting point for your match function. You will need to modify the
// matchmaking logic in this function based on your game's requirements.
package mmf

import (
	"container/heap"
	"sort"
	"time"

	mmfHarness "open-match.dev/open-match/pkg/harness/function/golang"
	"open-match.dev/open-match/pkg/pb"
)

var (
	matchName = "a-simple-1v1-matchfunction"
)

// MakeMatches is where your custom matchmaking logic lives.
func MakeMatches(p *mmfHarness.MatchFunctionParams) ([]*pb.Match, error) {
	// This simple match function does the following things
	// 1. Deduplicates the tickets from the pools into a single list.
	// 2. Groups players into 1v1 matches.

	ticketsById := map[string]*pb.Ticket{}

	for _, pool := range p.PoolNameToTickets {
		for _, ticket := range pool {
			ticketsById[ticket.GetId()] = ticket
		}
	}

	tickets := make([]ticket, 0, len(ticketsById))

	timeStamp := time.Now().Unix()

	_ = timeStamp
	for _, t := range ticketsById {
		mmr := t.Properties.Fields["mmr"].GetNumberValue()
		tickets = append(tickets, ticket{
			t:   t,
			mmr: mmr,
			low: mmr - maxDiff,
			high: mmr + maxDiff,
		})
	}

	sort.Sort(ticketSorter(tickets))

	diffs := make(differenceHeap, 0)

	var last *difference
	for i := 0; i < len(tickets)-1; i++ {
		diff := &difference{
			lower: &tickets[i],
			higher: &tickets[i + 1],
			left: last,
			right: nil,
		}
		diff.diff = diff.higher.mmr - diff.lower.mmr
		last.right = diff

		heap.Push(&diffs, diff)
		last = diff
	}

	var matches []*pb.Match

	// t := time.Now().Format("2006-01-02T15:04:05.00")

	// thisMatch := make([]*pb.Ticket, 0, 2)
	// matchNum := 0

	// for _, ticket := range tickets {
	// 	thisMatch = append(thisMatch, ticket)

	// 	if len(thisMatch) >= 2 {
	// 		matches = append(matches, &pb.Match{
	// 			MatchId:       fmt.Sprintf("profile-%s-time-%s-num-%d", p.ProfileName, t, matchNum),
	// 			MatchProfile:  p.ProfileName,
	// 			MatchFunction: matchName,
	// 			Tickets:       thisMatch,
	// 		})

	// 		thisMatch = make([]*pb.Ticket, 0, 2)
	// 		matchNum++
	// 	}
	// }

	return matches, nil
}

type ticket struct {
	t    *pb.Ticket
	mmr  float64
	low  float64
	high float64
}

type difference struct {
	lower  *ticket
	higher *ticket
	left   *difference
	right  *difference
	diff   float64
}

type ticketSorter []ticket

func (s ticketSorter) Len() int {
	return len(s)
}

func (s ticketSorter) Less(i, j int) bool {
	return s[i].mmr < s[j].mmr
}

func (s ticketSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type differenceHeap []*difference

func (h *differenceHeap) Len() int {
	return len(*h)
}

func (h *differenceHeap) Less(i, j int) bool {
	return (*h)[i].diff < (*h)[j].diff
}

func (h *differenceHeap) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
}

func (h *differenceHeap) Push(x interface{}) {
	*h = append(*h, x.(*difference))
}

func (h *differenceHeap) Pop() interface{} {
	newLen := len(*h) - 1
	result := (*h)[newLen]
	*h = (*h)[0:newLen]
	return result
}
