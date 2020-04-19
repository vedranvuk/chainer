// Copyright 2019 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package chainer

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/vedranvuk/testex"
)

var testkey = "chainer"

var verbose = testex.Verbose()

func MakeRequest(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

func MakeHandler(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Handler '%s' reporting in.\n", name)
	})
}

func MakeHandlerThatSetsAnError(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Handler '%s' is setting an error!\n", name)
		chain, exists := Unpack(r, testkey)
		if !exists {
			panic("nope")
		}
		chain.SetError(fmt.Errorf("Handler '%s' error.", name))
	})
}

func MakeHandlerThatMovesToAHandler(name, moveto string, t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Handler '%s' is moving to Handler '%s'!\n", name, moveto)
		chain, exists := Unpack(r, testkey)
		if !exists {
			panic("nope")
		}
		if err := chain.MoveTo("non-existent handler"); !errors.Is(err, ErrInvalidName) {
			t.Fatal("MoveTO() failed")
		}
		chain.MoveTo(moveto)
	})
}

func TestRegistration(t *testing.T) {

	const chainlength = 5
	const varlength = 5

	reggedhandlers := make([]string, 0, chainlength)
	for i := 0; i < chainlength; i++ {
		reggedhandlers = append(reggedhandlers, fmt.Sprintf("Handler %d", i))
	}

	checkchain := func(c *Chain) {
		for i := 0; i < varlength; i++ {
			c.Set(fmt.Sprintf("var %d", i), i)
		}
		for i := 0; i < varlength; i++ {
			v, ok := c.Get(fmt.Sprintf("var %d", i))
			if !ok {
				t.Fatal("Get/Set failed")
			}
			if v != i {
				t.Fatal("Get/Set failed")
			}
		}
		for _, name := range reggedhandlers {
			if err := c.Append(name, MakeHandler(name)); err != nil {
				t.Fatal("Append() failed")
			}
		}
		for _, name := range reggedhandlers {
			if err := c.Append(name, MakeHandler(name)); !errors.Is(err, ErrDupName) {
				t.Fatal("Append() failed")
			}
		}
		names := c.Names()
		if len(names) != len(reggedhandlers) {
			t.Fatal("Append() failed")
		}
		for index, name := range names {
			if name != reggedhandlers[index] {
				t.Fatal("Append() failed")
			}
		}
	}

	c := New(testkey)
	checkchain(c)
	clone := c.Clone()
	checkchain(clone)
}

func TestChain(t *testing.T) {

	const want = `FakeResponseWriter: Handler 'h1' reporting in.
FakeResponseWriter: Handler 'h2' reporting in.
FakeResponseWriter: Handler 'h3' reporting in.
`

	buf := bytes.NewBuffer(nil)
	c := New(testkey)
	c.Append("h1", MakeHandler("h1"))
	c.Append("h2", MakeHandler("h2"))
	c.Append("h3", MakeHandler("h3"))
	c.ServeHTTP(testex.NewFakeResponseWriter(buf), makeRequest("/"))
	if verbose {
		fmt.Printf(string(buf.Bytes()))
	}
	if string(buf.Bytes()) != want {
		t.Fatal("TestChain() failed")
	}
}

func TestError(t *testing.T) {

	const want = `FakeResponseWriter: Handler 'h1' reporting in.
FakeResponseWriter: Handler 'h2' is setting an error!
`

	buf := bytes.NewBuffer(nil)
	c := New(testkey)
	c.Append("h1", MakeHandler("h1"))
	c.Append("h2", MakeHandlerThatSetsAnError("h2"))
	c.Append("h3", MakeHandler("h3"))
	c.ServeHTTP(testex.NewFakeResponseWriter(buf), makeRequest("/"))
	if verbose {
		fmt.Printf(string(buf.Bytes()))
	}
	if string(buf.Bytes()) != want {
		t.Fatal("TestError() failed")
	}
}

