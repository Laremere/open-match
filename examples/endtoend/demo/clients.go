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

// Package demo contains
package demo

import (
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
)

type Clients struct {
	BE pb.BackendClient
	FE pb.FrontendClient
}

func NewClients(cfg config.View) (*Clients, error) {
	c := &Clients{}

	{
		conn, err := rpc.GRPCClientFromConfig(cfg, "api.backend")
		if err != nil {
			return nil, err
		}
		c.BE = pb.NewBackendClient(conn)
	}

	{
		conn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
		if err != nil {
			// TODO: if this fails, BE client isn't closed??
			return nil, err
		}
		c.FE = pb.NewFrontendClient(conn)
	}

	// TODO: How to close all connections?

	return c, nil
}
