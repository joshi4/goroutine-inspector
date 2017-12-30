build:
	go build github.com/joshi4/goroutine-inspector

test: build
	go test -v -race -count=5 ./...
