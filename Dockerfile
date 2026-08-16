# Frontend build stage
FROM node:22-alpine AS frontend

WORKDIR /web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./

ARG VITE_FIREBASE_API_KEY
ARG VITE_FIREBASE_AUTH_DOMAIN
ARG VITE_FIREBASE_PROJECT_ID
ARG VITE_FIREBASE_APP_ID
ARG VITE_FIREBASE_AUTH_EMULATOR_HOST
ARG VITE_FIREBASE_FIRESTORE_EMULATOR_HOST

ENV VITE_FIREBASE_API_KEY="$VITE_FIREBASE_API_KEY" \
    VITE_FIREBASE_AUTH_DOMAIN="$VITE_FIREBASE_AUTH_DOMAIN" \
    VITE_FIREBASE_PROJECT_ID="$VITE_FIREBASE_PROJECT_ID" \
    VITE_FIREBASE_APP_ID="$VITE_FIREBASE_APP_ID" \
    VITE_FIREBASE_AUTH_EMULATOR_HOST="$VITE_FIREBASE_AUTH_EMULATOR_HOST" \
    VITE_FIREBASE_FIRESTORE_EMULATOR_HOST="$VITE_FIREBASE_FIRESTORE_EMULATOR_HOST"

RUN npm run build

# Go build stage
FROM golang:1.26-alpine AS build

WORKDIR /app

RUN apk add --no-cache ca-certificates

# Cache Go module downloads
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# The frontend is embedded by web/embed.go during this build.
COPY --from=frontend /web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Final stage: scratch. Firebase Admin and Firestore use outbound TLS, so keep
# the CA bundle without adding a shell or package manager to the image.
FROM scratch

COPY --from=build /out/server /server
COPY --from=build /app/assets /assets
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENV ASSETS_DIR=/assets

EXPOSE 8080

ENTRYPOINT ["/server"]
