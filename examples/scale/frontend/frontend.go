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

package frontend

import (
	"context"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/examples/scale/scenarios"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "scale.frontend",
	})
	activeScenario = scenarios.ActiveScenario
	statProcessor  = scenarios.NewStatProcessor()

	mTicketsCreated            = telemetry.Counter("scale_frontend_tickets_created", "tickets created")
	mTicketCreationsFailed     = telemetry.Counter("scale_frontend_ticket_creations_failed", "tickets created")
	mTicketAssignmentsReceived = telemetry.Counter("scale_frontend_ticket_assignments_received", "ticket assignments received")
	mTicketAssignmentFailed    = telemetry.Counter("scale_frontend_ticket_assignment_failed", "ticket assignments failed")
	mTicketsDeleted            = telemetry.Counter("scale_frontend_tickets_deleted", "tickets deleted")
	mTicketDeletesFailed       = telemetry.Counter("scale_frontend_ticket_deletes_failed", "ticket deletes failed")
)

// Run triggers execution of the scale frontend component that creates
// tickets at scale in Open Match.
func BindService(p *rpc.ServerParams, cfg config.View) error {
	go run(cfg)

	return nil
}

func run(cfg config.View) {
	conn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("failed to get Frontend connection")
	}
	fe := pb.NewFrontendClient(conn)

	ticketQPS := int(activeScenario.FrontendTicketCreatedQPS)
	ticketTotal := activeScenario.FrontendTotalTicketsToCreate
	totalStarted := 0

outer:
	for range time.Tick(time.Second) {
		for i := 0; i < ticketQPS; i++ {
			go runScenario(fe)

			if totalStarted >= ticketTotal {
				break outer
			}
		}
	}

	select {}
}

func runScenario(fe pb.FrontendClient) {
	ctx := context.Background()

	// Space out requests by sleeping a random amount less than a second.
	time.Sleep(time.Duration(rand.Int63n(int64(time.Second))))

	var ticketId string

	{
		resp, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{
			Ticket: activeScenario.Ticket(),
		})
		if err != nil {
			telemetry.RecordUnitMeasurement(ctx, mTicketCreationsFailed)
			statProcessor.RecordError("failed to create a ticket", err)
			return
		}
		telemetry.RecordUnitMeasurement(ctx, mTicketsCreated)
		ticketId = resp.Ticket.Id
	}

	if activeScenario.FrontendWaitForAssignment {
		stream, err := fe.GetAssignments(ctx, &pb.GetAssignmentsRequest{
			TicketId: ticketId,
		})
		for {
			telemetry.RecordUnitMeasurement(ctx, mTicketsCreated)
			resp, err := stream.Recv()
			if err != nil {
				telemetry.RecordUnitMeasurement(ctx, mTicketAssignmentFailed)
				statProcessor.RecordError("failed to receive assignment", err)
				return
			}
			if resp.Assignment != nil && resp.Assignment.GetConnection() != "" {
				break
			}
		}

		err = stream.CloseSend()
		if err != nil {
			telemetry.RecordUnitMeasurement(ctx, mTicketAssignmentFailed)
			statProcessor.RecordError("failed to receive assignment", err)
			return
		}
		telemetry.RecordUnitMeasurement(ctx, mTicketAssignmentsReceived)
	}

	if activeScenario.FrontendDeletesTickets {
		_, err := fe.DeleteTicket(context.Background(), &pb.DeleteTicketRequest{
			TicketId: ticketId,
		})

		if err == nil {
			telemetry.RecordUnitMeasurement(ctx, mTicketsDeleted)
			statProcessor.IncrementStat("Deleted", 1)
		} else {
			telemetry.RecordUnitMeasurement(ctx, mTicketDeletesFailed)
			statProcessor.RecordError("failed to delete tickets", err)
		}
	}
}
