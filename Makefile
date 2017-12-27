build:
	go build ./

test: build
	go test -v -race ./...
