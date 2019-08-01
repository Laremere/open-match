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

package store

import (
	"context"

	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
)

type service struct {
}

func newStoreService(cfg config.View) *service {
	return &service{}
}

func (s *service) IndexUpdates(r *pb.IndexUpdatesRequest, resp pb.Store_IndexUpdatesServer) error {
	return nil
}

func (s *service) DeindexTicket(ctx context.Context, r *pb.DeindexTicketRequest) (*pb.DeindexTicketResponse, error) {
	return nil, nil
}

func (s *service) CreateTicket(ctx context.Context, r *pb.CreateTicketRequest) (*pb.CreateTicketResponse, error) {
	return nil, nil
}

func (s *service) AssignTickets(ctx context.Context, r *pb.AssignTicketsRequest) (*pb.AssignTicketsResponse, error) {
	return nil, nil
}

func (s *service) AssignmentSubscribe(stream pb.Store_AssignmentSubscribeServer) error {
	return nil
}
