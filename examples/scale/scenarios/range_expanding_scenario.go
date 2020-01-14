package scenarios

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/testing/customize/evaluator"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/test/evaluator/evaluate"
)

var (
	rangeExpandingScenario = &res{
		regions: 5,
	}.Scenario()
)

type res struct {
	regions             int
	modePopulations     map[string]int
	modePopulationTotal int
}

func (r *res) Scenario() *Scenario {
	// TODO:Specify ticket and pools.
	return &Scenario{
		MMF:                          queryPoolsWrapper(r.rangeExpandingMatchFunction),
		Evaluator:                    fifoEvaluate,
		FrontendTotalTicketsToCreate: -1,
		FrontendTicketCreatedQPS:     500,
		BackendAssignsTickets:        true,
		BackendDeletesTickets:        true,
	}
}

//         B            F
//  A
//       C           E
//   D

func (r *res) pools() []*pb.Pool {
	p := []*pb.Pool{}

	for i := 0; i < r.regions; i++ {
		p = append(p, *pb.Pool{
			TagPresentFilters: {
				{
					Tag: fmt.Sprintf("region_%d", i),
				},
			},
		})
	}

	return p
}

func (r *res) ticket() *pb.Ticket {

	switch rand.IntN(10) {
	case 0:
		dargs["A"] = rand.Float64()
	}

	mode := ""
	switch rand.IntN(6) {
	case 0, 1, 2:
		mode = "FOO"
	case 3, 4:
		mode = "BAR"
	case 5:
		mode = "BAZ"
	}

	return *pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"skill": clamp(rand.NormFloat64(), -3, 3),
			},
			StringArgs: map[string]string{
				"Mode": mode,
			},
		},
	}
}

func (r *res) mmf(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	tickets := poolTickets["all"]
	var matches []*pb.Match

	for {
		matches = append(matches, &pb.Match{
			MatchId:       fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
			MatchProfile:  p.GetName(),
			MatchFunction: "rangeExpandingMatchFunction",
			Tickets:       matchTickets,
			Rosters:       matchRosters,
		})
	}

	return matches, nil
}

func clamp(v float64, min float64, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