func TestMove(t *testing.T) {

	const want = `FakeResponseWriter: Handler 'h1' is moving to Handler 'h3'!
FakeResponseWriter: Handler 'h3' reporting in.
`

	buf := bytes.NewBuffer(nil)
	c := New(testkey)
	c.Append("h1", MakeHandlerThatMovesToAHandler("h1", "h3", t))
	c.Append("h2", MakeHandlerThatSetsAnError("h2"))
	c.Append("h3", MakeHandler("h3"))
	c.ServeHTTP(testex.NewFakeResponseWriter(buf), makeRequest("/"))
	if verbose {
		fmt.Printf(string(buf.Bytes()))
	}
	if string(buf.Bytes()) != want {
		t.Fatal("TestMove() failed")
	}
}

func TestNested(t *testing.T) {

	const want = `FakeResponseWriter: Handler 'chain 1 handler 1' reporting in.
FakeResponseWriter: Handler 'chain 1 handler 2' reporting in.
FakeResponseWriter: Handler 'chain 1 handler 3' reporting in.
FakeResponseWriter: Handler 'chain 2 handler 1' reporting in.
FakeResponseWriter: Handler 'chain 2 handler 2' reporting in.
FakeResponseWriter: Handler 'chain 2 handler 3' reporting in.
FakeResponseWriter: Handler 'chain 3 handler 1' reporting in.
FakeResponseWriter: Handler 'chain 3 handler 2' reporting in.
FakeResponseWriter: Handler 'chain 3 handler 3' reporting in.
`

	buf := bytes.NewBuffer(nil)

	ch1 := New(testkey)
	ch1.Append("chain 1 handler 1", MakeHandler("chain 1 handler 1"))
	ch1.Append("chain 1 handler 2", MakeHandler("chain 1 handler 2"))
	ch1.Append("chain 1 handler 3", MakeHandler("chain 1 handler 3"))

	ch2 := New(testkey)
	ch2.Append("chain 2 h1", MakeHandler("chain 2 handler 1"))
	ch2.Append("chain 2 h2", MakeHandler("chain 2 handler 2"))
	ch2.Append("chain 2 h3", MakeHandler("chain 2 handler 3"))

	ch3 := New(testkey)
	ch3.Append("chain 3 h1", MakeHandler("chain 3 handler 1"))
	ch3.Append("chain 3 h2", MakeHandler("chain 3 handler 2"))
	ch3.Append("chain 3 h3", MakeHandler("chain 3 handler 3"))

	ch := New(testkey)
	ch.Append("chain 1", ch1)
	ch.Append("chain 2", ch2)
	ch.Append("chain 3", ch3)

	ch.ServeHTTP(testex.NewFakeResponseWriter(buf), MakeRequest("/"))

	if verbose {
		fmt.Printf(string(buf.Bytes()))
	}
	if string(buf.Bytes()) != want {
		t.Fatal("TestNested() failed")
	}
}

func TestNestedError(t *testing.T) {

	const want = `FakeResponseWriter: Handler 'Handler 1' reporting in.
FakeResponseWriter: Handler 'Child handler 1' reporting in.
FakeResponseWriter: Handler 'Child Handler 2' is setting an error!
`

	buf := bytes.NewBuffer(nil)

	ch := New(testkey)
	ch.Append("Handler 1", MakeHandler("Handler 1"))
	nc := New(testkey)
	nc.Append("Child Handler 1", MakeHandler("Child handler 1"))
	nc.Append("Child Handler 2", MakeHandlerThatSetsAnError("Child Handler 2"))
	nc.Append("Child Handler 3", MakeHandler("Child handler 3"))
	ch.Append("Handler 2", nc)
	ch.Append("Handler 3", MakeHandler("Handler 3"))
	ch.ServeHTTP(testex.NewFakeResponseWriter(buf), MakeRequest("/"))

	if verbose {
		fmt.Printf(string(buf.Bytes()))
	}
	if string(buf.Bytes()) != want {
		t.Fatal("TestNested() failed")
	}
}
