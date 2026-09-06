FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY lib/ ./lib/
COPY lab-1/ ./lab-1/

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./lab-1/cmd/api

FROM alpine:3.21

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /app/server /app/server

USER app

EXPOSE 8080

CMD ["/app/server"]
