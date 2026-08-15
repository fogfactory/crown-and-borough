.PHONY: build run run-dev test test-firestore compose-up compose-up-frontend compose-down compose-logs vet clean web-deps web-dev

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

run-dev: build
	ONLINE_DEV_MODE=true ./bin/server

test:
	go test ./...

test-firestore:
	@test -n "$(FIRESTORE_EMULATOR_HOST)"
	go test -count=1 -tags=integration ./internal/store/firestore/...
	cd web && npm run test:rules

compose-up:
	docker compose up -d --build firestore server

compose-up-frontend:
	docker compose --profile frontend up -d --build

compose-down:
	docker compose --profile frontend down --remove-orphans

compose-logs:
	docker compose --profile frontend logs -f firestore server web

vet:
	go vet ./...

clean:
	rm -rf bin

web-deps:
	@test -d web/node_modules || (cd web && npm ci)

web-dev: web-deps
	cd web && npm run dev
