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

package publisher

import (
	"net/http"

	"github.com/bobvawter/latch"
)

type option struct{}

// An Option provides additional configuration to the Publisher.
type Option interface {
	option() option
}

// WithHTTPCounter uses the given counter to signal if there are any
// in-flight http requests.
func WithHTTPCounter(l *latch.Counter) Option {
	return counter{l}
}

type counter struct {
	*latch.Counter
}

func (counter) option() option { return option{} }

// WithHTTPMux will register handlers in the given ServeMux to accept
// incoming connections and publish them.
func WithHTTPMux(m *http.ServeMux, pathPrefix string) Option {
	return mux{m, pathPrefix}
}

// WithSharedKeys provides a limited degree of authentication between
// the bridge server and the CDC client.
func WithSharedKeys(keys []string) Option {
	return sharedKeys(keys)
}

type sharedKeys []string

func (sharedKeys) option() option { return option{} }

type mux struct {
	mux    *http.ServeMux
	prefix string
}

func (mux) option() option { return option{} }

// WithTopicPrefix will prepend the given prefix to the table name.
func WithTopicPrefix(prefix string) Option {
	return topicPrefix(prefix)
}

type topicPrefix string

func (topicPrefix) option() option { return option{} }
