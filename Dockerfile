FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-s -w -extldflags '-static'" \
    -o /gotask-server ./cmd/server

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-s -w -extldflags '-static'" \
    -o /gotask-cli ./cmd/gotask

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget && \
    addgroup -S gotask && \
    adduser -S -G gotask gotask

WORKDIR /app

COPY --from=builder /gotask-server /usr/local/bin/gotask-server
COPY --from=builder /gotask-cli /usr/local/bin/gotask

RUN mkdir -p /var/lib/gotask && chown -R gotask:gotask /var/lib/gotask && \
    mkdir -p /app && chown -R gotask:gotask /app

USER gotask

ENV GOTASK_DB_PATH=/var/lib/gotask/gotask.db

VOLUME ["/var/lib/gotask"]

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/gotask-server"]
