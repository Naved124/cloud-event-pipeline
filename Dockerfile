# Stage 1: building the binary
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Stage 2: minimal runtime environment
FROM alpine:3.24

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /app/server .
RUN chown appuser:appgroup /app/server

USER appuser
EXPOSE 8080

CMD ["./server"]