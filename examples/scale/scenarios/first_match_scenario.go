package scenarios

import (
	"fmt"
	"io"
<<<<<<< HEAD
	"open-match.dev/open-match/pkg/pb"
	"time"
=======
	"time"

	"open-match.dev/open-match/pkg/pb"
)

const (
	poolName = "all"
>>>>>>> 8e1fbaf93832b3ea5118939ad33dabb8d1a260b4
)

var (
	firstMatchScenario = &Scenario{
		MMF:                          queryPoolsWrapper(firstMatchMmf),
		Evaluator:                    fifoEvaluate,
		FrontendTotalTicketsToCreate: -1,
		FrontendTicketCreatedQPS:     100,
		BackendAssignsTickets:        true,
		BackendDeletesTickets:        true,
		Ticket:                       firstMatchTicket,
		Profiles:                     firstMatchProfile,
	}
)

func firstMatchProfile() []*pb.MatchProfile {
	return []*pb.MatchProfile{
		{
			Name: "entirePool",
			Pools: []*pb.Pool{
				{
<<<<<<< HEAD
					Name: "all",
=======
					Name: poolName,
>>>>>>> 8e1fbaf93832b3ea5118939ad33dabb8d1a260b4
				},
			},
		},
	}
}

func firstMatchTicket() *pb.Ticket {
	return &pb.Ticket{}
}

func firstMatchMmf(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
<<<<<<< HEAD
	tickets := poolTickets["all"]
	var matches []*pb.Match

	for i := 0; i+1 <= len(tickets); i += 2 {
=======
	tickets := poolTickets[poolName]
	var matches []*pb.Match

	for i := 0; i+1 < len(tickets); i += 2 {
>>>>>>> 8e1fbaf93832b3ea5118939ad33dabb8d1a260b4
		matches = append(matches, &pb.Match{
			MatchId:       fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
			Tickets:       []*pb.Ticket{tickets[i], tickets[i+1]},
			MatchProfile:  p.GetName(),
			MatchFunction: "rangeExpandingMatchFunction",
		})
	}

	return matches, nil
}

// fifoEvaluate accepts all matches which don't contain the same ticket as in a
// previously accepted match.  Essentially first to claim the ticket wins.
func fifoEvaluate(stream pb.Evaluator_EvaluateServer) error {
	used := map[string]struct{}{}

	// TODO: once the evaluator client supports sending and recieving at the
	// same time, don't buffer, just send results immediately.
	matches := []*pb.Match{}

outer:
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Error reading evaluator input stream: %w", err)
		}

		m := req.GetMatch()

		for _, t := range m.Tickets {
			if _, ok := used[t.Id]; ok {
				continue outer
			}
		}

		for _, t := range m.Tickets {
			used[t.Id] = struct{}{}
		}

		matches = append(matches, m)
	}

	for _, m := range matches {
		err := stream.Send(&pb.EvaluateResponse{Match: m})
		if err != nil {
			return fmt.Errorf("Error sending evaluator output stream: %w", err)
		}
	}

	return nil
}
