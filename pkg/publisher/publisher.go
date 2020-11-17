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

// Package publisher contains the code to talk to Google Pub/Sub.
package publisher

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"regexp"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/bobvawter/latch"
	"github.com/pkg/errors"
)

// A Publisher is used to manage Topic instances.
type Publisher struct {
	client *pubsub.Client
	keys   map[string]bool
	latch  *latch.Counter
	mu     struct {
		sync.Mutex
		topics map[string]*maybeTopic
	}
	pathPrefixLen int
	topicPrefix   topicPrefix
}

type maybeTopic struct {
	sync.Mutex
	*pubsub.Topic
}

// New constructs a new Publisher around the given Client.
func New(client *pubsub.Client, options ...Option) *Publisher {
	p := &Publisher{
		client: client,
		keys:   map[string]bool{},
	}
	p.mu.topics = make(map[string]*maybeTopic)
	for _, opt := range options {
		switch t := opt.(type) {
		case counter:
			p.latch = t.Counter
		case mux:
			t.mux.HandleFunc(t.prefix, p.http)
			p.pathPrefixLen = len(t.prefix)
		case sharedKeys:
			p.keys = make(map[string]bool, len(t))
			for i := range t {
				p.keys[t[i]] = true
			}
		case topicPrefix:
			p.topicPrefix = t
		default:
			panic(errors.Errorf("unimplemented %t", t))
		}
	}
	return p
}

// Publish will send the given message and attributes to the Pub/Sub
// endpoint.
func (p *Publisher) Publish(ctx context.Context, table string, data []byte, attrs map[string]string) (*pubsub.PublishResult, error) {
	if p.topicPrefix != "" {
		table = string(p.topicPrefix) + table
	}
	t, err := p.ensureTopic(ctx, table)
	if err != nil {
		return nil, err
	}

	msg := &pubsub.Message{
		Data:       data,
		Attributes: attrs,
	}

	return t.Publish(ctx, msg), nil
}

// ensureTopic finds or creates a pubsub.Topic with the given name.  It
// ensures that only a single instance of the pubsub.Topic is actually
// created for the given name.
func (p *Publisher) ensureTopic(ctx context.Context, topic string) (*pubsub.Topic, error) {
	p.mu.Lock()
	t := p.mu.topics[topic]
	if t == nil {
		t = &maybeTopic{}
		p.mu.topics[topic] = t
	}
	p.mu.Unlock()

	t.Lock()
	defer t.Unlock()

	if t.Topic == nil {
		temp := p.client.Topic(topic)
		if exists, err := temp.Exists(ctx); err != nil {
			return nil, errors.Wrapf(err, "cold not check existence of %q", topic)
		} else if exists {
			t.Topic = temp
			log.Printf("using existing topic %q", t.Topic)
		} else {
			temp, err := p.client.CreateTopic(ctx, topic)
			if err != nil {
				return nil, errors.Wrapf(err, "could not create topic %q", topic)
			}
			t.Topic = temp
			log.Printf("created topic %q", t.Topic)
		}
	}

	return t.Topic, nil
}

// From https://www.cockroachlabs.com/docs/v20.2/create-changefeed#files
var (
	generalFile  = regexp.MustCompile(`/([^/]*)/(\d{4}-\d{2}-\d{2})/(\d{33})-(.+)-([^-]+)-([^-]+).ndjson$`)
	resolvedFile = regexp.MustCompile(`/([^/]*)/(\d{4}-\d{2}-\d{2})/(\d{33).RESOLVED$`)
)

func (p *Publisher) http(w http.ResponseWriter, req *http.Request) {
	if p.latch != nil {
		p.latch.Hold()
		defer p.latch.Release()
	}

	if len(p.keys) > 0 {
		found := req.URL.Query().Get("sharedKey")
		if found == "" || !p.keys[found] {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("invalid sharedKey parameter from %s", req.RemoteAddr)
			return
		}
	}

	table := ""
	topic := ""

	if parts := resolvedFile.FindStringSubmatch(req.URL.Path); parts != nil {
		table = "RESOLVED"
		topic = parts[1]
	} else if parts := generalFile.FindStringSubmatch(req.URL.Path); parts != nil {
		table = parts[5]
		topic = parts[1]
	} else {
		w.WriteHeader(http.StatusNotFound)
		log.Printf("unexpected filename %q", req.URL.Path)
		return
	}

	var err error
	defer func() {
		if err == nil {
			w.WriteHeader(http.StatusCreated)
			return
		}
		log.Printf("%s: %v", req.URL, err)
		w.WriteHeader(http.StatusInternalServerError)
	}()

	attrs := map[string]string{
		"path":  req.URL.Path,
		"table": table,
	}

	s := bufio.NewScanner(req.Body)
	var futures []*pubsub.PublishResult

	for s.Scan() {
		var future *pubsub.PublishResult
		future, err = p.Publish(req.Context(), topic, s.Bytes(), attrs)
		if err != nil {
			return
		}
		futures = append(futures, future)
	}

	for _, future := range futures {
		if id, ferr := future.Get(req.Context()); ferr == nil {
			log.Printf("published topic: %s table: %s id: %s", topic, table, id)
		} else {
			log.Printf("failed to publish: topic: %s table: %s id:%s %v", topic, table, id, ferr)
			if err != nil {
				err = ferr
			}
		}
	}
}
