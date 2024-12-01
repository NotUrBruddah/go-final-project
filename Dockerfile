FROM golang:1.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./

COPY . .

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/main.go

FROM ubuntu:latest
WORKDIR /app
COPY --from=builder /app/app .

COPY web /app/web
COPY .env /app

CMD ["/app/app"] 