.PHONY: test build lint install

test:
	go test ./...

build:
	go build -o ntzh ./cmd/ntzh

lint:
	golangci-lint run

install:
	go install ./cmd/ntzh
