package scenarios

import (
	"fmt"
	"io"
	"math/rand"
	"sort"
	"time"

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
	modePopulationTotal int /////////////////////////////////TODO calculate somewhere
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

func (r *res) pools() []*pb.Pool {
	p := []*pb.Pool{}

	for region := 0; region < r.regions; region++ {
		for mode := range r.modePopulations {
			p = append(p, &pb.Pool{
				TagPresentFilters: []*pb.TagPresentFilter{
					{
						Tag: fmt.Sprintf("region_%d", region),
					},
				},
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						StringArg: "mode",
						Value:     mode,
					},
				},
			})
		}
	}

	return p
}

func (r *res) ticket() *pb.Ticket {

	///////////////////////////////////////////////// TODO: regions
	v := rand.Intn(r.modePopulationTotal)
	mode := ""
	for m, pop := range r.modePopulations {
		v -= pop
		if pop <= 0 {
			mode = m
			break
		}
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
	skill := func(t *pb.Ticket) float64 {
		return t.SearchFields.DoubleArgs["skill"]
	}

	tickets := poolTickets["all"]
	var matches []*pb.Match

	sort.Slice(tickets, func(i, j int) bool {
		return skill(tickets[i]) < skill(tickets[j])
	})

	for i := 0; i+r.playersPerGame <= len(tickets); i++ {
		mt := tickets[i : i+r.playersPerGame]
		if skill(mt[len(mt)-1])-skill(mt[0]) < r.maxSkillDifference {
			avg := float64(0)
			for _, t := range mt {
				avg += skill(t)
			}
			avg /= float64(len(mt))

			q := float64(0)
			for _, t := range mt {
				diff := skill(t) - avg
				q -= diff * diff
			}

			m, err := (&matchExt{
				id:            fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
				matchProfile:  p.GetName(),
				matchFunction: "rangeExpandingMatchFunction",
				tickets:       mt,
				quality:       q,
			}).pack()
			if err != nil {
				return nil, err
			}
			matches = append(matches, m)
		}
	}

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

	// Higher quality is better.
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].quality < proposals[j].quality
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
	id            string
	tickets       []*pb.Ticket
	quality       float64
	matchProfile  string
	matchFunction string
	original      *pb.Match
}

func unpackMatch(m *pb.Match) (*matchExt, error) {
	v := &wrappers.DoubleValue{}

	err := ptypes.UnmarshalAny(m.Extensions["quality"], v)
	if err != nil {
		return nil, fmt.Errorf("Error unpacking match quality: %w", err)
	}

	return &matchExt{
		id:            m.MatchId,
		tickets:       m.Tickets,
		quality:       v.Value,
		matchProfile:  m.MatchProfile,
		matchFunction: m.MatchFunction,
	}, nil
}

func (m *matchExt) pack() (*pb.Match, error) {
	if m.original != nil {
		return nil, fmt.Errorf("Packing match which has original, not safe to preserve extensions.")
	}

	v := &wrappers.DoubleValue{Value: m.quality}

	a, err := ptypes.MarshalAny(v)
	if err != nil {
		return nil, fmt.Errorf("Error packing match quality: %w", err)
	}

	return &pb.Match{
		MatchId:       m.id,
		Tickets:       m.tickets,
		MatchProfile:  m.matchProfile,
		MatchFunction: m.matchFunction,
		Extensions: map[string]*any.Any{
			"quality": a,
		},
	}, nil
}
