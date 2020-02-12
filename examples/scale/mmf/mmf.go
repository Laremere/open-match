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

package mmf

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"open-match.dev/open-match/examples/scale/scenarios"
	"open-match.dev/open-match/internal/config"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "scale.mmf",
	})
)

// Run triggers execution of a MMF.
func Run() {
	m := &matchFunction{}

	logger.Warning("Starting mmf")

	go scenarios.Run(nil, func(ctx context.Context, cfg config.View, scenario *scenarios.Scenario, qps func() int) {
		logger.Warning("Setting mmf")
		m.set(scenario.MMF)
	})

	conn, err := grpc.Dial("om-query.open-match.svc.cluster.local:50503", utilTesting.NewGRPCDialOptions(logger)...)
	if err != nil {
		logger.Fatalf("Failed to connect to Open Match, got %v", err)
	}
	defer conn.Close()

	server := grpc.NewServer(utilTesting.NewGRPCServerOptions(logger)...)
	pb.RegisterMatchFunctionServer(server, m)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", 50502))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
			"port":  50502,
		}).Fatal("net.Listen() error")
	}

	logger.WithFields(logrus.Fields{
		"port": 50502,
	}).Info("TCP net listener initialized")

	logger.Info("Serving gRPC endpoint")
	err = server.Serve(ln)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("gRPC serve() error")
	}
}

type matchFunction struct {
	f func(*pb.RunRequest, pb.MatchFunction_RunServer) error
	m sync.RWMutex
}

func (m *matchFunction) Run(req *pb.RunRequest, srv pb.MatchFunction_RunServer) error {
	m.m.RLock()
	defer m.m.RUnlock()

	if m.f == nil {
		return fmt.Errorf("Match function not set")
	}
	return m.f(req, srv)
}

func (m *matchFunction) set(f func(*pb.RunRequest, pb.MatchFunction_RunServer) error) {
	logger.WithFields(logrus.Fields{
		"mmf": f,
	}).Warning("Updating mmf to value")
	m.m.Lock()
	defer m.m.Unlock()

	m.f = f
}
