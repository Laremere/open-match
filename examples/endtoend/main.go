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

// Package main TODO
package main

//  "open-match.dev/open-match/internal/pb"
import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
	"open-match.dev/open-match/examples/endtoend/dashboard"
	"open-match.dev/open-match/examples/endtoend/demo"
	"open-match.dev/open-match/examples/endtoend/listen"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "endtoend",
	})
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	logging.ConfigureLogging(cfg)

	logger.Info("Initializing Server")

	updates := make(chan func(*dashboard.Dashboard))

	l := listen.New()
	go relayDashboard(updates, l.AnnounceLatest)
	_ = demo.New(updates)

	fileServe := http.FileServer(http.Dir("/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fileServe))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dashboard/" {
			http.NotFound(w, r)
			return
		}
		fileServe.ServeHTTP(w, r)
	})

	http.Handle("/dashboard/connect", websocket.Handler(l.HandleSubscriber))

	logger.Info("Starting Server")
	// TODO: Other services read their port from the common config map (oddly including
	// the mmf, which isn't part of the core open-match), how should this be choosing the ports
	// it exposes?
	err = http.ListenAndServe(":3389", nil)
	logger.WithFields(logrus.Fields{
		"err": err.Error(),
	}).Fatal("Http ListenAndServe failed.")
}

func relayDashboard(updates chan func(*dashboard.Dashboard), announce func([]byte)) {
	d := dashboard.New()

	for f := range updates {
		f(d)

	applyAllWaitingUpdates:
		for {
			select {
			case f := <-updates:
				f(d)
			default:
				break applyAllWaitingUpdates
			}
		}

		b, err := json.MarshalIndent(d, "", "  ")
		if err == nil {
			announce(b)
		} else {
			logger.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("Error encoding dashboard state.")
		}
	}
}
