// +build e2ecluster

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

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"

	// "google.golang.org/grpc/resolver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"open-match.dev/open-match/internal/app/evaluator"
	"open-match.dev/open-match/internal/appmain/apptest"
	"open-match.dev/open-match/internal/config"
	mmfService "open-match.dev/open-match/internal/testing/mmf"
	"open-match.dev/open-match/pkg/pb"
	"strings"
)

// func TestServiceHealth(t *testing.T) {
// 	om := e2e.New(t)
// 	if err := om.HealthCheck(); err != nil {
// 		t.Errorf("cluster health checks failed, %s", err)
// 	}
// }

func TestMain(m *testing.M) {
	clusterStarted = true
	mmf := func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		return clusterMMF(ctx, profile, out)
	}

	eval := func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		return clusterEval(ctx, in, out)
	}

	cleanup, err := apptest.RunInCluster(mmfService.BindServiceFor(mmf), evaluator.BindServiceFor(eval))
	if err != nil {
		fmt.Println("Error starting mmf and evaluator:", err)
		os.Exit(1)
	}

	exitCode := m.Run()

	err = cleanup()
	if err != nil {
		fmt.Println("Error stopping mmf and evaluator:", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

// TestConfigMatch covers that the config file used for local in memory e2e
// tests matches the configs used for the in cluster tests, to avoid drift.
func TestConfigMatch(t *testing.T) {
	cfg, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}

	cfgMemory := viper.New()
	cfgMemory.SetConfigType("yaml")
	err = cfgMemory.ReadConfig(strings.NewReader(configFile))
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, cfgMemory.AllSettings(), cfg.AllSettings())
}