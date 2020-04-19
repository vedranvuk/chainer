// Copyright 2019 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package chainer

import (
	"fmt"
	"net/http"

	"github.com/vedranvuk/testex"
)

var mykey = 42

type TestHandler struct {
	prefix string
}

func newTestHandler(prefix string) *TestHandler { return &TestHandler{prefix} }

func (th *TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Hello from handler %s\n", th.prefix)

	if th.prefix == "c" {
		if chain, exists := Unpack(r, mykey); exists {
			chain.SetError(fmt.Errorf("let me throw an error just for kicks"))
		}
	}
}

func makeRequest(path string) *http.Request {
	r, _ := http.NewRequest("GET", path, nil)
	return r
}

func ExampleChain() {

	// Create a chain and push some handlers to the chain.
	chain := New(mykey)
	chain.Append("a", newTestHandler("a"))
	chain.Append("b", newTestHandler("b"))
	chain.Append("c", newTestHandler("c"))

	// Create a request for the chain to process.
	req, _ := http.NewRequest("GET", "/", nil)

	// Process the request and write to a fake writer that writes to stdout.
	chain.ServeHTTP(testex.NewFakeResponseWriter(), req)

	// Check for errors durring execution.
	if err := chain.LastError(); err != nil {
		fmt.Printf("Got an error: %v\n", err)
	}

	// Output: Hello from handler a
	// Hello from handler b
	// Hello from handler c
	// Got an error: let me throw an error just for kicks
}
