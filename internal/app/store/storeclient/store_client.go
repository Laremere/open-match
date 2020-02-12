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

package storeclient

import (
	"context"

	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/internal/rpc"
)

type client struct {
	cacher *config.Cacher
}

func FromCfg(cfg config.View) ipb.StoreClient {
	newInstance := func(cfg config.View) (interface{}, func(), error) {
		conn, err := rpc.GRPCClientFromConfig(cfg, "api.store")
		if err != nil {
			return nil, nil, err
		}

		closer := func() {
			conn.Close()
		}

		return ipb.NewStoreClient(conn), closer, nil
	}

	return &client{
		cacher: config.NewCacher(cfg, newInstance),
	}
}

func (c *client) Firehose(ctx context.Context, in *ipb.FirehoseRequest, opts ...grpc.CallOption) (ipb.Store_FirehoseClient, error) {
	store, err := c.cacher.Get()
	if err != nil {
		return nil, err
	}
	return store.(ipb.StoreClient).Firehose(ctx, in, opts...)
}

func (c *client) CreateTicket(ctx context.Context, in *ipb.CreateTicketRequest, opts ...grpc.CallOption) (*ipb.CreateTicketResponse, error) {
	store, err := c.cacher.Get()
	if err != nil {
		return nil, err
	}
	return store.(ipb.StoreClient).CreateTicket(ctx, in, opts...)
}

func (c *client) GetTicket(ctx context.Context, in *ipb.GetTicketRequest, opts ...grpc.CallOption) (*ipb.GetTicketResponse, error) {
	store, err := c.cacher.Get()
	if err != nil {
		return nil, err
	}
	return store.(ipb.StoreClient).GetTicket(ctx, in, opts...)
}

func (c *client) AssignTickets(ctx context.Context, in *ipb.AssignTicketsRequest, opts ...grpc.CallOption) (*ipb.AssignTicketsResponse, error) {
	store, err := c.cacher.Get()
	if err != nil {
		return nil, err
	}
	return store.(ipb.StoreClient).AssignTickets(ctx, in, opts...)
}

func (c *client) ReleaseTickets(ctx context.Context, in *ipb.ReleaseTicketsRequest, opts ...grpc.CallOption) (*ipb.ReleaseTicketsResponse, error) {
	store, err := c.cacher.Get()
	if err != nil {
		return nil, err
	}
	return store.(ipb.StoreClient).ReleaseTickets(ctx, in, opts...)
}

func (c *client) MarkPending(ctx context.Context, in *ipb.MarkPendingRequest, opts ...grpc.CallOption) (*ipb.MarkPendingResponse, error) {
	store, err := c.cacher.Get()
	if err != nil {
		return nil, err
	}
	return store.(ipb.StoreClient).MarkPending(ctx, in, opts...)
}

func (c *client) GetCurrentWatermark(ctx context.Context, in *ipb.GetCurrentWatermarkRequest, opts ...grpc.CallOption) (*ipb.GetCurrentWatermarkResponse, error) {
	store, err := c.cacher.Get()
	if err != nil {
		return nil, err
	}
	return store.(ipb.StoreClient).GetCurrentWatermark(ctx, in, opts...)
}

func (c *client) DeleteTicket(ctx context.Context, in *ipb.DeleteTicketRequest, opts ...grpc.CallOption) (*ipb.DeleteTicketResponse, error) {
	store, err := c.cacher.Get()
	if err != nil {
		return nil, err
	}
	return store.(ipb.StoreClient).DeleteTicket(ctx, in, opts...)
}
