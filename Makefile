IMAGE ?= crown-and-borough:local
IMAGE_CONTAINER ?= crown-and-borough-local
COMPOSE_PROJECT_NAME ?= crown-and-borough
SERVER_PORT ?= 8080
FIRESTORE_PORT ?= 8081
AUTH_PORT ?= 9099
EMULATOR_UI_PORT ?= 4000
SMOKE_PORT ?= 18080
APP_VERSION ?= dev

.PHONY: build build-hotseat run run-dev run-hotseat run-online test test-firestore compose-up compose-up-frontend compose-down compose-logs vet clean web-deps web-build web-build-hotseat web-dev image image-run image-stop image-smoke check-web-env

build: web-build
	go build -ldflags="-X main.version=$(APP_VERSION)" -o bin/server ./cmd/server

run: build
	./bin/server

run-dev: run-hotseat

build-hotseat: web-build-hotseat
	go build -ldflags="-X main.version=$(APP_VERSION)" -o bin/server ./cmd/server

run-hotseat: build-hotseat
	ONLINE_DEV_MODE=true ./bin/server

run-online: check-web-env web-build
	@SERVER_PORT="$(SERVER_PORT)" FIRESTORE_PORT="$(FIRESTORE_PORT)" AUTH_PORT="$(AUTH_PORT)" EMULATOR_UI_PORT="$(EMULATOR_UI_PORT)" \
		docker compose -p "$(COMPOSE_PROJECT_NAME)" up -d --build firestore auth
	@trap 'docker compose -p "$(COMPOSE_PROJECT_NAME)" down --remove-orphans' EXIT INT TERM; \
		ASSETS_DIR=assets \
		FIRESTORE_EMULATOR_HOST=127.0.0.1:$(FIRESTORE_PORT) \
		FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:$(AUTH_PORT) \
		FIREBASE_PROJECT_ID=demo-crown-and-borough \
		GOOGLE_CLOUD_PROJECT=crown-and-borough-local \
		ALLOWED_CREATOR_EMAILS=admin@mail.com \
		APP_VERSION="$(APP_VERSION)" \
		ONLINE_DEV_MODE=false \
		PUBLIC_APP_URL=http://localhost:$(SERVER_PORT) \
		PORT=$(SERVER_PORT) \
		go run ./cmd/server

test: web-build
	go test ./...

test-firestore:
	@test -n "$(FIRESTORE_EMULATOR_HOST)"
	go test -count=1 -tags=integration ./internal/store/firestore/...
	cd web && npm run test:rules

compose-up: check-web-env
	@set -a; . ./web/.env.local; set +a; \
		PUBLIC_APP_URL="$${PUBLIC_APP_URL:-http://localhost:$(SERVER_PORT)}" \
		APP_VERSION="$(APP_VERSION)" \
		SERVER_PORT="$(SERVER_PORT)" FIRESTORE_PORT="$(FIRESTORE_PORT)" AUTH_PORT="$(AUTH_PORT)" EMULATOR_UI_PORT="$(EMULATOR_UI_PORT)" \
		docker compose -p "$(COMPOSE_PROJECT_NAME)" up -d --build firestore auth server

compose-up-frontend:
	@$(MAKE) compose-up

compose-down:
	docker compose -p "$(COMPOSE_PROJECT_NAME)" down --remove-orphans

compose-logs:
	docker compose -p "$(COMPOSE_PROJECT_NAME)" logs -f firestore auth server

vet: web-build
	go vet ./...

clean:
	rm -rf bin

web-deps:
	@test -d web/node_modules || (cd web && npm ci)

check-web-env:
	@test -f web/.env.local || { printf '%s\n' 'error: copy web/.env.example to web/.env.local first'; exit 1; }

web-build: web-deps
	cd web && VITE_APP_VERSION="$(APP_VERSION)" npm run build

web-build-hotseat: web-deps
	cd web && \
		VITE_APP_VERSION="$(APP_VERSION)" \
		VITE_FIREBASE_API_KEY= \
		VITE_FIREBASE_AUTH_DOMAIN= \
		VITE_FIREBASE_PROJECT_ID= \
		VITE_FIREBASE_APP_ID= \
		VITE_FIREBASE_AUTH_EMULATOR_HOST= \
		VITE_FIREBASE_FIRESTORE_EMULATOR_HOST= \
		npm run build

web-dev: web-deps
	cd web && npm run dev

image:
	@set -a; [ ! -f web/.env.local ] || . ./web/.env.local; set +a; \
		docker build -t "$(IMAGE)" \
			--build-arg APP_VERSION="$(APP_VERSION)" \
			--build-arg VITE_APP_VERSION="$(APP_VERSION)" \
			--build-arg VITE_FIREBASE_API_KEY="$${VITE_FIREBASE_API_KEY:-}" \
			--build-arg VITE_FIREBASE_AUTH_DOMAIN="$${VITE_FIREBASE_AUTH_DOMAIN:-}" \
			--build-arg VITE_FIREBASE_PROJECT_ID="$${VITE_FIREBASE_PROJECT_ID:-}" \
			--build-arg VITE_FIREBASE_APP_ID="$${VITE_FIREBASE_APP_ID:-}" \
			--build-arg VITE_FIREBASE_AUTH_EMULATOR_HOST="$${VITE_FIREBASE_AUTH_EMULATOR_HOST:-}" \
			--build-arg VITE_FIREBASE_FIRESTORE_EMULATOR_HOST="$${VITE_FIREBASE_FIRESTORE_EMULATOR_HOST:-}" \
			.

image-run: image
	docker run --rm --name "$(IMAGE_CONTAINER)" -p "$(SERVER_PORT):8080" \
		-e ASSETS_DIR=/assets \
		-e ONLINE_DEV_MODE=true \
		"$(IMAGE)"

image-stop:
	-docker stop "$(IMAGE_CONTAINER)"

image-smoke: image
	@docker rm -f crown-and-borough-smoke >/dev/null 2>&1 || true
	@docker run -d --name crown-and-borough-smoke -p "$(SMOKE_PORT):8080" \
		-e ASSETS_DIR=/assets \
		-e ONLINE_DEV_MODE=true \
		"$(IMAGE)" >/dev/null
	@tmpdir=$$(mktemp -d); \
		trap 'docker rm -f crown-and-borough-smoke >/dev/null 2>&1 || true; rm -rf "$$tmpdir"' EXIT; \
		for attempt in $$(seq 1 30); do \
			if curl -fsS "http://127.0.0.1:$(SMOKE_PORT)/healthz" >/dev/null 2>&1; then break; fi; \
			if [ "$$attempt" = 30 ]; then exit 1; fi; \
			sleep 1; \
		done; \
		curl -fsS "http://127.0.0.1:$(SMOKE_PORT)/" > "$$tmpdir/index.html"; \
		curl -fsS "http://127.0.0.1:$(SMOKE_PORT)/games/smoke" > "$$tmpdir/route.html"; \
		grep -Fq '<div id="root"></div>' "$$tmpdir/index.html"; \
		grep -Fq '<div id="root"></div>' "$$tmpdir/route.html"
