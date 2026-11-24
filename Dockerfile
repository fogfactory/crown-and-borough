# Frontend build stage
FROM node:26-alpine AS frontend

WORKDIR /web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./

ARG VITE_FIREBASE_PROJECT_ID
ARG VITE_FIREBASE_APP_ID
ARG VITE_FIREBASE_FIRESTORE_EMULATOR_HOST
ARG APP_VERSION=dev
ARG VITE_APP_VERSION

ENV VITE_FIREBASE_PROJECT_ID="$VITE_FIREBASE_PROJECT_ID" \
    VITE_FIREBASE_APP_ID="$VITE_FIREBASE_APP_ID" \
    VITE_FIREBASE_FIRESTORE_EMULATOR_HOST="$VITE_FIREBASE_FIRESTORE_EMULATOR_HOST"

# The three values Docker's secret lint flags (API key, auth domain, auth
# emulator host) come in as BuildKit secrets, not ARG/ENV, so that check stays
# active for any real secret added later. A missing secret gives an empty
# value, which builds the hotseat bundle.

# dist/.gitkeep only keeps go:embed compiling on a fresh clone; do not ship it.
RUN --mount=type=secret,id=firebase_api_key,required=false \
    --mount=type=secret,id=firebase_auth_domain,required=false \
    --mount=type=secret,id=firebase_auth_emulator_host,required=false \
    read_secret() { if [ -f "/run/secrets/$1" ]; then cat "/run/secrets/$1"; fi; } && \
    export VITE_FIREBASE_API_KEY="$(read_secret firebase_api_key)" \
           VITE_FIREBASE_AUTH_DOMAIN="$(read_secret firebase_auth_domain)" \
           VITE_FIREBASE_AUTH_EMULATOR_HOST="$(read_secret firebase_auth_emulator_host)" \
           VITE_APP_VERSION="${VITE_APP_VERSION:-$APP_VERSION}" && \
    npm run build &&rm -f dist/.gitkeep

# Go build stage
FROM golang:1.27-alpine AS build

WORKDIR /app

RUN apk add --no-cache ca-certificates

# Cache Go module downloads
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# The frontend is embedded by web/embed.go during this build.
COPY --from=frontend /web/dist ./web/dist

ARG APP_VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=$APP_VERSION" -o /out/server ./cmd/server

# Final stage: scratch. Firebase Admin and Firestore use outbound TLS, so keep
# the CA bundle without adding a shell or package manager to the image.
FROM scratch

COPY --from=build /out/server /server
COPY --from=build /app/assets /assets
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENV ASSETS_DIR=/assets

EXPOSE 8080

ENTRYPOINT ["/server"]
