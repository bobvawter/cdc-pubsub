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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bobvawter/cdc-pubsub/pkg/server"
	"github.com/spf13/pflag"
)

func main() {
	s := &server.Server{}

	flags := pflag.FlagSet{SortFlags: true}
	flags.StringVar(&s.BindAddr, "bindAddr", ":13013", "the address to bind to")
	flags.StringVar(&s.CredentialsFile, "credentials", "cdc-pubsub.json", "a JSON-formatted Google Cloud credentials file")
	flags.DurationVar(&s.GracePeriod, "gracePeriod", 30*time.Second, "shutdown grace period")
	flags.StringVar(&s.ProjectID, "projectID", "", "the Google Cloud project ID")
	flags.StringSliceVar(&s.SharedKeys, "sharedKey", nil, "require clients to provide one of these secret values")
	flags.StringVar(&s.TopicPrefix, "topicPrefix", "", "a prefix to add to topic names")

	help := flags.BoolP("help", "h", false, "display this message")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Printf("%v", err)
		println(flags.FlagUsagesWrapped(100))
		os.Exit(1)
	} else if *help {
		println(flags.FlagUsagesWrapped(100))
		os.Exit(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch
		log.Printf("signal received, shutting down")
		cancel()
	}()

	if err := s.ListenAndServe(ctx); ctx.Err() == context.Canceled {
		log.Printf("goodbye")
		os.Exit(0)
	} else {
		log.Printf("%v", err)
		os.Exit(1)
	}
}
