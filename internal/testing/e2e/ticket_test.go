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
	"io"
	"testing"
	// 	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/pkg/pb"
)

// TestAssignTickets covers assigning multiple tickets, using two different
// assignment groups.
func TestAssignTickets(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)
	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)
	t3, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	req := &pb.AssignTicketsRequest{
		Assignments: []*pb.AssignmentGroup{
			{
				TicketIds:  []string{t1.Id},
				Assignment: &pb.Assignment{Connection: "a"},
			},
			{
				TicketIds:  []string{t2.Id, t3.Id},
				Assignment: &pb.Assignment{Connection: "b"},
			},
		},
	}

	resp, err := om.Backend().AssignTickets(ctx, req)
	require.Nil(t, err)
	require.Equal(t, &pb.AssignTicketsResponse{}, resp)

	get, err := om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: t1.Id})
	require.Nil(t, err)
	require.Equal(t, "a", get.Assignment.Connection)

	get, err = om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: t2.Id})
	require.Nil(t, err)
	require.Equal(t, "b", get.Assignment.Connection)

	get, err = om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: t3.Id})
	require.Nil(t, err)
	require.Equal(t, "b", get.Assignment.Connection)
}

// TestAssignTicketsInvalidArgument covers various invalid calls to assign
// tickets.
func TestAssignTicketsInvalidArgument(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	ctResp, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	for _, tt := range []struct {
		name string
		req  *pb.AssignTicketsRequest
		msg  string
	}{
		{
			"missing assignment",
			&pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{},
				},
			},
			"AssignmentGroup.Assignment is required",
		},
		{
			"ticket used twice one group",
			&pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{
						TicketIds:  []string{ctResp.Id, ctResp.Id},
						Assignment: &pb.Assignment{},
					},
				},
			},
			"Ticket id " + ctResp.Id + " is assigned multiple times in one assign tickets call.",
		},
		{
			"ticket used twice two groups",
			&pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{
						TicketIds:  []string{ctResp.Id},
						Assignment: &pb.Assignment{Connection: "a"},
					},
					{
						TicketIds:  []string{ctResp.Id},
						Assignment: &pb.Assignment{Connection: "b"},
					},
				},
			},
			"Ticket id " + ctResp.Id + " is assigned multiple times in one assign tickets call.",
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := om.Backend().AssignTickets(ctx, tt.req)
			require.Equal(t, codes.InvalidArgument, status.Convert(err).Code())
			require.Equal(t, tt.msg, status.Convert(err).Message())
		})
	}
}

// TestAssignTicketsMissingTicket covers that when a ticket was deleted before
// being assigned, the assign tickets calls succeeds, however it returns a
// notice that the ticket was missing.
func TestAssignTicketsMissingTicket(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	t3, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	_, err = om.Frontend().DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: t2.Id})
	require.Nil(t, err)

	req := &pb.AssignTicketsRequest{
		Assignments: []*pb.AssignmentGroup{
			{
				TicketIds:  []string{t1.Id, t2.Id, t3.Id},
				Assignment: &pb.Assignment{Connection: "a"},
			},
		},
	}

	resp, err := om.Backend().AssignTickets(ctx, req)
	require.Nil(t, err)
	require.Equal(t, &pb.AssignTicketsResponse{
		Failures: []*pb.AssignmentFailure{
			{
				TicketId: t2.Id,
				Cause:    pb.AssignmentFailure_TICKET_NOT_FOUND,
			},
		},
	}, resp)
}

func TestTicketDelete(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	_, err = om.Frontend().DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: t1.Id})
	require.Nil(t, err)

	resp, err := om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: t1.Id})
	require.Nil(t, resp)
	require.Equal(t, "Ticket id:"+t1.Id+" not found", status.Convert(err).Message())
	require.Equal(t, codes.NotFound, status.Convert(err).Code())
}

// // TestTicketLifeCycle tests creating, getting and deleting a ticket using Frontend service.
// func TestTicketLifeCycle(t *testing.T) {
// 	require := require.New(t)

// 	om := newOM(t)
// 	fe := om.MustFrontendGRPC()
// 	require.NotNil(fe)
// 	ctx := om.Context()

// 	ticket := &pb.Ticket{
// 		SearchFields: &pb.SearchFields{
// 			DoubleArgs: map[string]float64{
// 				"test-property": 1,
// 			},
// 		},
// 	}

// 	// Create a ticket, validate that it got an id and set its id in the expected ticket.
// 	createResp, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: ticket})
// 	require.NotNil(createResp)
// 	require.NotNil(createResp.GetId())
// 	require.Nil(err)
// 	ticket.Id = createResp.GetId()
// 	validateTicket(t, createResp, ticket)

// 	// Fetch the ticket and validate that it is identical to the expected ticket.
// 	gotTicket, err := fe.GetTicket(ctx, &pb.GetTicketRequest{TicketId: ticket.GetId()})
// 	require.NotNil(gotTicket)
// 	require.Nil(err)
// 	validateTicket(t, gotTicket, ticket)

