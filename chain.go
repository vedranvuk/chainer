// Copyright 2019 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package chainer

import (
	"context"
	"net/http"
	"sync"

	"github.com/vedranvuk/errorex"
)

var (
	// ErrChainer is the base error of chainer package.
	ErrChainer = errorex.New("chainer")
	// ErrDupName is returned by Append if a duplicate name was specified.
	ErrDupName = ErrChainer.WrapFormat("duplicate name '%s'")
	// ErrInvalidName is returned when an invalid name is specified.
	ErrInvalidName = ErrChainer.WrapFormat("no handler registered under name '%s'")
)

// Chain is a chain of http.Handlers executed in sequential order.
type Chain struct {
	key interface{}

	runmu   sync.Mutex
	links   []http.Handler
	names   map[string]int
	indexes []string

	varmu sync.Mutex
	vars  map[string]interface{}
	err   error
	next  string
}

// New creates a new Chain instance with specified context key.
func New(key interface{}) *Chain {
	p := &Chain{
		key:   key,
		runmu: sync.Mutex{},
		varmu: sync.Mutex{},
		names: make(map[string]int),
		vars:  make(map[string]interface{}),
	}
	return p
}

// Append appends a handler to the chain under a specified name
// which must be unique or ErrDupName sibling is returned.
func (c *Chain) Append(name string, handler http.Handler) error {
	c.runmu.Lock()
	defer c.runmu.Unlock()

	if _, exists := c.names[name]; exists {
		return ErrDupName.WrapArgs(name)
	}
	c.links = append(c.links, handler)
	c.names[name] = len(c.links) - 1
	c.indexes = append(c.indexes, name)
	return nil
}

// Names returns the names of handlers as registered in order
// as they were registered or an empty slice if none registered.
// Names() shares the lock with ServeHTTP.
func (c *Chain) Names() []string {
	c.runmu.Lock()
	defer c.runmu.Unlock()

	r := make([]string, 0, len(c.links))
	for _, name := range c.indexes {
		r = append(r, name)
	}
	return r
}

// Clone clones this chain.
// Possibly to have instances for multiple threads.
func (c *Chain) Clone() *Chain {
	c.runmu.Lock()
	defer c.runmu.Unlock()

	clone := New(c.key)
	for _, link := range c.links {
		clone.links = append(clone.links, link)
	}
	for k, v := range c.vars {
		clone.vars[k] = v
	}
	return clone
}

// SetError records an error and stops chain execution
// once surrently executed handler finishes.
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

// MoveTo moves chain execution point to a handler specified by name.
// The handler specified by name can be further down or back up the chain.
// A handler being executed in a chain can call this function to adjust
// chain execution. Chain execution will continue on specified handler
// once the currently executed handler finishes.
//
// It is entirely possible to enter an infinite loop using this call.
//
// If an error occurs it is returned.
func (c *Chain) MoveTo(name string) error {
	c.varmu.Lock()
	defer c.varmu.Unlock()
	if _, exists := c.names[name]; !exists {
		return ErrInvalidName.WrapArgs(name)
	}
	c.next = name
	return nil
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
		// Execute link supporting nested Chains.
		chain, ok := c.links[i].(*Chain)
		if ok {
			chain.ServeHTTP(w, r)
			c.SetError(chain.LastError())
		} else {
			c.links[i].ServeHTTP(w, r)
		}
		// Process MoveTo.
		c.varmu.Lock()
		if c.next != "" {
			i = c.names[c.next] - 1
			c.next = ""
		}
		c.varmu.Unlock()
	}
}

// Unpack unpacks a chain from a request by key.
// Returns a chain and a truth if it exists, which if false, chain will be nil.
func Unpack(r *http.Request, key interface{}) (chain *Chain, exists bool) {
	chain, exists = r.Context().Value(key).(*Chain)
	return
}
