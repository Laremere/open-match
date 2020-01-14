package scenarios

import (
	"fmt"
	"io"
	"math/rand"
	"sort"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"open-match.dev/open-match/pkg/pb"
)

var (
	rangeExpandingScenario = (&res{
		regions: 5,
	}).Scenario()
)

// Currently doesn't "expand".
type res struct {
	// Inputs
	regions            int
	modePopulations    map[string]int
	playersPerGame     int
	maxSkillDifference float64

	// Calculated
	modePopulationTotal int
}

func (r *res) Scenario() *Scenario {
	// TODO:Specify ticket and pools.
	return &Scenario{
		MMF:                          queryPoolsWrapper(r.mmf),
		Evaluator:                    r.evaluate,
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

	for region := 0; region < r.regions; region++ {

		p = append(p, &pb.Pool{
			TagPresentFilters: []*pb.TagPresentFilter{
				{
					Tag: fmt.Sprintf("region_%d", region),
				},
			},
		})
	}

	return p
}

func (r *res) ticket() *pb.Ticket {

	// switch rand.Intn(10) {
	// case 0:
	// 	dargs["A"] = rand.Float64()
	// }

	mode := ""
	switch rand.Intn(6) {
	case 0, 1, 2:
		mode = "FOO"
	case 3, 4:
		mode = "BAR"
	case 5:
		mode = "BAZ"
	}

	return &pb.Ticket{
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

	_ = tickets
	// for {
	// 	matches = append(matches, &pb.Match{
	// 		MatchId:       fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
	// 		MatchProfile:  p.GetName(),
	// 		MatchFunction: "rangeExpandingMatchFunction",
	// 		Tickets:       matchTickets,
	// 	})
	// }

	return matches, nil
}

func (r *res) evaluate(stream pb.Evaluator_EvaluateServer) error {
	// Unpacked proposal matches.
	proposals := []*matchExt{}
	// Ticket ids which are used in a match.
	used := map[string]struct{}{}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Error reading evaluator input stream: %w", err)
		}

		p, err := unpackMatch(req.GetMatch())
		if err != nil {
			return err
		}
		proposals = append(proposals, p)
	}

	// Lower variance is better.
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].variance < proposals[j].variance
	})

outer:
	for _, p := range proposals {
		for _, t := range p.tickets {
			if _, ok := used[t.Id]; ok {
				continue outer
			}
		}

		for _, t := range p.tickets {
			used[t.Id] = struct{}{}
		}

		err := stream.Send(&pb.EvaluateResponse{Match: p.original})
		if err != nil {
			return fmt.Errorf("Error sending evaluator output stream: %w", err)
		}
	}

	return nil
}

type matchExt struct {
	id       string
	tickets  []*pb.Ticket
	variance float64
	original *pb.Match
}

func unpackMatch(m *pb.Match) (*matchExt, error) {
	v := &wrappers.DoubleValue{}

	err := ptypes.UnmarshalAny(m.Extensions["variance"], v)
	if err != nil {
		return nil, fmt.Errorf("Error unpacking match variance: %w", err)
	}

	return &matchExt{
		id:       m.MatchId,
		tickets:  m.Tickets,
		variance: v.Value,
	}, nil
}

func (m *matchExt) pack() (*pb.Match, error) {
	if m.original != nil {
		return nil, fmt.Errorf("Packing match which has original, not safe to preserve extensions.")
	}

	v := &wrappers.DoubleValue{Value: m.variance}

	a, err := ptypes.MarshalAny(v)
	if err != nil {
		return nil, fmt.Errorf("Error packing match variance: %w", err)
	}

	return &pb.Match{
		MatchId: m.id,
		Tickets: m.tickets,
		Extensions: map[string]*any.Any{
			"variance": a,
		},
	}, nil
}
