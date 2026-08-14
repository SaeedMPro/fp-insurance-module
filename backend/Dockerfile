# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# Cache deps: copy go.mod/go.sum first so `go mod download` is only re-run
# when dependencies actually change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/api ./cmd/api

# ---- final stage ----
FROM alpine:3.20 AS final

RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=build /app/api ./
COPY --from=build /src/db ./db

USER app

EXPOSE 8080

CMD ["/app/api"]
