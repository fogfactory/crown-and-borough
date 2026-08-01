.PHONY: build run test vet clean

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin
