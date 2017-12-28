build:
	go build github.com/joshi4/goroutine_inspector

test: build
	go test -v -race ./...
