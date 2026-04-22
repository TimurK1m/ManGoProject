FROM golang:1.25.6 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

# 👇 КЛЮЧЕВАЯ СТРОКА
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/main.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]