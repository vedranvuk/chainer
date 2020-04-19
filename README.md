# chainer

Chainer is a http.Handler chainer implemented as a Handler itself.

It can contain one or more handlers which it executes in order as they're registered. Registered handlers can be chains themselves. An error can be returned from a handler being executed to stop chain execution and is propagated from a handler at any depth.

Handlers can have the reference to the chain that contains them via request context under a user defined key.

Several other utility functions such as cloning, execution order modification at execution time, and a built-in value storage are provided.

## Example

```
// A secret key for request context passing the chain reference to a handler in a chain.
var mykey = 42

// A test handler to put in a chain.
type TestHandler struct {
	prefix string
}

// Makes a new test handler and sets its prefix to make it unique.
func newTestHandler(prefix string) *TestHandler { return &TestHandler{prefix} }

func (th *TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Output a s tring with the prefix.
	fmt.Printf("Hello from handler %s\n", th.prefix)

	// Throw an error on the last handler, for science.
	if th.prefix == "c" {
		if chain, exists := Unpack(r, mykey); exists {
			chain.SetError(fmt.Errorf("let me throw an error just for kicks"))
		}
	}
}

func ExampleChain() {

	// Create a chain and push some handlers to the chain.
	chain := New(mykey)
	chain.Append(newTestHandler("a"))
	chain.Append(newTestHandler("b"))
	chain.Append(newTestHandler("c"))

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
```

## License

MIT, see included LICENSE file.