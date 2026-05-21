FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mango ./cmd/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/mango .
EXPOSE 8080
CMD ["./mango"]