// 	// Delete the ticket and validate that it was actually deleted.
// 	_, err = fe.DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: ticket.GetId()})
// 	require.Nil(err)
// 	validateDelete(ctx, t, fe, ticket.GetId())
// }

// // validateTicket validates that the fetched ticket is identical to the expected ticket.
// func validateTicket(t *testing.T, got *pb.Ticket, want *pb.Ticket) {
// 	require.Equal(t, got.GetId(), want.GetId())
// 	require.Equal(t, got.SearchFields.DoubleArgs["test-property"], want.SearchFields.DoubleArgs["test-property"])
// 	require.Equal(t, got.GetAssignment().GetConnection(), want.GetAssignment().GetConnection())
// }

// // validateDelete validates that the ticket is actually deleted from the state storage.
// // Given that delete is async, this method retries fetch every 100ms up to 5 seconds.
// func validateDelete(ctx context.Context, t *testing.T, fe pb.FrontendServiceClient, id string) {
// 	start := time.Now()
// 	for {
// 		if time.Since(start) > 5*time.Second {
// 			break
// 		}

// 		// Attempt to fetch the ticket every 100ms
// 		_, err := fe.GetTicket(ctx, &pb.GetTicketRequest{TicketId: id})
// 		if err != nil {
// 			// Only failure to fetch with NotFound should be considered as success.
// 			require.Equal(t, status.Code(err), codes.NotFound)
// 			return
// 		}

// 		time.Sleep(100 * time.Millisecond)
// 	}

// 	require.Failf(t, "ticket %v not deleted after 5 seconds", id)
// }

// TestEmptyReleaseTicketsRequest covers that it is valid to not have any ticket
// ids when releasing tickets.  (though it's not really doing anything...)
func TestEmptyReleaseTicketsRequest(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	resp, err := om.Backend().ReleaseTickets(ctx, &pb.ReleaseTicketsRequest{
		TicketIds: nil,
	})

	require.Nil(t, err)
	require.Equal(t, &pb.ReleaseTicketsResponse{}, resp)
}

// TestReleaseTickets covers that tickets returned from matches are no longer
// returned by query tickets, but will return after being released.
func TestReleaseTickets(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	var ticket *pb.Ticket

	{ // Create ticket
		var err error
		ticket, err = om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
		require.Nil(t, err)
		require.NotEmpty(t, ticket.Id)
	}

	{ // Ticket present in query
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Nil(t, err)
		require.Len(t, resp.Tickets, 1)
		require.Equal(t, ticket.Id, resp.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	{ // Ticket returned from match
		om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
			out <- &pb.Match{
				MatchId: "1",
				Tickets: []*pb.Ticket{ticket},
			}
			return nil
		})
		om.SetEvaluator(evaluatorAcceptAll)

		stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
			Config: om.MMFConfigGRPC(),
			Profile: &pb.MatchProfile{
				Name: "test-profile",
				Pools: []*pb.Pool{
					{Name: "pool"},
				},
			},
		})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Nil(t, err)
		require.Len(t, resp.Match.Tickets, 1)
		require.Equal(t, ticket.Id, resp.Match.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	{ // Ticket NOT present in query
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	{ // Return ticket
		resp, err := om.Backend().ReleaseTickets(ctx, &pb.ReleaseTicketsRequest{
			TicketIds: []string{ticket.Id},
		})

		require.Nil(t, err)
		require.Equal(t, &pb.ReleaseTicketsResponse{}, resp)
	}

	{ // Ticket present in query
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Nil(t, err)
		require.Len(t, resp.Tickets, 1)
		require.Equal(t, ticket.Id, resp.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}
}

func TestCreateTicketErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		req  *pb.CreateTicketRequest
		code codes.Code
		msg  string
	}{
		{
			"missing ticket",
			&pb.CreateTicketRequest{
				Ticket: nil,
			},
			codes.InvalidArgument,
			".ticket is required",
		},
		{
			"already has assignment",
			&pb.CreateTicketRequest{
				Ticket: &pb.Ticket{
					Assignment: &pb.Assignment{},
				},
			},
			codes.InvalidArgument,
			"tickets cannot be created with an assignment",
		},
		{
			"already has create time",
			&pb.CreateTicketRequest{
				Ticket: &pb.Ticket{
					CreateTime: ptypes.TimestampNow(),
				},
			},
			codes.InvalidArgument,
			"tickets cannot be created with create time set",
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			om := newOM(t)
			ctx := context.Background()

			resp, err := om.Frontend().CreateTicket(ctx, tt.req)
			require.Nil(t, resp)
			s := status.Convert(err)
			require.Equal(t, tt.code, s.Code())
			require.Equal(t, s.Message(), tt.msg)
		})
	}
}