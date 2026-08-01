# Build stage
FROM golang:1.26-alpine AS build

WORKDIR /app

# Cache Go module downloads
COPY go.mod ./
RUN go mod download

COPY . .

# Frontend build step (web/) will be inserted here in tasks P0.3/P3.5.
# It must produce the static assets that the final stage serves.

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Final stage: scratch (static binary, no outbound TLS at MVP -> no CA certs needed)
FROM scratch

COPY --from=build /out/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
