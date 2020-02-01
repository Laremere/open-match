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

package minimatch

import (
	"open-match.dev/open-match/internal/app/backend"
	"open-match.dev/open-match/internal/app/frontend"
	"open-match.dev/open-match/internal/app/query"
	"open-match.dev/open-match/internal/app/store"
	"open-match.dev/open-match/internal/app/synchronizer"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
)

var services = []func(*rpc.ServerParams, config.View) error{
	backend.BindService,
	frontend.BindService,
	query.BindService,
	synchronizer.BindService,
	store.BindService,
}

// BindService creates the minimatch service to the server Params.
func BindService(p *rpc.ServerParams, cfg config.View) error {
	for _, bind := range services {
		err := bind(p, cfg)
		if err != nil {
			return err
		}
	}

	return nil
}
