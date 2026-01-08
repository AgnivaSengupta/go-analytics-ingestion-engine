# Stage 1: Build
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build API
RUN go build -o /bin/api ./cmd/api
# Build Worker
RUN go build -o /bin/worker ./cmd/worker
# Build cron service
RUN go build -o /bin/cron  ./cmd/cron

# Stage 2: Final API Image
FROM alpine:latest AS api
WORKDIR /root/
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/api .
EXPOSE 8080
CMD ["./api"]

# Stage 3: Final Worker Image
FROM alpine:latest AS worker
WORKDIR /root/
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/worker .
CMD ["./worker"]

# Final CRON image
FROM alpine:latest as cron
WORKDIR /root/
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/cron .
CMD ["./cron"]