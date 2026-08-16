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
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/createadmin ./cmd/createadmin

# ---- final stage ----
FROM alpine:3.20 AS final

RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=build /app/api /app/createadmin ./
COPY --from=build /src/db ./db

# Uploaded claim documents live here. Created up front and owned by the app
# user because the process runs unprivileged; mount a volume over it in
# compose so uploads survive the container being replaced.
RUN mkdir -p /app/data/attachments && chown -R app:app /app/data
ENV ATTACHMENTS_DIR=/app/data/attachments
VOLUME ["/app/data/attachments"]

USER app

EXPOSE 8080

CMD ["/app/api"]
