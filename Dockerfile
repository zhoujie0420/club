FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY server/go.mod server/go.sum* ./server/
WORKDIR /src/server
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/club-api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -u 10001 app
USER app
COPY --from=builder /out/club-api /club-api
EXPOSE 8080
ENTRYPOINT ["/club-api"]
