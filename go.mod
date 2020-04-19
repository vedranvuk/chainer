module github.com/vedranvuk/chainer

go 1.14

require (
	github.com/vedranvuk/errorex v0.3.0
	github.com/vedranvuk/testex v0.0.0-20200419092702-c64516cc5e9b
)

replace (
	github.com/vedranvuk/errorex => ../errorex
	github.com/vedranvuk/testex => ../testex
)
