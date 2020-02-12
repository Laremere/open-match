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

package evaluator

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
		"component": "scale.evaluator",
	})
)

// Run triggers execution of an evaluator.
func Run() {
	e := &evaluatorFunction{}

	go scenarios.Run(nil, func(ctx context.Context, cfg config.View, scenario *scenarios.Scenario, qps func() int) {
		e.set(scenario.Evaluator)
	})

	server := grpc.NewServer(utilTesting.NewGRPCServerOptions(logger)...)
	pb.RegisterEvaluatorServer(server, e)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", 50508))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
			"port":  50508,
		}).Fatal("net.Listen() error")
	}

	logger.WithFields(logrus.Fields{
		"port": 50508,
	}).Info("TCP net listener initialized")

	logger.Info("Serving gRPC endpoint")
	err = server.Serve(ln)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("gRPC serve() error")
	}
}

type evaluatorFunction struct {
	f func(pb.Evaluator_EvaluateServer) error
	m sync.RWMutex
}

func (e *evaluatorFunction) Evaluate(srv pb.Evaluator_EvaluateServer) error {
	e.m.RLock()
	defer e.m.RUnlock()

	if e.f == nil {
		return fmt.Errorf("Evaluator function not set")
	}
	return e.f(srv)
}

func (e *evaluatorFunction) set(f func(pb.Evaluator_EvaluateServer) error) {
	e.m.Lock()
	defer e.m.Unlock()

	e.f = f
}
