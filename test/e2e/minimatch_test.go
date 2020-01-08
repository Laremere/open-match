// +build !e2ecluster

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

package e2e

import (
	"context"
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

type testProfile struct {
	name  string
	pools []*pb.Pool
}

type testTicket struct {
	skill      float64
	mapValue   string
	targetPool string
	id         string
}

func TestMinimatch(t *testing.T) {
	// TODO: Currently, the E2E test uses globally defined test data. Consider
	// improving this in future iterations to test data scoped to sepcific test cases

	// Tickets used for testing. The id will be populated once the Tickets are created.
	sourceTickets := []testTicket{
		{skill: 1, mapValue: e2e.DoubleArgMMR, targetPool: e2e.Map1BeginnerPool},
		{skill: 5, mapValue: e2e.DoubleArgMMR, targetPool: e2e.Map1BeginnerPool},
		{skill: 8, mapValue: e2e.DoubleArgMMR, targetPool: e2e.Map1AdvancedPool},
		{skill: 0, mapValue: e2e.DoubleArgLevel, targetPool: e2e.Map2BeginnerPool},
		{skill: 6, mapValue: e2e.DoubleArgLevel, targetPool: e2e.Map2AdvancedPool},
		{skill: 10, mapValue: e2e.DoubleArgLevel, targetPool: e2e.Map2AdvancedPool},
		{skill: 10, mapValue: "non-indexable-map"},
	}

	// The Pools that are used for testing. Currently, each pool simply represents
	// a combination of the DoubleArgs indexed on a ticket.
	testPools := map[string]*pb.Pool{
		e2e.Map1BeginnerPool: {
			Name: e2e.Map1BeginnerPool,
			DoubleRangeFilters: []*pb.DoubleRangeFilter{
				{DoubleArg: e2e.DoubleArgDefense, Min: 0, Max: 5},
				{DoubleArg: e2e.DoubleArgMMR, Max: math.MaxFloat64},
			},
		},
		e2e.Map1AdvancedPool: {
			Name: e2e.Map1AdvancedPool,
			DoubleRangeFilters: []*pb.DoubleRangeFilter{
				{DoubleArg: e2e.DoubleArgDefense, Min: 6, Max: 10},
				{DoubleArg: e2e.DoubleArgMMR, Max: math.MaxFloat64},
			},
		},
		e2e.Map2BeginnerPool: {
			Name: e2e.Map2BeginnerPool,
			DoubleRangeFilters: []*pb.DoubleRangeFilter{
				{DoubleArg: e2e.DoubleArgDefense, Min: 0, Max: 5},
				{DoubleArg: e2e.DoubleArgLevel, Max: math.MaxFloat64},
			},
		},
		e2e.Map2AdvancedPool: {
			Name: e2e.Map2AdvancedPool,
			DoubleRangeFilters: []*pb.DoubleRangeFilter{
				{DoubleArg: e2e.DoubleArgDefense, Min: 6, Max: 10},
				{DoubleArg: e2e.DoubleArgLevel, Max: math.MaxFloat64},
			},
		},
	}

	// Test profiles being tested for. Note that each profile embeds two pools - and
	// the current MMF returns a match per pool in the profile - so each profile should
	// output two matches that are comprised of tickets belonging to that pool.
	sourceProfiles := []testProfile{
		{name: "", pools: []*pb.Pool{testPools[e2e.Map1BeginnerPool], testPools[e2e.Map1AdvancedPool]}},
		{name: "", pools: []*pb.Pool{testPools[e2e.Map2BeginnerPool], testPools[e2e.Map2AdvancedPool]}},
	}

	tests := []struct {
		fcGen       func(e2e.OM) *pb.FunctionConfig
		description string
	}{
		{
			func(om e2e.OM) *pb.FunctionConfig { return om.MustMmfConfigGRPC() },
			"grpc config",
		},
		{
			func(om e2e.OM) *pb.FunctionConfig { return om.MustMmfConfigHTTP() },
			"http config",
		},
	}

	t.Run("TestMinimatch", func(t *testing.T) {
		for _, test := range tests {
			test := test
			testTickets := make([]testTicket, len(sourceTickets))
			testProfiles := make([]testProfile, len(sourceProfiles))
			copy(testTickets, sourceTickets)
			copy(testProfiles, sourceProfiles)

			t.Run(fmt.Sprintf("TestMinimatch-%v", test.description), func(t *testing.T) {
				t.Parallel()

				om, closer := e2e.New(t)
				defer closer()
				fc := test.fcGen(om)

				fe := om.MustFrontendGRPC()
				mml := om.MustMmLogicGRPC()
				ctx := om.Context()

				// Create all the tickets and validate ticket creation succeeds. Also populate ticket ids
				// to expected player pools.
				for i, td := range testTickets {
					resp, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{
						SearchFields: &pb.SearchFields{
							DoubleArgs: map[string]float64{
								e2e.DoubleArgDefense: td.skill,
								td.mapValue:          float64(time.Now().Unix()),
							},
						},
					}})

					require.Nil(t, err)
					require.NotNil(t, resp)
					testTickets[i].id = resp.Ticket.Id
				}

				// poolTickets represents a map of the pool name to all the ticket ids in the pool.
				poolTickets := make(map[string][]string)

				// Query tickets for each pool
				for _, pool := range testPools {
					qtstr, err := mml.QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: pool})
					require.Nil(t, err)
					require.NotNil(t, qtstr)

					var tickets []*pb.Ticket
					for {
						qtresp, err := qtstr.Recv()
						if err == io.EOF {
							break
						}
						require.Nil(t, err)
						require.NotNil(t, qtresp)
						tickets = append(tickets, qtresp.Tickets...)
					}

					// Generate a map of pool name to all tickets belonging to the pool.
					for _, ticket := range tickets {
						poolTickets[pool.Name] = append(poolTickets[pool.Name], ticket.Id)
					}

					// Generate a map of pool name to all tickets that should belong to the pool
					// based on test ticket data.
					var want []string
					for _, td := range testTickets {
						if td.targetPool == pool.Name {
							want = append(want, td.id)
						}
					}

					// Validate that all the pools have the expected tickets.
					require.ElementsMatch(t, poolTickets[pool.Name], want)
				}

				testFetchMatches(ctx, t, poolTickets, testProfiles, om, fc)
			})
		}
	})
}

