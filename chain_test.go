// Copyright 2019 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package chainer

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/vedranvuk/testex"
)

var key = 42

var frw = testex.NewFakeResponseWriter()

func testHandler(w http.ResponseWriter, r *http.Request) {

}

func handlerHead(w http.ResponseWriter, r *http.Request) {
	fmt.Println("HeadHandler")
	fmt.Fprintln(w, "Head")
}

func handlerBody(w http.ResponseWriter, r *http.Request) {
	fmt.Println("BodyHandler")
	fmt.Fprintln(w, "Body")
	chain, ok := r.Context().Value(key).(*Chain)
	if !ok {
		panic("No chain found in context.")
	}
	chain.SetError(errors.New("test error"))
}

func handlerTail(w http.ResponseWriter, r *http.Request) {
	fmt.Println("TailHandler")
	fmt.Fprintln(w, "Tail")
}

func TestChain(t *testing.T) {
	c := New(key)
	c.Add(http.HandlerFunc(handlerHead)).Add(http.HandlerFunc(handlerBody)).Add(http.HandlerFunc(handlerTail)).ServeHTTP(frw, makeRequest())

	fmt.Printf("Last error: %v\n", c.LastError())
}

func makeHandler(index, level int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Thread: %s, Index: %d, Child: %d.\n", r.URL.Path[1:], index, level)
	})
}

func makeRequest() *http.Request {
	req, _ := http.NewRequest("GET", "/", nil)
	return req
}

func TestLevels(t *testing.T) {
	c1 := New(key).Add(makeHandler(0, 0)).Add(makeHandler(0, 1)).Add(makeHandler(0, 2))
	c2 := New(key).Add(makeHandler(1, 0)).Add(makeHandler(1, 1)).Add(makeHandler(1, 2))
	c3 := New(key).Add(makeHandler(2, 0)).Add(makeHandler(2, 1)).Add(makeHandler(2, 2))
	c := New(key).Add(c1).Add(c2).Add(c3)
	c.ServeHTTP(frw, makeRequest())
}

func TestConcurrency(t *testing.T) {

	c1 := New(key).Add(makeHandler(0, 0)).Add(makeHandler(0, 1)).Add(makeHandler(0, 2))
	c2 := New(key).Add(makeHandler(1, 0)).Add(makeHandler(1, 1)).Add(makeHandler(1, 2))
	c3 := New(key).Add(makeHandler(2, 0)).Add(makeHandler(2, 1)).Add(makeHandler(2, 2))
	c := New(key).Add(c1).Add(c2).Add(c3)

	go func() {
		if err := http.ListenAndServe(":8085", c); err != nil {
			panic(err)
		}
	}()

	time.Sleep(10 * time.Millisecond)

	for i := 0; i < runtime.NumCPU(); i++ {
		go func(thread int) {
			for {
				resp, err := http.Get(fmt.Sprintf("http://localhost:8085/%d", thread))
				if err != nil {
					panic(err)
				}
				buf, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					panic(err)
				}
				resp.Body.Close()
				fmt.Printf("%s", string(buf))
				// time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	time.Sleep(1 * time.Second)
}
