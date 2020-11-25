// Copyright 2020 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Package server contains the bridge server.
package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/bobvawter/cdc-pubsub/pkg/publisher"
	"github.com/bobvawter/latch"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
)

// Server contains the configuration and main HTTP loop.
type Server struct {
	BindAddr        string
	CredentialsFile string
	DumpOnly        bool
	GracePeriod     time.Duration
	ProjectID       string
	SharedKeys      []string
	TopicPrefix     string
}

// ListenAndServe blocks until the context is cancelled and all in-flight requests have been processed.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if s.ProjectID == "" {
		return errors.New("must specify a project ID")
	}

	var client *pubsub.Client
	var err error
	if !s.DumpOnly {
		client, err = pubsub.NewClient(ctx, s.ProjectID, option.WithCredentialsFile(s.CredentialsFile))
		if err != nil {
			log.Fatal(err)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, request *http.Request) {
		if ctx.Err() == nil {
			writer.Header().Set("content-type", "text/plain")
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte("OK"))
		} else {
			writer.WriteHeader(http.StatusServiceUnavailable)
		}
	})

	ctr := latch.New()
	publisher.New(client,
		publisher.WithHTTPCounter(ctr),
		publisher.WithHTTPMux(mux, "/v1/"),
		publisher.WithSharedKeys(s.SharedKeys),
		publisher.WithTopicPrefix(s.TopicPrefix))

	l, err := net.Listen("tcp", s.BindAddr)
	if err != nil {
		return err
	}
	log.Printf("listening on %s", l.Addr())
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	err = http.Serve(l, mux)
	select {
	case <-ctr.Wait():
	// OK
	case <-time.After(s.GracePeriod):
		log.Printf("grace period %s expired", s.GracePeriod)
	}
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