func testFetchMatches(ctx context.Context, t *testing.T, poolTickets map[string][]string, testProfiles []testProfile, om e2e.OM, fc *pb.FunctionConfig) {
	// Fetch Matches for each test profile.
	be := om.MustBackendGRPC()
	for _, profile := range testProfiles {
		stream, err := be.FetchMatches(ctx, &pb.FetchMatchesRequest{
			Config:   fc,
			Profiles: []*pb.MatchProfile{{Name: profile.name, Pools: profile.pools}},
		})
		require.Nil(t, err)

		for {
			var resp *pb.FetchMatchesResponse
			resp, err = stream.Recv()
			if err == io.EOF {
				break
			}
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, resp.GetMatch())

			match := resp.GetMatch()
			// Currently, the MMF simply creates a match per pool in the match profile - and populates
			// the roster with the pool name. Thus validate that for the roster populated in the match
			// result has all the tickets expected in that pool.
			require.Equal(t, len(match.GetRosters()), 1)
			require.ElementsMatch(t, match.GetRosters()[0].GetTicketIds(), poolTickets[match.GetRosters()[0].GetName()])

			var gotTickets []string
			for _, ticket := range match.GetTickets() {
				gotTickets = append(gotTickets, ticket.Id)
			}

			// Given that currently we only populate all tickets in a match in a Roster, validate that
			// all the tickets present in the result match are equal to the tickets in the pool for that match.
			require.ElementsMatch(t, gotTickets, poolTickets[match.GetRosters()[0].GetName()])
		}

		// Verify calling fetch matches twice within ttl interval won't yield new results
		stream, err = be.FetchMatches(om.Context(), &pb.FetchMatchesRequest{
			Config:   fc,
			Profiles: []*pb.MatchProfile{{Name: profile.name, Pools: profile.pools}},
		})
		require.Nil(t, err)

		br, err := stream.Recv()
		require.Equal(t, err, io.EOF)
		require.Nil(t, br)
	}
}
