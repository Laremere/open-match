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

package expander

import (
	"github.com/rs/xid"
	"open-match.dev/open-match/examples"
	mmfHarness "open-match.dev/open-match/pkg/harness/function/golang"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

var (
	functionName = "expander-match"
)

// MakeMatches is where your custom matchmaking logic lives.
// This is the core match making function that will be triggered by Open Match to generate matches.
// The goal of this function is to generate predictable matches that can be validated without flakyness.
// This match function loops through all the pools and generates one match per pool aggregating all players
// in that pool in the generated match.
func MakeMatches(params *mmfHarness.MatchFunctionParams) ([]*pb.Match, error) {
	var result []*pb.Match
	for pool, tickets := range params.PoolNameToTickets {
		if len(tickets) != 0 {
			roster := &pb.Roster{Name: pool}

			for _, ticket := range tickets {
				roster.TicketIds = append(roster.GetTicketIds(), ticket.GetId())
			}

			result = append(result, &pb.Match{
				MatchId:       xid.New().String(),
				MatchProfile:  params.ProfileName,
				MatchFunction: functionName,
				Tickets:       tickets,
				Rosters:       []*pb.Roster{roster},
				Properties: structs.Struct{
					examples.MatchScore: structs.Number(scoreCalculator(tickets)),
				}.S(),
			})
		}
	}

	return result, nil
}

// This match function defines the quality of a match as the sum of the attribute values of all tickets per match
func scoreCalculator(tickets []*pb.Ticket) float64 {
	matchScore := 0.0
	for _, ticket := range tickets {
		for _, v := range ticket.GetProperties().GetFields() {
			matchScore += v.GetNumberValue()
		}
	}
	return matchScore
}

type backfill struct {
	team1 []*player
	team2 []*player
}

type group struct {
	averageSkill float64
	players      []*player
	ticketid     string
}

type player struct {
	id    string
	skill float64
}

func processNewMatches(groups []*group) []*pb.Match {

}

func processBackfills(backfills []*backfill, groups []*group) []*pb.Match {

}

// Lower is better
func rateMatch(teams [2][]*group) float64 {
	skillTotals := [2]float64{}
	for i, team := range teams {
		for _, group := range team {
			for _, player := range group {
				skillTotals[i] += player.skill
			}
		}
	}
}
