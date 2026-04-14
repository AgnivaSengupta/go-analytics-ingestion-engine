# Stage 1: Build
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY infra/migrations ./infra/migrations
COPY sdk ./sdk

# Build API
RUN go build -o /bin/api ./cmd/api
# Build Worker
RUN go build -o /bin/worker ./cmd/worker
# Build cron service
RUN go build -o /bin/cron  ./cmd/cron
# Build migration runner
RUN go build -o /bin/migrate ./cmd/migrate
# Build aggregate backfill runner
RUN go build -o /bin/backfill ./cmd/backfill

# Stage 2: Final API Image
FROM alpine:latest AS api
WORKDIR /root/
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/api .
COPY --from=builder /app/sdk ./sdk
EXPOSE 8080
CMD ["./api"]

# Stage 3: Final Worker Image
FROM alpine:latest AS worker
WORKDIR /root/
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/worker .
CMD ["./worker"]

# Final migration image
FROM alpine:latest AS migrate
WORKDIR /root/
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/migrate .
COPY --from=builder /app/infra/migrations ./infra/migrations
CMD ["./migrate"]

# Final CRON image
FROM alpine:latest AS cron
WORKDIR /root/
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/cron .
CMD ["./cron"]

# Final aggregate backfill image
FROM alpine:latest AS backfill
WORKDIR /root/
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/backfill .
CMD ["./backfill"]
