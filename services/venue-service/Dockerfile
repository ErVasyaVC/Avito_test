FROM golang:1.23-alpine AS base
WORKDIR /app

FROM base AS dev
RUN go install github.com/air-verse/air@v1.61.7
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
CMD ["air", "-c", ".air.toml"]

FROM base AS builder
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/bin/venue-service ./cmd/venue-service

FROM alpine:3.20 AS production
RUN apk add --no-cache ca-certificates \
    && addgroup -S app && adduser -S app -G app
COPY --from=builder /app/bin/venue-service /venue-service
USER app
EXPOSE 8081
ENTRYPOINT ["/venue-service"]
