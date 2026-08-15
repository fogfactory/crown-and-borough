.PHONY: build run run-dev test vet clean web-deps web-dev

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

run-dev: build
	ONLINE_DEV_MODE=true ./bin/server

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin

web-deps:
	@test -d web/node_modules || (cd web && npm ci)

web-dev: web-deps
	cd web && npm run dev
