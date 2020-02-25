// Copyright 2019 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package chainer

import (
	"context"
	"net/http"
	"sync"
)

// Chain is a chain of http.Handlers executed in sequential order.
type Chain struct {
	key   interface{}
	runmu sync.Mutex
	varmu sync.Mutex
	links []http.Handler
	vars  map[string]interface{}
	err   error
}

// New creates a new Chain instance with specified context key.
func New(key interface{}) *Chain {
	p := &Chain{
		key:   key,
		runmu: sync.Mutex{},
		varmu: sync.Mutex{},
		vars:  make(map[string]interface{}),
	}
	return p
}

// Add adds a handler to the chain.
func (c *Chain) Add(handler http.Handler) *Chain {
	c.runmu.Lock()
	defer c.runmu.Unlock()

	c.links = append(c.links, handler)
	return c
}

// Clone clones this chain.
func (c *Chain) Clone() *Chain {
	c.runmu.Lock()
	defer c.runmu.Unlock()

	clone := New(c.key)
	clone.links = c.links
	for k, v := range c.vars {
		clone.vars[k] = v
	}
	return clone
}

// SetError records an error and stops chain execution.
func (c *Chain) SetError(err error) {
	c.varmu.Lock()
	defer c.varmu.Unlock()

	c.err = err
}

// LastError returns last recorded error, if any.
func (c *Chain) LastError() error {
	c.varmu.Lock()
	defer c.varmu.Unlock()

	return c.err
}

// Get gets a context variable by key and returns it as interface and
// a truth if it exists.
func (c *Chain) Get(key string) (val interface{}, ok bool) {
	c.varmu.Lock()
	defer c.varmu.Unlock()

	val, ok = c.vars[key]
	return
}

// Set sets a context variable by key to val.
func (c *Chain) Set(key string, val interface{}) {
	c.varmu.Lock()
	defer c.varmu.Unlock()

	c.vars[key] = val
}

// ServeHTTP passes w and r across the handler chain.
// If a handler sets Chain error during execution, loop is aborted.
// Chained handlers are checked if they are Chains themselves. If an error
// occurs in such chain, the error is propagated to the top chain.
func (c *Chain) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	c.runmu.Lock()
	defer c.runmu.Unlock()

	c.SetError(nil)
	r = r.Clone(context.WithValue(r.Context(), c.key, c))
	for i := 0; i < len(c.links) && c.LastError() == nil; i++ {
		c.links[i].ServeHTTP(w, r)

		// Support nested chains.
		if c.LastError() != nil {
			break
		}
		chain, ok := c.links[i].(*Chain)
		if !ok {
			continue
		}
		chain.ServeHTTP(w, r)
		c.SetError(chain.LastError())
	}
}

// Unpack unpacks a chain from a request by key.
// Returns a chain and a truth if it exists, which if false, chain will be nil.
func Unpack(r *http.Request, key interface{}) (chain *Chain, exists bool) {
	chain, exists = r.Context().Value(key).(*Chain)
	return
}
