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

package scenarios

import (
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"open-match.dev/open-match/examples/scale/scenarios/battleroyal"
	"open-match.dev/open-match/examples/scale/scenarios/firstmatch"
	"open-match.dev/open-match/examples/scale/scenarios/teamshooter"
	"open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
)

var (
	queryServiceAddress = "om-query.open-match.svc.cluster.local:50503" // Address of the QueryService Endpoint.

	logger = logrus.WithFields(logrus.Fields{
		"app": "scale",
	})
)

// GameScenario defines what tickets look like, and how they should be matched.
type GameScenario interface {
	// Ticket creates a new ticket, with randomized parameters.
	Ticket() *pb.Ticket

	// Profiles lists all of the profiles that should run.
	Profiles() []*pb.MatchProfile

	// MatchFunction is the custom logic implementation of the match function.
	MatchFunction(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error)

	// Evaluate is the custom logic implementation of the evaluator.
	Evaluate(stream pb.Evaluator_EvaluateServer) error
}

// ActiveScenario sets the scenario with preset parameters that we want to use for current Open Match benchmark run.
var ActiveScenario = func() *Scenario {
	s := &Scenario{}

	{ // Game Scenario
		var gs GameScenario = firstmatch.Scenario()

		// TODO: Select which scenario to use based on some configuration or choice,
		// so it's easier to run different scenarios without changing code.
		gs = battleroyal.Scenario()
		gs = teamshooter.Scenario()

		s.Ticket = gs.Ticket
		s.Profiles = gs.Profiles

		s.MMF = queryPoolsWrapper(gs.MatchFunction)
		s.Evaluator = gs.Evaluate
	}

	{ // Usage Pattern
		// TODO: Select between several different usage patterns.
		s.BackendAssignsTickets = true
		s.BackendDeletesTickets = true
	}

	s.FrontendTotalTicketsToCreate = -1
	s.FrontendTicketCreatedQPS = 100

	// TODO: add optiosn for TicketExtensionSize and MatchExtensionSize

	return s
}()

// Scenario defines the controllable fields for Open Match benchmark scenarios
type Scenario struct {
	// GameFrontend Configs
	// TicketExtensionSize       int
	// MatchExtensionSize        int

	FrontendTotalTicketsToCreate int // TotalTicketsToCreate = -1 let scale-frontend create tickets forever
	FrontendTicketCreatedQPS     uint32

	// GameBackend Configs
	BackendAssignsTickets bool
	BackendDeletesTickets bool

	Ticket   func() *pb.Ticket
	Profiles func() []*pb.MatchProfile

	MMF       matchFunction
	Evaluator evaluatorFunction
}

type matchFunction func(*pb.RunRequest, pb.MatchFunction_RunServer) error
type evaluatorFunction func(pb.Evaluator_EvaluateServer) error

func (mmf matchFunction) Run(req *pb.RunRequest, srv pb.MatchFunction_RunServer) error {
	return mmf(req, srv)
}

func (eval evaluatorFunction) Evaluate(srv pb.Evaluator_EvaluateServer) error {
	return eval(srv)
}

func getQueryServiceGRPCClient() pb.QueryServiceClient {
	conn, err := grpc.Dial(queryServiceAddress, testing.NewGRPCDialOptions(logger)...)
	if err != nil {
		logger.Fatalf("Failed to connect to Open Match, got %v", err)
	}
	return pb.NewQueryServiceClient(conn)
}

func queryPoolsWrapper(mmf func(req *pb.MatchProfile, pools map[string][]*pb.Ticket) ([]*pb.Match, error)) matchFunction {
	var q pb.QueryServiceClient
	var startQ sync.Once

	return func(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
		startQ.Do(func() {
			q = getQueryServiceGRPCClient()
		})

		poolTickets, err := matchfunction.QueryPools(stream.Context(), q, req.GetProfile().GetPools())
		if err != nil {
			return err
		}

		proposals, err := mmf(req.GetProfile(), poolTickets)
		if err != nil {
			return err
		}

		logger.WithFields(logrus.Fields{
			"proposals": proposals,
		}).Trace("proposals returned by match function")

		for _, proposal := range proposals {
			if err := stream.Send(&pb.RunResponse{Proposal: proposal}); err != nil {
				return err
			}
		}

		return nil
	}
}
