.PHONY: build run test vet fmt clean

build:
	go build -o bin/server ./cmd/server
	go build -o bin/gotask ./cmd/gotask

run:
	go run ./cmd/server

test:
	go test -v ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/ *.db
