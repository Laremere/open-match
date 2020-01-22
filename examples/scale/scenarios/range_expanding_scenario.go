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
	rangeExpandingScenarioLarger = (&res{
		regions:    5,
		maxRegions: 3,
		modePopulations: map[string]int{
			"koth": 5,
			"pl":   4,
			"cp":   1,
		},
		playersPerGame:      12,
		maxSkillDifference:  0.05,
		skillBoundaries:     []float64{-3, -2, -1, 0, 1, 2, 3},
		modePopulationTotal: 10, // TODO calculate.
	}).Scenario()

	rangeExpandingScenario = (&res{
		regions:    2,
		maxRegions: 1,
		modePopulations: map[string]int{
			"koth": 6,
			"pl":   4,
		},
		playersPerGame:      20,
		maxSkillDifference:  0.01,
		skillBoundaries:     []float64{-3, 0, 3},
		modePopulationTotal: 10, // TODO calculate.
	}).Scenario()
)

// Currently doesn't "expand".
type res struct {
	// Inputs
	regions            int
	maxRegions         int
	modePopulations    map[string]int
	playersPerGame     int
	maxSkillDifference float64
	skillBoundaries    []float64

	// Calculated
	modePopulationTotal int /////////////////////////////////TODO calculate somewhere
}

func (r *res) Scenario() *Scenario {
	return &Scenario{
		MMF:                          queryPoolsWrapper(r.mmf),
		Evaluator:                    r.evaluate,
		FrontendTotalTicketsToCreate: -1,
		FrontendTicketCreatedQPS:     100,
		BackendAssignsTickets:        true,
		BackendDeletesTickets:        false,
		Ticket:                       r.ticket,
		Profiles:                     r.profiles,
		FrontendWaitForAssignment:    true,
		FrontendDeletesTickets:       true,
	}
}

func (r *res) profiles() []*pb.MatchProfile {
	p := []*pb.MatchProfile{}

	for region := 0; region < r.regions; region++ {
		for mode := range r.modePopulations {
			for i := 0; i+1 < len(r.skillBoundaries); i++ {
				skillMin := r.skillBoundaries[i] - r.maxSkillDifference/2
				skillMax := r.skillBoundaries[i+1] + r.maxSkillDifference/2
				p = append(p, &pb.MatchProfile{
					Name: fmt.Sprintf("region_%d_%s_%v-%v", region, mode, skillMin, skillMax),
					Pools: []*pb.Pool{
						{
							Name: poolName,
							DoubleRangeFilters: []*pb.DoubleRangeFilter{
								{
									DoubleArg: "skill",
									Min:       skillMin,
									Max:       skillMax,
								},
							},
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
						},
					},
				})
			}
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
		if v < 0 {
			mode = m
			break
		}
	}

	region := rand.Intn(r.regions)
	numRegions := rand.Intn(r.maxRegions) + 1

	tags := []string{}
	for i := 0; i < numRegions; i++ {
		tags = append(tags, fmt.Sprintf("region_%d", region))
		// The Earth is actually a circle.
		region = (region + 1) % r.regions
	}

	return &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"skill": clamp(rand.NormFloat64(), -3, 3),
			},
			StringArgs: map[string]string{
				"mode": mode,
			},
			Tags: tags,
		},
	}
}

func (r *res) mmf(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	skill := func(t *pb.Ticket) float64 {
		return t.SearchFields.DoubleArgs["skill"]
	}

	tickets := poolTickets[poolName]
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
