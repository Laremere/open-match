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
	// "encoding/json"
	// "fmt"
	// "io"
	// "net/http"
	// "strings"
	// "sync"

	// "github.com/golang/protobuf/jsonpb"
	"github.com/sirupsen/logrus"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/ipb"
	// "open-match.dev/open-match/internal/rpc"
	// "open-match.dev/open-match/internal/statestore"
	// "open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

type storeService struct {
}

type update struct {
	ready chan struct{}
	next  *update

	newTickets   []*pb.Ticket
	stateUpdates map[string]ipb.State
	watermarks   []*watermark
}

type watermark struct{}

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.store",
	})
)

func (s *storeService) CreateTicket(ctx context.Context, req *ipb.CreateTicketRequest) (*ipb.CreateTicketResponse, error) {
	return nil, nil
}

func (s *storeService) Firehose(stream ipb.Store_FirehoseServer) error {
	return nil
}

func (s *storeService) Freeze(ctx context.Context, req *ipb.FreezeRequest) (*ipb.FreezeResponse, error) {
	return nil, nil
}

func (s *storeService) GetTicket(ctx context.Context, req *ipb.GetTicketRequest) (*ipb.GetTicketResponse, error) {
	return nil, nil
}

func (s *storeService) UpdateState(ctx context.Context, req *ipb.UpdateStateRequest) (*ipb.UpdateStateResponse, error) {
	return nil, nil
}

func (s *storeService) run() {
	// Todo: load from crash file.
	tickets := map[string]*pb.Ticket{}
	states := map[string]ipb.State{}

	for {
		select {}
	}
}
