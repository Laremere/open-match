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
	"go.opencensus.io/stats"
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

	mScenariosSleeping         = runGauge(telemetry.Gauge("scale_frontend_scenarios_sleeping", "scenarios sleeping"))
	mScenariosCreating         = runGauge(telemetry.Gauge("scale_frontend_scenarios_creating", "scenarios creating"))
	mScenariosWaiting          = runGauge(telemetry.Gauge("scale_frontend_scenarios_waiting", "scenarios waiting"))
	mScenariosDeleting         = runGauge(telemetry.Gauge("scale_frontend_scenarios_deleting", "scenarios deleting"))
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

	startScenario := make(chan struct{})

	if activeScenario.MaxFrontendOutstanding == 0 {
		activeScenario.MaxFrontendOutstanding = 80
	}

	for i := 0; i < activeScenario.MaxFrontendOutstanding; i++ {
		go func() {
			for range startScenario {
				runScenario(fe)
			}
		}()
	}

	for range time.Tick(time.Second) {
		for i := 0; i < ticketQPS; i++ {
			startScenario <- struct{}{}

			totalStarted++
			if ticketTotal != -1 && totalStarted >= ticketTotal {
				return
			}
		}
	}

	close(startScenario)
}

func runScenario(fe pb.FrontendClient) {
	ctx := context.Background()

	h := &holder{}
	defer h.release()

	h.start(mScenariosSleeping)
	// Space out requests by sleeping a random amount less than a second.
	time.Sleep(time.Duration(rand.Int63n(int64(time.Second))))

	var ticketId string

	{
		h.start(mScenariosCreating)

		resp, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{
			Ticket: activeScenario.Ticket(),
		})
		if err != nil {
			telemetry.RecordUnitMeasurement(ctx, mTicketCreationsFailed)
			logger.Error("failed to create a ticket", err)
			return
		}
		telemetry.RecordUnitMeasurement(ctx, mTicketsCreated)
		ticketId = resp.Ticket.Id
	}

	if activeScenario.FrontendWaitForAssignment {
		h.start(mScenariosWaiting)

		stream, err := fe.GetAssignments(ctx, &pb.GetAssignmentsRequest{
			TicketId: ticketId,
		})
		for {
			resp, err := stream.Recv()
			if err != nil {
				telemetry.RecordUnitMeasurement(ctx, mTicketAssignmentFailed)
				logger.Error("failed to receive assignment", err)
				return
			}
			if resp.Assignment != nil && resp.Assignment.GetConnection() != "" {
				break
			}
		}

		err = stream.CloseSend()
		if err != nil {
			telemetry.RecordUnitMeasurement(ctx, mTicketAssignmentFailed)
			logger.Error("failed to receive assignment", err)
			return
		}
		telemetry.RecordUnitMeasurement(ctx, mTicketAssignmentsReceived)
	}

	if activeScenario.FrontendDeletesTickets {
		h.start(mScenariosDeleting)

		_, err := fe.DeleteTicket(context.Background(), &pb.DeleteTicketRequest{
			TicketId: ticketId,
		})

		if err == nil {
			telemetry.RecordUnitMeasurement(ctx, mTicketsDeleted)
		} else {
			telemetry.RecordUnitMeasurement(ctx, mTicketDeletesFailed)
			logger.Error("failed to delete tickets", err)
		}
	}
}

func runGauge(s *stats.Int64Measure) chan<- int {
	c := make(chan int)

	go func() {
		ctx := context.Background()
		v := int64(0)
		for {
			telemetry.SetGauge(ctx, s, v)
			v += int64(<-c)
		}
	}()

	return c
}

type holder struct {
	c chan<- int
}

func (h *holder) start(c chan<- int) {
	h.release()
	h.c = c
	c <- 1
}

func (h *holder) release() {
	if h.c != nil {
		h.c <- -1
		h.c = nil
	}
}
