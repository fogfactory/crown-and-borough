.PHONY: build run test vet clean web-dev

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

web-dev:
	cd web && npm run dev
