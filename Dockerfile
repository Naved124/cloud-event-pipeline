#Stage 1: building the binary
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

#stage 2: minimal runtime environment
FROM alpine:3.24

WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080

CMD ["./server"]
