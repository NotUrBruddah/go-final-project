FROM golang:1.22 AS builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./

COPY . .

RUN go mod download

RUN go build -o app ./cmd/main.go

FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/app /app/

COPY --from=builder /app/app .

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN chown -R appuser:appgroup /app

COPY web /app/web
COPY .env /app

USER appuser

CMD ["/app/app"]
